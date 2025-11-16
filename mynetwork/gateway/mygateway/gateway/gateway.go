package gateway

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"path"
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

//============ 创建gateway连接的代码 =======

// NewGrpcConnection 创建到网关服务器的gRPC连接
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

// GetGateway 创建并返回Gateway实例
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

// newIdentity 使用X.509证书创建客户端身份
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

// newSign 创建生成数字签名的函数
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

// readFirstFile 读取目录中的第一个文件
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

// EvaluateTransaction 执行查询交易
func EvaluateTransaction(gw *client.Gateway, channelName, chainCodeName, funcName string, args ...string) ([]byte, error) {
	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chainCodeName)
	return contract.EvaluateTransaction(funcName, args...)
}

// SubmitTransaction 执行提交交易
func SubmitTransaction(gw *client.Gateway, channelName, chainCodeName, funcName string, args ...string) ([]byte, error) {
	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chainCodeName)
	return contract.SubmitTransaction(funcName, args...)
}

// 写入数据
func PutString(gw *client.Gateway, channelName string, chaincodeName string, key string, data string) ([]byte, error) {
	return SubmitTransaction(gw, channelName, chaincodeName, "PutString", key, data)
}

// 写入map
func PutMap(gw *client.Gateway, channelName string, chaincodeName string, key string, jsonMap map[string]interface{}) ([]byte, error) {
	jsonBytes, err := json.Marshal(jsonMap)
	if err != nil {
		return nil, fmt.Errorf("JSON 序列化失败: %w", err)
	}
	jsonString := string(jsonBytes)
	return SubmitTransaction(gw, channelName, chaincodeName, "PutString", key, jsonString)
}

// 写入map
func PutKvs(gw *client.Gateway, channelName string, chaincodeName string, KVMap map[string]string) ([]byte, error) {
	v, err := json.Marshal(KVMap)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(v))
	return SubmitTransaction(gw, channelName, chaincodeName, "PutKVs", string(v))
}

func GetValue(gw *client.Gateway, channelName string, chaincodeName string, key string) ([]byte, error) {
	return EvaluateTransaction(gw, channelName, chaincodeName, "QueryByKey", key)
}
func DeleteKey(gw *client.Gateway, channelName string, chaincodeName string, key string) ([]byte, error) {
	return SubmitTransaction(gw, channelName, chaincodeName, "DeleteByKey", key)
}

// GetTransactionCount 返回通道中的交易总数
func GetTransactionCount(gw *client.Gateway, channelName string) (uint64, error) {
	// 复用EvaluateTransaction获取区块链信息
	chainInfoBytes, err := EvaluateTransaction(gw, channelName, "qscc", "GetChainInfo", channelName)
	if err != nil {
		return 0, fmt.Errorf("获取区块链信息失败: %w", err)
	}

	var chainInfo common.BlockchainInfo
	if err := proto.Unmarshal(chainInfoBytes, &chainInfo); err != nil {
		return 0, fmt.Errorf("解析区块链信息失败: %w", err)
	}

	var totalTransactionCount uint64 = 0
	blockchainHeight := chainInfo.Height

	// 遍历所有区块统计交易数量
	for blockNumber := uint64(0); blockNumber < blockchainHeight; blockNumber++ {
		// 复用EvaluateTransaction获取区块数据
		blockBytes, err := EvaluateTransaction(gw, channelName, "qscc", "GetBlockByNumber", channelName, fmt.Sprint(blockNumber))
		if err != nil {
			return totalTransactionCount, fmt.Errorf("获取区块%d失败: %w", blockNumber, err)
		}

		var block common.Block
		if err := proto.Unmarshal(blockBytes, &block); err != nil {
			return totalTransactionCount, fmt.Errorf("解析区块%d失败: %w", blockNumber, err)
		}

		totalTransactionCount += uint64(len(block.Data.Data))
	}

	return totalTransactionCount, nil
}

// GetBlockHeight 返回通道的当前区块高度
func GetBlockHeight(gw *client.Gateway, channelName string) (uint64, error) {
	// 复用EvaluateTransaction获取区块链信息
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

// GetOrganizationCount 返回通道中的组织数量
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

type BlockInfo struct {
	BlockHash    string    `json:"blockHash"`
	PreviousHash string    `json:"previousHash"`
	MerkleRoot   string    `json:"dataHash"`
	BlockNumber  uint64    `json:"blockNumber"`
	TxCount      uint64    `json:"txCount"`
	BlockSize    int64     `json:"blockSize"`
	Timestamp    time.Time `json:"timestamp"`
	ChannelID    string    `json:"channelId"`
	BlockCreator string    `json:"blockCreator,omitempty"` // 可选字段
	TxIDs        []string  `json:"txIds,omitempty"`        // 可选字段，交易ID列表
}

// =============分页查询区块列表
// 分页查询区块列表
// 解析区块信息的辅助函数
func GetBlockListByPage(gw *client.Gateway, channelName string, pageNum uint64, pageSize uint64, includeTxDetails bool) ([]*BlockInfo, error) {
	// 获取当前区块高度
	blockHeight, err := GetBlockHeight(gw, channelName)
	if err != nil {
		return nil, fmt.Errorf("获取区块高度失败: %w", err)
	}

	// 计算起始和结束区块号（改为倒序排列，最新的区块在前）
	totalBlocks := int64(blockHeight)
	if totalBlocks == 0 {
		return []*BlockInfo{}, nil
	}

	startIdx := int64(pageNum * pageSize)
	if startIdx >= totalBlocks {
		return []*BlockInfo{}, nil // 超出范围返回空列表
	}

	endIdx := startIdx + int64(pageSize) - 1
	if endIdx >= totalBlocks {
		endIdx = totalBlocks - 1
	}

	var blockList []*BlockInfo

	// 倒序遍历，最新的区块在前
	for blockNumber := endIdx; blockNumber >= startIdx; blockNumber-- {
		blockBytes, err := EvaluateTransaction(gw, channelName, "qscc", "GetBlockByNumber", channelName, fmt.Sprint(blockNumber))
		if err != nil {
			return blockList, fmt.Errorf("获取区块%d失败: %w", blockNumber, err)
		}

		blockInfo, err := parseBlockInfo(blockBytes, uint64(blockNumber), channelName, includeTxDetails)
		if err != nil {
			return blockList, fmt.Errorf("解析区块%d失败: %w", blockNumber, err)
		}

		blockList = append(blockList, blockInfo)
	}

	return blockList, nil
}

// 根据区块号获取区块详情
func GetBlockByNum(gw *client.Gateway, channelName string, blockNum uint64) (*BlockInfo, error) {
	// 复用EvaluateTransaction获取区块数据
	blockBytes, err := EvaluateTransaction(gw, channelName, "qscc", "GetBlockByNumber", channelName, fmt.Sprint(blockNum))
	if err != nil {
		return nil, fmt.Errorf("获取区块%d失败: %w", blockNum, err)
	}

	// 复用parseBlockInfo函数解析区块信息
	blockInfo, err := parseBlockInfo(blockBytes, blockNum, channelName, true) // 默认包含交易详情
	if err != nil {
		return nil, fmt.Errorf("解析区块%d失败: %w", blockNum, err)
	}

	return blockInfo, nil
}

func GetAllData(gw *client.Gateway, channelName string, chaincodeName string) ([]chaincode.QueryRichResult, error) {
	v, err := EvaluateTransaction(gw, channelName, chaincodeName, "QueryByRange", "", "")
	if err != nil {
		return nil, err
	}
	if len(v) == 0 {
		return []chaincode.QueryRichResult{}, nil
	}
	var results []chaincode.QueryRichResult
	err = json.Unmarshal(v, &results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

type Txinfo struct {
	TxId      string
	ChannelId string
	Type      string
	Timestamp time.Time

	Size           int
	ValidationCode int
	ChainCodeInfos []*ChainCodeInfo
}

type ChainCodeInfo struct {
	ChainCodeName string
	Args          []string
}

func (txInfo *Txinfo) String() string {
	return fmt.Sprintf("TxId:%v\nChannelId:%v\nType:%v\nTimestamp:%v\nValidationCode:%v\nChainCodeInfos:\n%v\n", txInfo.TxId, txInfo.ChannelId, txInfo.Type, txInfo.Timestamp, txInfo.ValidationCode, txInfo.ChainCodeInfos)
}

func (chainCodeInfo *ChainCodeInfo) String() string {
	return fmt.Sprintf("ChainCodeName:%v,args:%v", chainCodeInfo.ChainCodeName, chainCodeInfo.Args)
}
func GetTxByID(gw *client.Gateway, channelName string, TxID string) (*Txinfo, error) {
	v, err := EvaluateTransaction(gw, channelName, "qscc", "GetTransactionByID", channelName, TxID)
	if err != nil {
		return nil, fmt.Errorf("error in EvaluateTransaction %w", err)
	}

	// 简单显示原始数据
	fmt.Printf("原始数据长度: %d bytes\n", len(v))
	// fmt.Printf("十六进制: %x\n", v)

	txInfo := &Txinfo{Size: len(v)}

	// 基础解析
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
	fmt.Printf("交易ID: %s\n", chHeader.TxId)
	fmt.Printf("通道: %s\n", chHeader.ChannelId)
	fmt.Printf("类型: %s\n", common.HeaderType_name[chHeader.Type])
	fmt.Printf("时间: %v\n", chHeader.Timestamp.AsTime())
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
			// 解析 ChaincodeActionPayload 等
			v := &peer.ChaincodeActionPayload{}
			proto.Unmarshal(action.Payload, v)
			v2 := &peer.ChaincodeProposalPayload{}
			proto.Unmarshal(v.ChaincodeProposalPayload, v2)
			invocationSpec := &peer.ChaincodeInvocationSpec{}
			proto.Unmarshal(v2.Input, invocationSpec)
			spec := invocationSpec.ChaincodeSpec
			txInfo.ChainCodeInfos = append(txInfo.ChainCodeInfos, &ChainCodeInfo{ChainCodeName: spec.ChaincodeId.Name, Args: convertBytesToStrings(spec.Input.Args)})
		}
	}

	return txInfo, nil
	// parseReadWriteSets(tx)
}

func GetKeyHistory(gw *client.Gateway, channelName string, chaincodeName, key string) (*[]chaincode.KeyHistory, error) {
	v, err := EvaluateTransaction(gw, channelName, "basic", "GetKeyHistory", key)
	if err != nil {
		return nil, err
	}
	kh := &[]chaincode.KeyHistory{}
	json.Unmarshal(v, &kh)
	return kh, nil
}
