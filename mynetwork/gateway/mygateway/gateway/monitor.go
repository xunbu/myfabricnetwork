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

// safeCache 内部使用的封装结构 (只保留统计数据，不再缓存区块列表)
type safeCache struct {
	data ChainStats   // 统计数据
	mu   sync.RWMutex // 读写锁
}

// 全局单例缓存
var globalCache = &safeCache{
	data: ChainStats{OrgCount: 0, TxCount: 0, Height: 0},
}

// 全局数据库连接
var db *gorm.DB

// InitStore 初始化数据库连接并自动迁移表结构
func InitStore(dsn string) error {
	var err error
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	if err := db.AutoMigrate(&BlockModel{}, &ChannelInfoModel{}); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	return nil
}

// StartChainMonitor 启动区块监听
func StartChainMonitor(gw *client.Gateway, channelName string) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化，请先调用 InitStore")
	}

	network := gw.GetNetwork(channelName)

	// 1. 获取链上真实的创世块信息
	genesisBlockInfo, err := GetBlockByNum(gw, channelName, 0)
	if err != nil {
		return fmt.Errorf("无法获取链上创世块: %w", err)
	}
	realGenesisHash := genesisBlockInfo.BlockHash

	// 2. 检查数据库中的通道信息
	var channelInfo ChannelInfoModel
	err = db.Where("channel_id = ?", channelName).First(&channelInfo).Error

	isMismatch := false
	if err == gorm.ErrRecordNotFound {
		isMismatch = true
	} else if err != nil {
		return fmt.Errorf("查询数据库通道信息失败: %w", err)
	} else {
		if channelInfo.GenesisHash != realGenesisHash {
			isMismatch = true
		}
	}

	startBlockNum := int64(0)

	// 3. 判断是否需要重置数据
	if isMismatch {
		fmt.Printf("监控服务: 检测到环境变化或首次运行 (Chain Hash: %s)。正在重置数据库...\n", realGenesisHash)
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("channel_id = ?", channelName).Delete(&BlockModel{}).Error; err != nil {
				return err
			}
			if err := tx.Where("channel_id = ?", channelName).Delete(&ChannelInfoModel{}).Error; err != nil {
				return err
			}
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
		// 4. 断点续传
		var lastBlock BlockModel
		err := db.Where("channel_id = ?", channelName).Order("block_number desc").First(&lastBlock).Error
		if err == nil {
			startBlockNum = int64(lastBlock.BlockNumber) + 1
			fmt.Printf("监控服务: 创世块校验通过，从区块 %d 开始断点续传...\n", startBlockNum)
		} else {
			startBlockNum = 0
		}
	}

	// 初始化组织数量
	initialOrgCount, err := GetOrganizationCount(gw, channelName)
	if err == nil {
		globalCache.mu.Lock()
		globalCache.data.OrgCount = initialOrgCount
		globalCache.mu.Unlock()
		db.Model(&ChannelInfoModel{}).Where("channel_id = ?", channelName).Update("org_count", initialOrgCount)
	}

	// 5. 启动事件监听
	events, err := network.BlockEvents(context.Background(), client.WithStartBlock(uint64(startBlockNum)))
	if err != nil {
		return fmt.Errorf("启动区块监听失败: %w", err)
	}

	go func() {
		if startBlockNum == 0 {
			globalCache.mu.Lock()
			globalCache.data.TxCount = 0
			globalCache.data.Height = 0
			globalCache.mu.Unlock()
		} else {
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

// GetTrendData [修改后] 从数据库查询最近的 N 条记录
func GetTrendData(channelName string, limit int) ([]*BlockInfo, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	var models []BlockModel
	// 倒序查询最新的 limit 条数据
	err := db.Where("channel_id = ?", channelName).
		Order("block_number desc").
		Limit(limit).
		Find(&models).Error

	if err != nil {
		return nil, err
	}

	// 转换 Model -> BlockInfo
	results := make([]*BlockInfo, len(models))
	for i, m := range models {
		results[i] = &BlockInfo{
			BlockNumber:  m.BlockNumber,
			BlockHash:    m.BlockHash,
			PreviousHash: m.PreviousHash,
			MerkleRoot:   m.DataHash,
			TxCount:      m.TxCount,
			BlockSize:    m.BlockSize,
			Timestamp:    m.CreatedAt,
			ChannelID:    m.ChannelID,
			BlockCreator: m.BlockCreator,
			// TxIDs 在趋势图中通常不需要，数据库也没存列表，所以留空
			TxIDs: nil,
		}
	}

	return results, nil
}

// processBlock 处理单个区块
func processBlock(block *common.Block, channelName string) {
	blockBytes, err := proto.Marshal(block)
	if err != nil {
		fmt.Printf("Error marshaling block: %v\n", err)
		return
	}

	blockNum := block.GetHeader().GetNumber()
	blockInfo, err := parseBlockInfo(blockBytes, blockNum, channelName, false)
	if err != nil {
		fmt.Printf("Error parsing block info: %v\n", err)
		return
	}

	// 1. 更新内存中的统计数据 (Height, TxCount)
	globalCache.mu.Lock()
	globalCache.data.Height = blockNum + 1
	globalCache.data.TxCount += blockInfo.TxCount
	if !blockInfo.Timestamp.IsZero() {
		globalCache.data.LastBlockTime = blockInfo.Timestamp
	}
	// 注意：这里不再维护 recentBlocks 数组
	globalCache.mu.Unlock()

	// 2. 异步写入 MySQL
	go persistBlockToDB(blockInfo, channelName)
}

func persistBlockToDB(info *BlockInfo, channelName string) {
	if db == nil {
		return
	}

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

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&blockModel).Error; err != nil {
			return err
		}
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
		log.Printf("写入区块 [%d] 失败: %v", info.BlockNumber, err)
	}
}
