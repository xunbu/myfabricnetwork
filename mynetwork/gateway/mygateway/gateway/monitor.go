package gateway

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"google.golang.org/protobuf/proto"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- GORM Models ---

// BlockModel 映射数据库 blocks 表
// 复合主键: ChannelID + BlockNumber
type BlockModel struct {
	BlockNumber  uint64    `gorm:"primaryKey;autoIncrement:false;column:block_number" json:"blockNumber"`
	ChannelID    string    `gorm:"primaryKey;size:128;column:channel_id" json:"channelId"`
	BlockHash    string    `gorm:"size:128;column:block_hash" json:"blockHash"`
	PreviousHash string    `gorm:"size:128;column:previous_hash" json:"previousHash"`
	DataHash     string    `gorm:"size:128;column:data_hash" json:"dataHash"`
	TxCount      uint64    `gorm:"column:tx_count" json:"txCount"`
	BlockSize    int64     `gorm:"column:block_size" json:"blockSize"`
	BlockCreator string    `gorm:"size:255;column:block_creator" json:"blockCreator"`
	CreatedAt    time.Time `gorm:"column:created_at;index" json:"createdAt"`
}

// TableName 自定义表名
func (BlockModel) TableName() string {
	return "blocks"
}

// ChannelInfoModel 映射数据库 channel_info 表
type ChannelInfoModel struct {
	ChannelID    string    `gorm:"primaryKey;size:128;column:channel_id"`
	GenesisHash  string    `gorm:"size:128;column:genesis_hash"`
	Height       uint64    `gorm:"column:height"`
	TotalTxCount uint64    `gorm:"column:total_tx_count"`
	OrgCount     int       `gorm:"column:org_count"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

// TableName 自定义表名
func (ChannelInfoModel) TableName() string {
	return "channel_info"
}

// --- Memory Cache Structures ---

// ChainStats 纯数据结构体 (用于内存缓存和前端API)
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

const MaxRecentBlocks = 1000 // 缓存最近多少个区块用于趋势图

// 全局单例缓存 (内存中保留一份，用于快速响应前端 API)
var globalCache = &safeCache{
	data:         ChainStats{OrgCount: 0, TxCount: 0, Height: 0},
	recentBlocks: make([]*BlockInfo, 0, MaxRecentBlocks),
}

// 全局数据库连接
var db *gorm.DB

// InitStore 初始化数据库连接并自动迁移表结构
func InitStore(dsn string) error {
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info), // 调试时可开启
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 自动迁移模式
	if err := db.AutoMigrate(&BlockModel{}, &ChannelInfoModel{}); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	return nil
}

// StartChainMonitor 启动区块监听 (带数据库持久化和一致性检查)
func StartChainMonitor(gw *client.Gateway, channelName string) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化，请先调用 InitStore")
	}

	network := gw.GetNetwork(channelName)

	// --- 1. 获取链上真实的创世块信息 ---
	// GetBlockByNum 在 ledger.go 中定义
	genesisBlockInfo, err := GetBlockByNum(gw, channelName, 0)
	if err != nil {
		return fmt.Errorf("无法获取链上创世块: %w", err)
	}
	realGenesisHash := genesisBlockInfo.BlockHash

	// --- 2. 检查数据库中的通道信息 ---
	var channelInfo ChannelInfoModel
	err = db.Where("channel_id = ?", channelName).First(&channelInfo).Error

	isMismatch := false
	if err == gorm.ErrRecordNotFound {
		isMismatch = true // 不存在视为不匹配
	} else if err != nil {
		return fmt.Errorf("查询数据库通道信息失败: %w", err)
	} else {
		if channelInfo.GenesisHash != realGenesisHash {
			isMismatch = true
		}
	}

	startBlockNum := int64(0)

	// --- 3. 判断是否需要重置数据 ---
	if isMismatch {
		fmt.Printf("监控服务: 检测到环境变化或首次运行 (Chain Hash: %s)。正在重置数据库...\n", realGenesisHash)

		// 使用事务清空旧数据并初始化
		err = db.Transaction(func(tx *gorm.DB) error {
			// 删除该通道的所有区块数据
			if err := tx.Where("channel_id = ?", channelName).Delete(&BlockModel{}).Error; err != nil {
				return err
			}
			// 删除旧通道信息
			if err := tx.Where("channel_id = ?", channelName).Delete(&ChannelInfoModel{}).Error; err != nil {
				return err
			}

			// 插入新的通道信息
			newInfo := ChannelInfoModel{
				ChannelID:   channelName,
				GenesisHash: realGenesisHash,
				Height:      0,
				UpdatedAt:   time.Now(),
			}
			if err := tx.Create(&newInfo).Error; err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("重置数据库失败: %w", err)
		}
		startBlockNum = 0
	} else {
		// --- 4. 如果哈希一致，查找断点 (最大高度) ---
		var lastBlock BlockModel
		// 查找当前通道最大 block_number
		err := db.Where("channel_id = ?", channelName).Order("block_number desc").First(&lastBlock).Error
		if err == nil {
			startBlockNum = int64(lastBlock.BlockNumber) + 1
			fmt.Printf("监控服务: 创世块校验通过，从区块 %d 开始断点续传...\n", startBlockNum)
		} else {
			// 如果没有区块记录，从 0 开始
			startBlockNum = 0
		}
	}

	// 初始化内存缓存中的组织数量
	initialOrgCount, err := GetOrganizationCount(gw, channelName)
	if err == nil {
		globalCache.mu.Lock()
		globalCache.data.OrgCount = initialOrgCount
		globalCache.mu.Unlock()

		// 同步更新到 DB
		db.Model(&ChannelInfoModel{}).Where("channel_id = ?", channelName).Update("org_count", initialOrgCount)
	}

	// --- 5. 启动事件监听 ---
	events, err := network.BlockEvents(context.Background(), client.WithStartBlock(uint64(startBlockNum)))
	if err != nil {
		return fmt.Errorf("启动区块监听失败: %w", err)
	}

	go func() {
		// 如果是从 0 开始，清空内存缓存
		if startBlockNum == 0 {
			globalCache.mu.Lock()
			globalCache.data.TxCount = 0
			globalCache.data.Height = 0
			globalCache.recentBlocks = make([]*BlockInfo, 0, MaxRecentBlocks)
			globalCache.mu.Unlock()
		} else {
			// 如果是断点续传，从 DB 恢复内存中的计数器
			restoreCacheFromDB(channelName)
		}

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

// restoreCacheFromDB 从数据库恢复内存中的计数器
func restoreCacheFromDB(channelName string) {
	var info ChannelInfoModel
	if err := db.Where("channel_id = ?", channelName).First(&info).Error; err == nil {
		globalCache.mu.Lock()
		globalCache.data.Height = info.Height
		globalCache.data.TxCount = info.TotalTxCount
		globalCache.mu.Unlock()
	}
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

// processBlock 处理单个区块，更新统计、趋势缓存并写入 MySQL
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

	// 1. 更新内存缓存 (用于前端快速展示)
	globalCache.mu.Lock()
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
	globalCache.mu.Unlock()

	// 2. 异步写入 MySQL 数据库
	go persistBlockToDB(blockInfo, channelName)
}

func persistBlockToDB(info *BlockInfo, channelName string) {
	if db == nil {
		return
	}

	// 构建 BlockModel 对象
	blockModel := BlockModel{
		BlockNumber:  info.BlockNumber,
		ChannelID:    channelName,
		BlockHash:    info.BlockHash,
		PreviousHash: info.PreviousHash,
		DataHash:     info.MerkleRoot,
		TxCount:      info.TxCount,
		BlockSize:    info.BlockSize,
		BlockCreator: info.BlockCreator,
		CreatedAt:    info.Timestamp,
	}

	// 使用事务确保区块插入和通道信息更新的原子性
	// 即使是异步执行，为了数据一致性最好也放在事务里
	err := db.Transaction(func(tx *gorm.DB) error {
		// 1. 插入区块 (如果有重复主键则忽略，使用 Clauses)
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&blockModel).Error; err != nil {
			return err
		}

		// 2. 更新通道汇总信息
		// 更新 Height 和 TotalTxCount
		// 注意：这里的 Height 是当前区块号 + 1
		// gorm.Expr 用于执行 SQL 表达式
		if err := tx.Model(&ChannelInfoModel{}).
			Where("channel_id = ?", channelName).
			Updates(map[string]interface{}{
				"height":         info.BlockNumber + 1,
				"total_tx_count": gorm.Expr("total_tx_count + ?", info.TxCount),
				"updated_at":     info.Timestamp,
			}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Printf("写入区块 [%d] 到数据库失败: %v", info.BlockNumber, err)
	}
}
