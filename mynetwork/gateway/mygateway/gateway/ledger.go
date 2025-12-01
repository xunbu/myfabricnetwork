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
	TxId           string           `json:"txId"`
	ChannelId      string           `json:"channelId"`
	Type           string           `json:"type"`
	Timestamp      time.Time        `json:"timestamp"`
	Size           int              `json:"size"`
	ValidationCode int              `json:"validationCode"`
	ChainCodeInfos []*ChainCodeInfo `json:"chainCodeInfos,omitempty"`
}

type ChainCodeInfo struct {
	ChainCodeName string   `json:"chainCodeName"`
	Args          []string `json:"args"`
}

type TransactionPage struct {
	Results  []*Txinfo `json:"txPage"`
	Page     int       `json:"page"`
	PageSize int       `json:"pageSize"`
	Total    int       `json:"total"`
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
	stats := GetChainStats() // 调用 monitor.go 的缓存数据
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

		blockInfo, err := parseBlockInfo(blockBytes, uint64(blockNumber), channelName, includeTxDetails) // 调用 utils.go
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
	return parseBlockInfo(blockBytes, blockNum, channelName, true) // 调用 utils.go
}

// GetTxByID 查询交易详情 (重构后使用 utils.ParseTxInfo)
func GetTxByID(gw *client.Gateway, channelName string, TxID string) (*Txinfo, error) {
	v, err := EvaluateTransaction(gw, channelName, "qscc", "GetTransactionByID", channelName, TxID)
	if err != nil {
		return nil, err
	}

	// 1. 解析外层 ProcessedTransaction
	tx := &peer.ProcessedTransaction{}
	if err = proto.Unmarshal(v, tx); err != nil {
		return nil, fmt.Errorf("解析ProcessedTransaction失败: %w", err)
	}

	// 2. 将 Envelope 重新序列化为 bytes 以便复用 ParseTxInfo
	if tx.TransactionEnvelope == nil {
		return nil, fmt.Errorf("TransactionEnvelope为空")
	}
	envelopeBytes, err := proto.Marshal(tx.TransactionEnvelope)
	if err != nil {
		return nil, fmt.Errorf("重组Envelope失败: %w", err)
	}

	// 3. 复用 utils.go 中的解析逻辑
	return ParseTxInfo(envelopeBytes, int(tx.ValidationCode))
}

// GetBlockTransactionsByPage 根据区块号分页查询该区块内的交易列表
func GetBlockTransactionsByPage(gw *client.Gateway, channelName string, blockNum uint64, pageNum int, pageSize int) (*TransactionPage, error) {
	// 1. 获取区块数据
	blockBytes, err := EvaluateTransaction(gw, channelName, "qscc", "GetBlockByNumber", channelName, fmt.Sprint(blockNum))
	if err != nil {
		return nil, fmt.Errorf("获取区块[%d]失败: %w", blockNum, err)
	}

	block := &common.Block{}
	if err := proto.Unmarshal(blockBytes, block); err != nil {
		return nil, fmt.Errorf("解析区块数据失败: %w", err)
	}

	// 2. 准备分页数据
	totalTxs := len(block.Data.Data)

	// 获取验证码过滤器
	var txFilter []byte
	if len(block.Metadata.Metadata) > int(common.BlockMetadataIndex_TRANSACTIONS_FILTER) {
		txFilter = block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER]
	}

	if totalTxs == 0 {
		return &TransactionPage{Results: []*Txinfo{}, Page: pageNum, PageSize: pageSize, Total: 0}, nil
	}

	startIdx := pageNum * pageSize
	if startIdx >= totalTxs {
		return &TransactionPage{Results: []*Txinfo{}, Page: pageNum, PageSize: pageSize, Total: totalTxs}, nil
	}

	endIdx := startIdx + pageSize
	if endIdx > totalTxs {
		endIdx = totalTxs
	}

	// 3. 遍历并解析交易
	var txList []*Txinfo
	for i := startIdx; i < endIdx; i++ {
		envelopeBytes := block.Data.Data[i]

		// 获取验证状态
		validationCode := 0
		if len(txFilter) > i {
			validationCode = int(txFilter[i])
		}

		// 调用 utils.go 中的通用解析函数
		txInfo, err := ParseTxInfo(envelopeBytes, validationCode)
		if err != nil {
			// 解析失败跳过，避免中断整个列表
			continue
		}

		txList = append(txList, txInfo)
	}

	return &TransactionPage{
		Results:  txList,
		Page:     pageNum,
		PageSize: pageSize,
		Total:    totalTxs,
	}, nil
}
