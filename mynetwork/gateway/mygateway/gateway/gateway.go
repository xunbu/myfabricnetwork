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

//============下面为创建gateway连接的代码=======

// newGrpcConnection 函数用于创建到网关服务器的 gRPC 连接。
func NewGrpcConnection(tlsCertPath string, gatewayPeer string, peerEndpoint string) *grpc.ClientConn {
	certificatePEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		panic(fmt.Errorf("failed to read TLS certifcate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

func GetGateWay(clientConnection *grpc.ClientConn, mspID string, cryptoPath string, certPath string, keyPath string) (*client.Gateway, error) {

	id := newIdentity(certPath, mspID)
	sign := newSign(keyPath)

	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		fmt.Println("创建gateway链接错误")
		return nil, err
	}
	return gw, nil
}

// newIdentity creates a client identity for this Gateway connection using an X.509 certificate.
func newIdentity(certPath string, mspID string) *identity.X509Identity {
	certificatePEM, err := readFirstFile(certPath)
	if err != nil {
		panic(fmt.Errorf("failed to read certificate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
func newSign(keyPath string) identity.Sign {
	privateKeyPEM, err := readFirstFile(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}

func readFirstFile(dirPath string) ([]byte, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}

	fileNames, err := dir.Readdirnames(1)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(path.Join(dirPath, fileNames[0]))
}

// ============下面为gateway代码=======
// 查询交易
func EvaluateTransaction(gw *client.Gateway, channelName string, chainCodeName string, funcName string, args ...string) ([]byte, error) {
	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chainCodeName)
	result, err := contract.EvaluateTransaction(funcName, args...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// 提交交易
func SubmitTransaction(gw *client.Gateway, channelName string, chainCodeName string, funcName string, args ...string) ([]byte, error) {
	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chainCodeName)
	result, err := contract.SubmitTransaction(funcName, args...)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// 获取交易总数
func GetTransactionCount(gw *client.Gateway, channelName string) uint64 {
	network := gw.GetNetwork(channelName)
	qsccContract := network.GetContract("qscc")
	fmt.Println("\n--> 正在获取交易总数...")

	// 1. 获取区块链信息，主要是为了得到区块高度
	// 调用系统链码 'qscc' 的 'GetChainInfo' 方法。它返回的是一个序列化的 common.BlockchainInfo protobuf 消息
	chainInfoBytes, err := qsccContract.EvaluateTransaction("GetChainInfo", channelName)
	if err != nil {
		panic(fmt.Errorf("获取区块链信息失败: %w", err))
	}

	// 使用 proto.Unmarshal 来解析 Protobuf 数据，而不是 json.Unmarshal
	var chainInfo common.BlockchainInfo
	if err := proto.Unmarshal(chainInfoBytes, &chainInfo); err != nil {
		panic(fmt.Errorf("解析区块链信息失败: %w", err))
	}

	blockchainHeight := chainInfo.Height
	fmt.Printf("当前区块高度为: %d\n", blockchainHeight)

	var totalTransactionCount uint64 = 0

	// 2. 遍历所有区块
	for blockNumber := range blockchainHeight {
		// 3. 通过 qscc 的 GetBlockByNumber 方法获取区块数据
		// 返回的是序列化后的 common.Block protobuf 消息
		blockBytes, err := qsccContract.EvaluateTransaction("GetBlockByNumber", channelName, fmt.Sprint(blockNumber))
		if err != nil {
			fmt.Printf("警告: 获取区块 %d 失败: %v\n", blockNumber, err)
			continue // 如果某个区块获取失败，可以跳过或进行错误处理
		}

		// 4. 反序列化区块数据
		var block common.Block
		if err := proto.Unmarshal(blockBytes, &block); err != nil {
			fmt.Printf("警告: 解析区块 %d 失败: %v\n", blockNumber, err)
			continue
		}

		// 5. 统计区块中的交易数量
		// block.Data.Data 是一个字节数组的切片，每个元素代表一笔交易
		transactionCountInBlock := len(block.Data.Data)
		fmt.Printf("区块 %d 中包含 %d 笔交易\n", blockNumber, transactionCountInBlock)
		totalTransactionCount += uint64(transactionCountInBlock)
	}

	fmt.Printf("\n--> 完成: 通道 '%s' 上的交易总数为: %d\n", channelName, totalTransactionCount)
	return totalTransactionCount
}

func GetBlockHeight(gw *client.Gateway, channelName string) int {
	fmt.Println("\n--> 正在获取区块高度...")

	chainInfoBytes, err := EvaluateTransaction(gw, channelName, "qscc", "GetChainInfo", channelName)
	if err != nil {
		panic(fmt.Errorf("获取区块链信息失败: %w", err))
	}

	// 使用 proto.Unmarshal 来解析 Protobuf 数据，而不是 json.Unmarshal
	var chainInfo common.BlockchainInfo
	if err := proto.Unmarshal(chainInfoBytes, &chainInfo); err != nil {
		panic(fmt.Errorf("解析区块链信息失败: %w", err))
	}

	blockchainHeight := chainInfo.Height
	fmt.Printf("当前区块高度为: %d\n", blockchainHeight)
	return int(blockchainHeight)
}

// 获取组织数
func GetOrganizationCount(gw *client.Gateway, channelName string) (int, error) {
	network := gw.GetNetwork(channelName)
	csccContract := network.GetContract("cscc")
	fmt.Println("\n--> 正在获取通道组织数量...")

	// 1. 调用CSCC的GetConfigBlock方法获取配置区块
	configBlockBytes, err := csccContract.EvaluateTransaction("GetConfigBlock", channelName)
	if err != nil {
		return 0, fmt.Errorf("调用CSCC获取配置区块失败: %w", err)
	}

	// 2. 反序列化配置区块
	var configBlock common.Block
	if err := proto.Unmarshal(configBlockBytes, &configBlock); err != nil {
		return 0, fmt.Errorf("解析配置区块失败: %w", err)
	}

	// 3. 从区块数据中提取通道配置
	if len(configBlock.Data.Data) == 0 {
		return 0, fmt.Errorf("配置区块中没有数据")
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

	// 4. 从通道配置中提取组织信息
	channelGroup := configEnvelope.Config.ChannelGroup
	if channelGroup == nil {
		return 0, fmt.Errorf("配置中缺少ChannelGroup")
	}

	// 检查是否是应用通道（Application）或系统通道（Consortium）
	var orgsGroup *common.ConfigGroup
	if appGroup, exists := channelGroup.Groups["Application"]; exists {
		// 应用通道的组织在Application.groups下
		orgsGroup = appGroup
	} else if consortiumGroup, exists := channelGroup.Groups["Consortiums"]; exists {
		// 系统通道的组织在Consortiums.groups下
		// 注意：系统通道可能有多个联盟，这里取第一个联盟的组织
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

	// 5. 统计组织数量
	orgCount := len(orgsGroup.Groups)
	fmt.Printf("通道 '%s' 中的组织数量为: %d\n", channelName, orgCount)

	return orgCount, nil
}

// Format JSON data
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}
