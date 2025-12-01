package gateway

import (
	"bytes"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"
)

// ParseTxInfo 是核心通用函数，用于将 Envelope 字节解析为标准化的 Txinfo 结构
// 被 GetTxByID, GetBlockTransactionsByPage 复用
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

	// 仅针对“背书交易”解析具体的 Chaincode 参数
	if common.HeaderType(chHeader.Type) == common.HeaderType_ENDORSER_TRANSACTION {
		if err := parseChaincodeArgs(payload.Data, txInfo); err != nil {
			// 解析参数非关键路径，可选择忽略错误或记录日志
			// fmt.Printf("Tx %s 参数解析警告: %v\n", txInfo.TxId, err)
		}
	}

	return txInfo, nil
}

// parseChaincodeArgs 辅助函数：深度解析 Chaincode 参数
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

// parseBlockInfo 解析区块头及元数据
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

	info := &BlockInfo{
		BlockHash:    fmt.Sprintf("%x", blockHash),
		PreviousHash: fmt.Sprintf("%x", blockHeader.PreviousHash),
		MerkleRoot:   fmt.Sprintf("%x", blockHeader.DataHash),
		BlockNumber:  blockNumber,
		TxCount:      uint64(len(block.Data.Data)),
		BlockSize:    int64(len(blockBytes)),
		ChannelID:    channelName,
	}

	// 遍历交易以提取时间戳、创建者和TxIDs
	// 我们只解析必要的信息，不进行全量 Chaincode 参数解析以节省性能
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
			info.BlockCreator = getCreatorFromEnvelope(data)
		}

		if includeTxDetails {
			txIDs = append(txIDs, txEnv.TxId)
		}
	}

	// 兜底时间戳（如果是创世块或者解析失败）
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

// simpleTxHeader 内部结构，用于 parseBlockInfo 快速提取信息
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

// getCreatorFromEnvelope 辅助函数：从 SignatureHeader 中提取创建者摘要
func getCreatorFromEnvelope(envelopeBytes []byte) string {
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

	if len(sigHeader.Creator) > 0 {
		return fmt.Sprintf("Creator:%x", sigHeader.Creator[:min(8, len(sigHeader.Creator))])
	}
	return ""
}

// CalculateBlockHash 计算区块哈希
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

func FormatJSON(data []byte) (string, error) {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		return "", fmt.Errorf("格式化JSON失败: %w", err)
	}
	return prettyJSON.String(), nil
}

func convertBytesToStrings(byteSlices [][]byte) []string {
	strSlice := make([]string, len(byteSlices))
	for i, b := range byteSlices {
		strSlice[i] = string(b)
	}
	return strSlice
}
