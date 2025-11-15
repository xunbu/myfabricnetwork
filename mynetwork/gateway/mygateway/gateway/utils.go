package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"
)

func parseBlockInfo(blockBytes []byte, blockNumber uint64, channelName string, includeTxDetails bool) (*BlockInfo, error) {
	var block common.Block
	if err := proto.Unmarshal(blockBytes, &block); err != nil {
		return nil, fmt.Errorf("解析区块失败: %w", err)
	}

	blockHeader := block.Header
	if blockHeader == nil {
		return nil, fmt.Errorf("区块头信息为空")
	}

	// 解析区块数据大小
	blockSize := int64(len(blockBytes))

	// 从第一个交易中获取时间戳和创建者信息
	timestamp, blockCreator, txIDs := extractBlockMetadata(block.Data.Data, includeTxDetails)

	// 对于创世区块（区块0），设置默认时间戳
	if blockNumber == 0 && timestamp.IsZero() {
		timestamp = time.Now().AddDate(-1, 0, 0) // 设置为一年前
	}

	return &BlockInfo{
		BlockHash:    fmt.Sprintf("%x", blockHeader.DataHash),
		PreviousHash: fmt.Sprintf("%x", blockHeader.PreviousHash),
		MerkleRoot:   fmt.Sprintf("%x", blockHeader.DataHash),
		BlockNumber:  blockNumber,
		TxCount:      uint64(len(block.Data.Data)),
		BlockSize:    blockSize,
		Timestamp:    timestamp,
		ChannelID:    channelName,
		BlockCreator: blockCreator,
		TxIDs:        txIDs,
	}, nil
}

// 从区块交易中提取元数据
func extractBlockMetadata(blockData [][]byte, includeTxDetails bool) (time.Time, string, []string) {
	if len(blockData) == 0 {
		return time.Time{}, "", nil
	}

	var timestamp time.Time
	var blockCreator string
	var txIDs []string

	// 遍历所有交易，获取时间戳和创建者信息
	for i, data := range blockData {
		var envelope common.Envelope
		if err := proto.Unmarshal(data, &envelope); err != nil {
			continue
		}

		var payload common.Payload
		if err := proto.Unmarshal(envelope.Payload, &payload); err != nil {
			continue
		}

		var channelHeader common.ChannelHeader
		if err := proto.Unmarshal(payload.Header.ChannelHeader, &channelHeader); err != nil {
			continue
		}

		// 使用第一个有效的交易时间戳作为区块时间戳
		if timestamp.IsZero() {
			timestamp = time.Unix(channelHeader.Timestamp.Seconds, int64(channelHeader.Timestamp.Nanos))
		}

		// 从第一个交易获取创建者信息
		if i == 0 && blockCreator == "" {
			blockCreator = extractCreatorFromSignature(payload.Header.SignatureHeader)
		}

		// 如果需要包含交易详情，收集交易ID
		if includeTxDetails {
			txIDs = append(txIDs, channelHeader.TxId)
		}
	}

	return timestamp, blockCreator, txIDs
}

// 从签名头中提取创建者信息
func extractCreatorFromSignature(signatureHeaderBytes []byte) string {
	if len(signatureHeaderBytes) == 0 {
		return ""
	}

	var signatureHeader common.SignatureHeader
	if err := proto.Unmarshal(signatureHeaderBytes, &signatureHeader); err != nil {
		return ""
	}

	if len(signatureHeader.Creator) > 0 {
		// 简化显示：返回创建者信息的简短哈希
		return fmt.Sprintf("Creator:%x", signatureHeader.Creator[:min(8, len(signatureHeader.Creator))])
	}

	return ""
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

			// 解析读写集
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

		// 解析K-V读写集
		kvRwSet := &kvrwset.KVRWSet{}
		if err := proto.Unmarshal(nsRwSet.Rwset, kvRwSet); err == nil {
			// 读取集
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

			// 写入集
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

			// 范围查询
			if len(kvRwSet.RangeQueriesInfo) > 0 {
				fmt.Printf("  范围查询 (%d 个):\n", len(kvRwSet.RangeQueriesInfo))
				for j, rangeQuery := range kvRwSet.RangeQueriesInfo {
					fmt.Printf("    [%d] StartKey: %s, EndKey: %s\n",
						j+1, rangeQuery.StartKey, rangeQuery.EndKey)
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
