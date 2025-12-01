package gateway

import (
	"fmt"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"
)

type BlockInfo struct {
	BlockHash    string    `json:"blockHash"`
	PreviousHash string    `json:"previousHash"`
	MerkleRoot   string    `json:"dataHash"`
	BlockNumber  uint64    `json:"blockNumber"`
	TxCount      uint64    `json:"txCount"`
	BlockSize    int64     `json:"blockSize"`
	Timestamp    time.Time `json:"timestamp"`
	ChannelID    string    `json:"channelId"`
	BlockCreator string    `json:"blockCreator,omitempty"`
	TxIDs        []string  `json:"txIds,omitempty"`
}

type BlockPage struct {
	Results  []*BlockInfo `json:"blockPage"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
	Total    int          `json:"total"`
}

type Txinfo struct {
	TxId           string
	ChannelId      string
	Type           string
	Timestamp      time.Time
	Size           int
	ValidationCode int
	ChainCodeInfos []*ChainCodeInfo
}

type ChainCodeInfo struct {
	ChainCodeName string
	Args          []string
}

// GetBlockHeight 通过 QSCC 查询高度
func GetBlockHeight(gw *client.Gateway, channelName string) (uint64, error) {
	chainInfoBytes, err := EvaluateTransaction(gw, channelName, "qscc", "GetChainInfo", channelName)
	if err != nil {
		return 0, fmt.Errorf("获取区块链信息失败: %w", err)
	}
	var chainInfo common.BlockchainInfo
	if err := proto.Unmarshal(chainInfoBytes, &chainInfo); err != nil {
		return 0, fmt.Errorf("解析区块链信息失败: %w", err)
	}
	return chainInfo.Height, nil
}

// GetOrganizationCount 通过 CSCC 查询组织数
func GetOrganizationCount(gw *client.Gateway, channelName string) (int, error) {
	// 复用EvaluateTransaction获取配置区块
	configBlockBytes, err := EvaluateTransaction(gw, channelName, "cscc", "GetConfigBlock", channelName)
	if err != nil {
		return 0, fmt.Errorf("获取配置区块失败: %w", err)
	}

	var configBlock common.Block
	if err := proto.Unmarshal(configBlockBytes, &configBlock); err != nil {
		return 0, fmt.Errorf("解析配置区块失败: %w", err)
	}

	if len(configBlock.Data.Data) == 0 {
		return 0, fmt.Errorf("配置区块数据为空")
	}

	var envelope common.Envelope
	if err := proto.Unmarshal(configBlock.Data.Data[0], &envelope); err != nil {
		return 0, fmt.Errorf("解析区块Envelope失败: %w", err)
	}

	var payload common.Payload
	if err := proto.Unmarshal(envelope.Payload, &payload); err != nil {
		return 0, fmt.Errorf("解析Payload失败: %w", err)
	}

	var configEnvelope common.ConfigEnvelope
	if err := proto.Unmarshal(payload.Data, &configEnvelope); err != nil {
		return 0, fmt.Errorf("解析ConfigEnvelope失败: %w", err)
	}

	channelGroup := configEnvelope.Config.ChannelGroup
	if channelGroup == nil {
		return 0, fmt.Errorf("配置中缺少ChannelGroup")
	}

	var orgsGroup *common.ConfigGroup
	if appGroup, exists := channelGroup.Groups["Application"]; exists {
		orgsGroup = appGroup
	} else if consortiumGroup, exists := channelGroup.Groups["Consortiums"]; exists {
		for _, consortium := range consortiumGroup.Groups {
			orgsGroup = consortium
			break
		}
	} else {
		return 0, fmt.Errorf("无法确定通道类型，缺少Application或Consortiums配置")
	}

	if orgsGroup == nil {
		return 0, fmt.Errorf("配置中缺少组织信息")
	}

	return len(orgsGroup.Groups), nil
}

// GetBlockListByPage 分页查询区块列表
func GetBlockListByPage(gw *client.Gateway, channelName string, pageNum uint64, pageSize uint64, includeTxDetails bool) (*BlockPage, error) {
	var blockHeight uint64
	stats := GetChainStats() // 跨文件调用 monitor.go
	if stats.Height > 0 {
		blockHeight = stats.Height
	} else {
		h, err := GetBlockHeight(gw, channelName)
		if err != nil {
			return nil, err
		}
		blockHeight = h
	}

	totalBlocks := int64(blockHeight)
	if totalBlocks == 0 {
		return nil, nil
	}
	startIdx := int64(pageNum * pageSize)
	if startIdx >= totalBlocks {
		return nil, nil
	}
	endIdx := startIdx + int64(pageSize) - 1
	if endIdx >= totalBlocks {
		endIdx = totalBlocks - 1
	}

	var blockList []*BlockInfo
	for blockNumber := endIdx; blockNumber >= startIdx; blockNumber-- {
		blockBytes, err := EvaluateTransaction(gw, channelName, "qscc", "GetBlockByNumber", channelName, fmt.Sprint(blockNumber))
		if err != nil {
			return nil, err
		}

		blockInfo, err := parseBlockInfo(blockBytes, uint64(blockNumber), channelName, includeTxDetails) // utils.go
		if err != nil {
			return nil, err
		}
		blockList = append(blockList, blockInfo)
	}

	return &BlockPage{Results: blockList, Page: int(pageNum), PageSize: int(pageSize), Total: int(totalBlocks)}, nil
}

// GetBlockByNum 根据区块号获取区块详情
func GetBlockByNum(gw *client.Gateway, channelName string, blockNum uint64) (*BlockInfo, error) {
	blockBytes, err := EvaluateTransaction(gw, channelName, "qscc", "GetBlockByNumber", channelName, fmt.Sprint(blockNum))
	if err != nil {
		return nil, err
	}
	return parseBlockInfo(blockBytes, blockNum, channelName, true) // utils.go
}

// GetTxByID 查询交易详情
func GetTxByID(gw *client.Gateway, channelName string, TxID string) (*Txinfo, error) {
	v, err := EvaluateTransaction(gw, channelName, "qscc", "GetTransactionByID", channelName, TxID)
	if err != nil {
		return nil, err
	}

	txInfo := &Txinfo{Size: len(v)}
	tx := &peer.ProcessedTransaction{}
	if err = proto.Unmarshal(v, tx); err != nil {
		return nil, err
	}

	txInfo.ValidationCode = int(tx.ValidationCode)
	if tx.TransactionEnvelope == nil {
		return txInfo, nil
	}

	payload := &common.Payload{}
	if err = proto.Unmarshal(tx.TransactionEnvelope.Payload, payload); err != nil {
		return txInfo, err
	}

	chHeader := &common.ChannelHeader{}
	if err = proto.Unmarshal(payload.Header.ChannelHeader, chHeader); err != nil {
		return txInfo, err
	}

	txInfo.TxId = chHeader.TxId
	txInfo.ChannelId = chHeader.ChannelId
	txInfo.Type = common.HeaderType_name[chHeader.Type]
	txInfo.Timestamp = chHeader.Timestamp.AsTime()

	if common.HeaderType(chHeader.Type) == common.HeaderType_ENDORSER_TRANSACTION {
		transaction := &peer.Transaction{}
		proto.Unmarshal(payload.Data, transaction)
		for _, action := range transaction.Actions {
			v := &peer.ChaincodeActionPayload{}
			proto.Unmarshal(action.Payload, v)
			v2 := &peer.ChaincodeProposalPayload{}
			proto.Unmarshal(v.ChaincodeProposalPayload, v2)
			invocationSpec := &peer.ChaincodeInvocationSpec{}
			proto.Unmarshal(v2.Input, invocationSpec)
			spec := invocationSpec.ChaincodeSpec

			txInfo.ChainCodeInfos = append(txInfo.ChainCodeInfos, &ChainCodeInfo{
				ChainCodeName: spec.ChaincodeId.Name,
				Args:          convertBytesToStrings(spec.Input.Args), // utils.go
			})
		}
	}
	return txInfo, nil
}
