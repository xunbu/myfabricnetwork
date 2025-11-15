package gateway

import (
	"fmt"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"
)

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
