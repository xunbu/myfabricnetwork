package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"google.golang.org/protobuf/proto"
)

// ChainStats 纯数据结构体
type ChainStats struct {
	Height        uint64    `json:"blockHeight"`           // 区块高度
	TxCount       uint64    `json:"totalTransactionCount"` // 总交易量
	OrgCount      int       `json:"orgCount"`              // 组织数量
	LastBlockTime time.Time `json:"lastBlockTime"`         // 最新区块时间
}

// safeCache 内部使用的封装结构 (包含数据 + 锁)
type safeCache struct {
	data         ChainStats   // 统计数据
	recentBlocks []*BlockInfo // 最近生成的区块缓存 (用于图表)
	mu           sync.RWMutex // 读写锁
}

const MaxRecentBlocks = 50 // 缓存最近多少个区块用于趋势图

// 全局单例缓存
var globalCache = &safeCache{
	data:         ChainStats{OrgCount: 0, TxCount: 0, Height: 0},
	recentBlocks: make([]*BlockInfo, 0, MaxRecentBlocks),
}

// StartChainMonitor 启动全量区块监听
func StartChainMonitor(gw *client.Gateway, channelName string) error {
	network := gw.GetNetwork(channelName)

	// 初始化组织数量
	initialOrgCount, err := GetOrganizationCount(gw, channelName)
	if err == nil {
		globalCache.mu.Lock()
		globalCache.data.OrgCount = initialOrgCount
		globalCache.mu.Unlock()
	}

	fmt.Println("监控服务: 开始监听区块事件，正在同步全量数据...")
	events, err := network.BlockEvents(context.Background(), client.WithStartBlock(0))
	if err != nil {
		return fmt.Errorf("启动区块监听失败: %w", err)
	}

	go func() {
		globalCache.mu.Lock()
		globalCache.data.TxCount = 0
		globalCache.data.Height = 0
		globalCache.recentBlocks = make([]*BlockInfo, 0, MaxRecentBlocks)
		globalCache.mu.Unlock()

		for block := range events {
			if block == nil {
				continue
			}
			processBlock(block, channelName)
		}
		fmt.Println("监控服务: 区块事件通道已关闭")
	}()

	return nil
}

// GetChainStats 读取缓存的统计信息
func GetChainStats() ChainStats {
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()
	return globalCache.data
}

// GetTrendData 直接从内存返回最近的区块列表
func GetTrendData() []*BlockInfo {
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()
	result := make([]*BlockInfo, len(globalCache.recentBlocks))
	copy(result, globalCache.recentBlocks)
	return result
}

// processBlock 处理单个区块，更新统计和趋势缓存
func processBlock(block *common.Block, channelName string) {
	blockBytes, err := proto.Marshal(block)
	if err != nil {
		fmt.Printf("Error marshaling block for processing: %v\n", err)
		return
	}

	blockNum := block.GetHeader().GetNumber()
	// 调用 utils.go 中的标准解析逻辑
	blockInfo, err := parseBlockInfo(blockBytes, blockNum, channelName, false)
	if err != nil {
		fmt.Printf("Error parsing block info in monitor: %v\n", err)
		return
	}

	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.data.Height = blockNum + 1
	globalCache.data.TxCount += blockInfo.TxCount
	if !blockInfo.Timestamp.IsZero() {
		globalCache.data.LastBlockTime = blockInfo.Timestamp
	}

	// 维护最近区块切片 (倒序)
	newRecent := make([]*BlockInfo, 0, MaxRecentBlocks)
	newRecent = append(newRecent, blockInfo)
	currentLen := len(globalCache.recentBlocks)
	take := MaxRecentBlocks - 1
	if currentLen < take {
		take = currentLen
	}
	if take > 0 {
		newRecent = append(newRecent, globalCache.recentBlocks[:take]...)
	}
	globalCache.recentBlocks = newRecent
}
