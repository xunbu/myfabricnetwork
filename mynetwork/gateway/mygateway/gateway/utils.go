package gateway

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/msp"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"
)

// --- 核心解析逻辑 ---

// ParseTxInfo 将 Envelope 字节解析为标准化的 Txinfo 结构
func ParseTxInfo(envelopeBytes []byte, validationCode int) (*Txinfo, error) {
	txInfo := &Txinfo{
		Size:           len(envelopeBytes),
		ValidationCode: validationCode,
	}

	envelope := &common.Envelope{}
	if err := proto.Unmarshal(envelopeBytes, envelope); err != nil {
		return nil, fmt.Errorf("解析Envelope失败: %w", err)
	}

	payload := &common.Payload{}
	if err := proto.Unmarshal(envelope.Payload, payload); err != nil {
		return nil, fmt.Errorf("解析Payload失败: %w", err)
	}

	chHeader := &common.ChannelHeader{}
	if err := proto.Unmarshal(payload.Header.ChannelHeader, chHeader); err != nil {
		return nil, fmt.Errorf("解析ChannelHeader失败: %w", err)
	}

	txInfo.TxId = chHeader.TxId
	txInfo.ChannelId = chHeader.ChannelId
	txInfo.Type = common.HeaderType_name[chHeader.Type]
	txInfo.Timestamp = chHeader.Timestamp.AsTime()

	// 解析交易发起者的 MSP 和 域名
	mspID, domain := parseCreatorIdentity(payload.Header.SignatureHeader)
	txInfo.CreatorMSP = mspID
	txInfo.CreatorDomain = domain

	// 仅针对“背书交易”解析具体的 Chaincode 参数
	if common.HeaderType(chHeader.Type) == common.HeaderType_ENDORSER_TRANSACTION {
		if err := parseChaincodeArgs(payload.Data, txInfo); err != nil {
			// 参数解析失败不影响主体信息返回
			// fmt.Printf("解析链码参数警告: %v\n", err)
		}
	}

	return txInfo, nil
}

// parseBlockInfo 解析区块头及元数据，提取 Orderer 域名作为 Creator
func parseBlockInfo(blockBytes []byte, blockNumber uint64, channelName string, includeTxDetails bool) (*BlockInfo, error) {
	var block common.Block
	if err := proto.Unmarshal(blockBytes, &block); err != nil {
		return nil, fmt.Errorf("解析区块失败: %w", err)
	}

	blockHeader := block.Header
	if blockHeader == nil {
		return nil, fmt.Errorf("区块头信息为空")
	}

	blockHash, err := CalculateBlockHash(blockHeader)
	if err != nil {
		return nil, err
	}

	// 1. 尝试从元数据获取区块创建者 (通常是 Orderer 的域名)
	creatorDomain := getBlockCreatorFromMetadata(&block)

	info := &BlockInfo{
		BlockHash:    fmt.Sprintf("%x", blockHash),
		PreviousHash: fmt.Sprintf("%x", blockHeader.PreviousHash),
		MerkleRoot:   fmt.Sprintf("%x", blockHeader.DataHash),
		BlockNumber:  blockNumber,
		TxCount:      uint64(len(block.Data.Data)),
		BlockSize:    int64(len(blockBytes)),
		ChannelID:    channelName,
		BlockCreator: creatorDomain, // 存储域名，例如 orderer.example.com
	}

	var txIDs []string
	for i, data := range block.Data.Data {
		// 快速解析 Header 获取时间戳和 TxID
		txEnv, err := fastParseHeader(data)
		if err != nil {
			continue
		}

		// 使用第一个交易的时间戳作为区块时间戳
		if i == 0 {
			info.Timestamp = txEnv.Timestamp

			// 兜底逻辑：如果元数据中没有 Orderer 签名（罕见），
			// 则使用区块中第一个交易的发送者域名作为 Creator
			if info.BlockCreator == "" {
				info.BlockCreator = getCreatorDomainFromEnvelope(data)
			}
		}

		if includeTxDetails {
			txIDs = append(txIDs, txEnv.TxId)
		}
	}

	// 兜底时间戳处理 (防止空区块导致时间为零值)
	if info.Timestamp.IsZero() {
		if blockNumber == 0 {
			info.Timestamp = time.Now().AddDate(-1, 0, 0)
		} else {
			info.Timestamp = time.Now()
		}
	}

	info.TxIDs = txIDs
	return info, nil
}

// --- 身份与证书解析辅助函数 ---

// getCommonNameFromIdentity 从 SerializedIdentity 字节中解析 X.509 证书的 Common Name (CN)
func getCommonNameFromIdentity(idBytes []byte) string {
	if len(idBytes) == 0 {
		return ""
	}

	// 解码 PEM 块
	block, _ := pem.Decode(idBytes)
	if block == nil {
		return ""
	}

	// 解析 X.509 证书
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return ""
	}

	// 返回 Common Name (通常是域名，如 orderer.example.com 或 peer0.org1.example.com)
	return cert.Subject.CommonName
}

// getBlockCreatorFromMetadata 从区块元数据中提取 Orderer 的域名
func getBlockCreatorFromMetadata(block *common.Block) string {
	if block.Metadata == nil || len(block.Metadata.Metadata) <= int(common.BlockMetadataIndex_SIGNATURES) {
		return ""
	}

	// 获取签名元数据
	metadataEntry := &common.Metadata{}
	if err := proto.Unmarshal(block.Metadata.Metadata[common.BlockMetadataIndex_SIGNATURES], metadataEntry); err != nil {
		return ""
	}

	if len(metadataEntry.Signatures) == 0 {
		return ""
	}

	// 解析 SignatureHeader
	sh := &common.SignatureHeader{}
	if err := proto.Unmarshal(metadataEntry.Signatures[0].SignatureHeader, sh); err != nil {
		return ""
	}

	// 解析 SerializedIdentity
	sId := &msp.SerializedIdentity{}
	if err := proto.Unmarshal(sh.Creator, sId); err != nil {
		return ""
	}

	// 提取 CN
	return getCommonNameFromIdentity(sId.IdBytes)
}

// getCreatorDomainFromEnvelope 从交易 Envelope 中提取发送者的域名 (兜底用)
func getCreatorDomainFromEnvelope(envelopeBytes []byte) string {
	env := &common.Envelope{}
	if err := proto.Unmarshal(envelopeBytes, env); err != nil {
		return ""
	}
	pay := &common.Payload{}
	if err := proto.Unmarshal(env.Payload, pay); err != nil {
		return ""
	}
	sigHeader := &common.SignatureHeader{}
	if err := proto.Unmarshal(pay.Header.SignatureHeader, sigHeader); err != nil {
		return ""
	}

	identity := &msp.SerializedIdentity{}
	if err := proto.Unmarshal(sigHeader.Creator, identity); err != nil {
		return ""
	}

	return getCommonNameFromIdentity(identity.IdBytes)
}

// parseCreatorIdentity 解析签名头中的身份信息，返回 (MSPID, Domain/CN)
func parseCreatorIdentity(signatureHeaderBytes []byte) (string, string) {
	if len(signatureHeaderBytes) == 0 {
		return "", ""
	}

	sigHeader := &common.SignatureHeader{}
	if err := proto.Unmarshal(signatureHeaderBytes, sigHeader); err != nil {
		return "", ""
	}

	identity := &msp.SerializedIdentity{}
	if err := proto.Unmarshal(sigHeader.Creator, identity); err != nil {
		return "", ""
	}

	mspID := identity.Mspid

	// 使用统一的函数获取域名
	cn := getCommonNameFromIdentity(identity.IdBytes)

	// 这里保留之前的逻辑：如果是 User1@org1.example.com，可能只想取 @ 之后的部分作为 Domain
	// 如果需要完整 CN (如 peer0.org1.example.com)，直接返回 cn 即可
	domain := cn
	if strings.Contains(cn, "@") {
		parts := strings.Split(cn, "@")
		if len(parts) > 1 {
			domain = parts[1]
		}
	}

	return mspID, domain
}

// --- 其他通用辅助函数 ---

// CalculateBlockHash 计算区块哈希 (ASN.1 编码头部的 SHA256)
func CalculateBlockHash(header *common.BlockHeader) ([]byte, error) {
	type asn1Header struct {
		Number       int64
		PreviousHash []byte
		DataHash     []byte
	}
	asn1HeaderData := asn1Header{
		Number:       int64(header.Number),
		PreviousHash: header.PreviousHash,
		DataHash:     header.DataHash,
	}
	result, err := asn1.Marshal(asn1HeaderData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal block header for hashing: %w", err)
	}
	hash := sha256.Sum256(result)
	return hash[:], nil
}

// fastParseHeader 快速解析 Envelope 获取 TxId 和 Timestamp
type simpleTxHeader struct {
	TxId      string
	Timestamp time.Time
}

func fastParseHeader(envelopeBytes []byte) (*simpleTxHeader, error) {
	envelope := &common.Envelope{}
	if err := proto.Unmarshal(envelopeBytes, envelope); err != nil {
		return nil, err
	}
	payload := &common.Payload{}
	if err := proto.Unmarshal(envelope.Payload, payload); err != nil {
		return nil, err
	}
	chHeader := &common.ChannelHeader{}
	if err := proto.Unmarshal(payload.Header.ChannelHeader, chHeader); err != nil {
		return nil, err
	}
	return &simpleTxHeader{
		TxId:      chHeader.TxId,
		Timestamp: chHeader.Timestamp.AsTime(),
	}, nil
}

// parseChaincodeArgs 深度解析 Chaincode 参数
func parseChaincodeArgs(dataBytes []byte, txInfo *Txinfo) error {
	transaction := &peer.Transaction{}
	if err := proto.Unmarshal(dataBytes, transaction); err != nil {
		return err
	}

	for _, action := range transaction.Actions {
		cap := &peer.ChaincodeActionPayload{}
		if err := proto.Unmarshal(action.Payload, cap); err != nil {
			continue
		}

		cpp := &peer.ChaincodeProposalPayload{}
		if err := proto.Unmarshal(cap.ChaincodeProposalPayload, cpp); err != nil {
			continue
		}

		cis := &peer.ChaincodeInvocationSpec{}
		if err := proto.Unmarshal(cpp.Input, cis); err != nil {
			continue
		}

		if cis.ChaincodeSpec != nil {
			txInfo.ChainCodeInfos = append(txInfo.ChainCodeInfos, &ChainCodeInfo{
				ChainCodeName: cis.ChaincodeSpec.ChaincodeId.Name,
				Args:          convertBytesToStrings(cis.ChaincodeSpec.Input.Args),
			})
		}
	}
	return nil
}

// FormatJSON 格式化 JSON 字符串
func FormatJSON(data []byte) (string, error) {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		return "", fmt.Errorf("格式化JSON失败: %w", err)
	}
	return prettyJSON.String(), nil
}

// convertBytesToStrings 将字节切片数组转换为字符串数组
func convertBytesToStrings(byteSlices [][]byte) []string {
	strSlice := make([]string, len(byteSlices))
	for i, b := range byteSlices {
		strSlice[i] = string(b)
	}
	return strSlice
}

// parseReadWriteSets (用于调试) 打印读写集
func parseReadWriteSets(tx *peer.ProcessedTransaction) {
	if tx.TransactionEnvelope == nil || tx.TransactionEnvelope.Payload == nil {
		return
	}
	payload := &common.Payload{}
	if err := proto.Unmarshal(tx.TransactionEnvelope.Payload, payload); err != nil {
		return
	}
	txData := &peer.Transaction{}
	if err := proto.Unmarshal(payload.Data, txData); err != nil {
		return
	}

	for i, action := range txData.Actions {
		cap := &peer.ChaincodeActionPayload{}
		if err := proto.Unmarshal(action.Payload, cap); err != nil {
			continue
		}
		if cap.Action != nil && cap.Action.ProposalResponsePayload != nil {
			prp := &peer.ProposalResponsePayload{}
			if err := proto.Unmarshal(cap.Action.ProposalResponsePayload, prp); err != nil {
				continue
			}
			chaincodeAction := &peer.ChaincodeAction{}
			if err := proto.Unmarshal(prp.Extension, chaincodeAction); err != nil {
				continue
			}
			if chaincodeAction.Results != nil {
				txReadWriteSet := &rwset.TxReadWriteSet{}
				if err := proto.Unmarshal(chaincodeAction.Results, txReadWriteSet); err == nil {
					fmt.Printf("\n=== 动作 %d 的读写集 ===\n", i+1)
					displayReadWriteSet(txReadWriteSet)
				}
			}
		}
	}
}

func displayReadWriteSet(rwSet *rwset.TxReadWriteSet) {
	fmt.Printf("数据模型: %s\n", rwset.TxReadWriteSet_DataModel_name[int32(rwSet.DataModel)])
	for _, nsRwSet := range rwSet.NsRwset {
		fmt.Printf("命名空间: %s\n", nsRwSet.Namespace)
		kvRwSet := &kvrwset.KVRWSet{}
		if err := proto.Unmarshal(nsRwSet.Rwset, kvRwSet); err == nil {
			if len(kvRwSet.Reads) > 0 {
				fmt.Printf("  读取集 (%d 个):\n", len(kvRwSet.Reads))
				for j, read := range kvRwSet.Reads {
					fmt.Printf("    [%d] Key: %s\n", j+1, read.Key)
					if read.Version != nil {
						fmt.Printf("        Version: BlockNum=%d, TxNum=%d\n",
							read.Version.BlockNum, read.Version.TxNum)
					}
				}
			}
			if len(kvRwSet.Writes) > 0 {
				fmt.Printf("  写入集 (%d 个):\n", len(kvRwSet.Writes))
				for j, write := range kvRwSet.Writes {
					fmt.Printf("    [%d] Key: %s, 是否删除: %v\n",
						j+1, write.Key, write.IsDelete)
					if !write.IsDelete {
						fmt.Printf("        Value: %s\n", string(write.Value))
					}
				}
			}
		}
	}
}
