package gateway

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"
	"guolong.com/basic-chaincode/chaincode"
)

// ==========================================
// 第一部分：全局监控缓存 (高性能核心)
// ==========================================

// BlockInfo 定义区块详情结构 (与 utils.go 配合使用)
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
	data: ChainStats{
		OrgCount: 0,
		TxCount:  0,
		Height:   0,
	},
	recentBlocks: make([]*BlockInfo, 0, MaxRecentBlocks),
}

// StartChainMonitor 启动全量区块监听
func StartChainMonitor(gw *client.Gateway, channelName string) error {
	network := gw.GetNetwork(channelName)

	// 1. 初始化组织数量
	initialOrgCount, err := GetOrganizationCount(gw, channelName)
	if err == nil {
		globalCache.mu.Lock()
		globalCache.data.OrgCount = initialOrgCount
		globalCache.mu.Unlock()
	}

	// 2. 订阅区块事件
	fmt.Println("监控服务: 开始监听区块事件，正在同步全量数据...")
	events, err := network.BlockEvents(context.Background(), client.WithStartBlock(0))
	if err != nil {
		return fmt.Errorf("启动区块监听失败: %w", err)
	}

	// 3. 异步处理
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

// GetChainStats 读取缓存的统计信息 (O(1))
func GetChainStats() ChainStats {
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()
	return globalCache.data
}

// GetTrendData 直接从内存返回最近的区块列表 (O(1))
func GetTrendData() []*BlockInfo {
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()

	// 返回副本，保证并发安全
	result := make([]*BlockInfo, len(globalCache.recentBlocks))
	copy(result, globalCache.recentBlocks)
	return result
}

// processBlock 处理单个区块，更新统计和趋势缓存
func processBlock(block *common.Block, channelName string) {
	// 为了复用 utils.go 中的标准解析逻辑，这里先将 block 序列化为 bytes
	// 虽然多了一次 Marshal/Unmarshal，但保证了 Hash 计算的准确性，且代码逻辑统一
	// 这里的性能损耗对于后台监控协程是可以接受的
	blockBytes, err := proto.Marshal(block)
	if err != nil {
		fmt.Printf("Error marshaling block for processing: %v\n", err)
		return
	}

	blockNum := block.GetHeader().GetNumber()

	// 调用 utils.go 中的 parseBlockInfo 获取标准化的 BlockInfo
	// includeTxDetails=false，因为监控统计和趋势图不需要详细的 TxID 列表，节省内存
	blockInfo, err := parseBlockInfo(blockBytes, blockNum, channelName, false)
	if err != nil {
		fmt.Printf("Error parsing block info in monitor: %v\n", err)
		return
	}

	// --- 更新全局缓存 ---
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	// 1. 更新高度
	globalCache.data.Height = blockNum + 1

	// 2. 更新交易数量
	// blockInfo.TxCount 已经在 parseBlockInfo 中计算好了
	globalCache.data.TxCount += blockInfo.TxCount

	// 3. 更新时间
	// parseBlockInfo 已经处理了时间戳提取和创世区块的特殊情况
	if !blockInfo.Timestamp.IsZero() {
		globalCache.data.LastBlockTime = blockInfo.Timestamp
	}

	// 4. 更新趋势图缓存 (RecentBlocks)
	// 维护固定长度的切片 (倒序：新的在最前)
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

// ==========================================
// 第二部分：Gateway 连接与基础配置
// ==========================================

func NewGrpcConnection(tlsCertPath, gatewayPeer, peerEndpoint string) (*grpc.ClientConn, error) {
	certificatePEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		return nil, fmt.Errorf("读取TLS证书文件失败: %w", err)
	}
	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)
	connection, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		return nil, fmt.Errorf("创建gRPC连接失败: %w", err)
	}
	return connection, nil
}

func GetGateway(clientConnection *grpc.ClientConn, mspID, cryptoPath, certPath, keyPath string) (*client.Gateway, error) {
	id, err := newIdentity(certPath, mspID)
	if err != nil {
		return nil, fmt.Errorf("创建身份失败: %w", err)
	}
	sign, err := newSign(keyPath)
	if err != nil {
		return nil, fmt.Errorf("创建签名器失败: %w", err)
	}
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(clientConnection),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("连接到gateway失败: %w", err)
	}
	return gw, nil
}

func newIdentity(certPath, mspID string) (*identity.X509Identity, error) {
	certificatePEM, err := readFirstFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("读取证书文件失败: %w", err)
	}
	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}
	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		return nil, fmt.Errorf("创建X509身份失败: %w", err)
	}
	return id, nil
}

func newSign(keyPath string) (identity.Sign, error) {
	privateKeyPEM, err := readFirstFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("读取私钥文件失败: %w", err)
	}
	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %w", err)
	}
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, fmt.Errorf("创建签名器失败: %w", err)
	}
	return sign, nil
}

func readFirstFile(dirPath string) ([]byte, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, fmt.Errorf("打开目录失败: %w", err)
	}
	fileNames, err := dir.Readdirnames(1)
	if err != nil {
		return nil, fmt.Errorf("读取目录内容失败: %w", err)
	}
	if len(fileNames) == 0 {
		return nil, fmt.Errorf("目录中没有文件")
	}
	data, err := os.ReadFile(path.Join(dirPath, fileNames[0]))
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	return data, nil
}

// ==========================================
// 第三部分：链上交互 (交易与查询)
// ==========================================

func EvaluateTransaction(gw *client.Gateway, channelName, chainCodeName, funcName string, args ...string) ([]byte, error) {
	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chainCodeName)
	return contract.EvaluateTransaction(funcName, args...)
}

func SubmitTransaction(gw *client.Gateway, channelName, chainCodeName, funcName string, args ...string) ([]byte, error) {
	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chainCodeName)
	return contract.SubmitTransaction(funcName, args...)
}

func PutValue(gw *client.Gateway, channelName string, chaincodeName string, key string, data string) ([]byte, error) {
	return SubmitTransaction(gw, channelName, chaincodeName, "PutValue", key, data)
}

func PutMap(gw *client.Gateway, channelName string, chaincodeName string, key string, jsonMap map[string]interface{}) ([]byte, error) {
	jsonBytes, err := json.Marshal(jsonMap)
	if err != nil {
		return nil, fmt.Errorf("JSON 序列化失败: %w", err)
	}
	return SubmitTransaction(gw, channelName, chaincodeName, "PutValue", key, string(jsonBytes))
}

func PutKVs(gw *client.Gateway, channelName string, chaincodeName string, KVMap map[string]string) ([]byte, error) {
	v, err := json.Marshal(KVMap)
	if err != nil {
		return nil, err
	}
	return SubmitTransaction(gw, channelName, chaincodeName, "PutKVs", string(v))
}

func GetValue(gw *client.Gateway, channelName string, chaincodeName string, key string) (string, error) {
	v, err := EvaluateTransaction(gw, channelName, chaincodeName, "QueryByKey", key)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

func DeleteKey(gw *client.Gateway, channelName string, chaincodeName string, key string) ([]byte, error) {
	return SubmitTransaction(gw, channelName, chaincodeName, "DeleteByKey", key)
}

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
		return 0, fmt.Errorf("无法确定通道类型")
	}
	if orgsGroup == nil {
		return 0, fmt.Errorf("配置中缺少组织信息")
	}
	return len(orgsGroup.Groups), nil
}

// ==========================================
// 第四部分：数据查询与分页
// ==========================================

type BlockPage struct {
	Results  []*BlockInfo `json:"blockPage"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
	Total    int          `json:"total"`
}

// GetBlockListByPage 分页查询区块列表
func GetBlockListByPage(gw *client.Gateway, channelName string, pageNum uint64, pageSize uint64, includeTxDetails bool) (*BlockPage, error) {
	// 1. 获取当前区块高度 (优先使用缓存)
	var blockHeight uint64
	stats := GetChainStats()
	if stats.Height > 0 {
		blockHeight = stats.Height
	} else {
		h, err := GetBlockHeight(gw, channelName)
		if err != nil {
			return nil, fmt.Errorf("获取区块高度失败: %w", err)
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
			return nil, fmt.Errorf("获取区块%d失败: %w", blockNumber, err)
		}

		// 复用 utils.go 中的标准解析逻辑
		blockInfo, err := parseBlockInfo(blockBytes, uint64(blockNumber), channelName, includeTxDetails)
		if err != nil {
			return nil, fmt.Errorf("解析区块%d失败: %w", blockNumber, err)
		}
		blockList = append(blockList, blockInfo)
	}

	return &BlockPage{
		Results:  blockList,
		Page:     int(pageNum),
		PageSize: int(pageSize),
		Total:    int(totalBlocks),
	}, nil
}

func GetBlockByNum(gw *client.Gateway, channelName string, blockNum uint64) (*BlockInfo, error) {
	blockBytes, err := EvaluateTransaction(gw, channelName, "qscc", "GetBlockByNumber", channelName, fmt.Sprint(blockNum))
	if err != nil {
		return nil, fmt.Errorf("获取区块%d失败: %w", blockNum, err)
	}
	// 复用 utils.go 中的标准解析逻辑
	blockInfo, err := parseBlockInfo(blockBytes, blockNum, channelName, true)
	if err != nil {
		return nil, fmt.Errorf("解析区块%d失败: %w", blockNum, err)
	}
	return blockInfo, nil
}

func GetAllData(gw *client.Gateway, channelName string, chaincodeName string) ([]chaincode.QueryRichResult, error) {
	query := map[string]interface{}{"selector": map[string]interface{}{}}
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	v, err := EvaluateTransaction(gw, channelName, chaincodeName, "QueryByRich", string(queryBytes))
	if err != nil {
		return nil, err
	}
	if len(v) == 0 {
		return []chaincode.QueryRichResult{}, nil
	}
	var richResults []chaincode.QueryRichResult
	err = json.Unmarshal(v, &richResults)
	if err != nil {
		return nil, err
	}
	return richResults, nil
}

type RichPageResult struct {
	Results  []chaincode.QueryRichResult `json:"results"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"pageSize"`
	Total    int                         `json:"total"`
}

func GetAllDataByPageWithLimit(gw *client.Gateway, channelName string, chaincodeName string, page int, pageSize int) (*RichPageResult, error) {
	skip := page * pageSize
	query := map[string]interface{}{"selector": map[string]interface{}{}, "skip": skip}
	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	v, err := EvaluateTransaction(gw, channelName, chaincodeName, "QueryByRichWithLimit", string(queryBytes), strconv.Itoa(pageSize))
	if err != nil {
		return nil, err
	}
	var results []chaincode.QueryRichResult
	if len(v) > 0 {
		err = json.Unmarshal(v, &results)
		if err != nil {
			return nil, err
		}
	}
	return &RichPageResult{
		Results:  results,
		Page:     page,
		PageSize: pageSize,
		Total:    skip + len(results),
	}, nil
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

func GetTxByID(gw *client.Gateway, channelName string, TxID string) (*Txinfo, error) {
	v, err := EvaluateTransaction(gw, channelName, "qscc", "GetTransactionByID", channelName, TxID)
	if err != nil {
		return nil, fmt.Errorf("EvaluateTransaction Error: %w", err)
	}

	txInfo := &Txinfo{Size: len(v)}
	tx := &peer.ProcessedTransaction{}
	err = proto.Unmarshal(v, tx)
	if err != nil {
		return nil, err
	}

	txInfo.ValidationCode = int(tx.ValidationCode)
	if tx.TransactionEnvelope == nil || tx.TransactionEnvelope.Payload == nil {
		return txInfo, nil
	}

	payload := &common.Payload{}
	err = proto.Unmarshal(tx.TransactionEnvelope.Payload, payload)
	if err != nil {
		return txInfo, err
	}
	if payload.Header == nil {
		return txInfo, nil
	}
	chHeader := &common.ChannelHeader{}
	err = proto.Unmarshal(payload.Header.ChannelHeader, chHeader)
	if err != nil {
		return txInfo, err
	}
	txInfo.TxId = chHeader.TxId
	txInfo.ChannelId = chHeader.ChannelId
	txInfo.Type = common.HeaderType_name[chHeader.Type]
	txInfo.Timestamp = chHeader.Timestamp.AsTime()

	if common.HeaderType(chHeader.Type) == common.HeaderType_ENDORSER_TRANSACTION {
		transaction := &peer.Transaction{}
		err = proto.Unmarshal(payload.Data, transaction)
		if err != nil {
			return nil, err
		}
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
				Args:          convertBytesToStrings(spec.Input.Args), // 调用 utils.go 中的函数
			})
		}
	}
	return txInfo, nil
}

func GetKeyHistory(gw *client.Gateway, channelName string, chaincodeName, key string) (*[]chaincode.KeyHistory, error) {
	v, err := EvaluateTransaction(gw, channelName, chaincodeName, "GetKeyHistory", key)
	if err != nil {
		return nil, err
	}
	kh := &[]chaincode.KeyHistory{}
	json.Unmarshal(v, &kh)
	return kh, nil
}
