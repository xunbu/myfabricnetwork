package gateway

import (
	"bytes"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/proto"
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

//============ Gateway操作函数 =======

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

// FormatJSON 格式化JSON数据
func FormatJSON(data []byte) (string, error) {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		return "", fmt.Errorf("格式化JSON失败: %w", err)
	}
	return prettyJSON.String(), nil
}
