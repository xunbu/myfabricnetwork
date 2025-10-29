package admin

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path"

	"github.com/hyperledger/fabric-admin-sdk/pkg/chaincode"
	"github.com/hyperledger/fabric-admin-sdk/pkg/channel"
	"github.com/hyperledger/fabric-admin-sdk/pkg/discovery"
	"github.com/hyperledger/fabric-admin-sdk/pkg/identity"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/orderer"
	"github.com/hyperledger/fabric-protos-go-apiv2/orderer/etcdraft"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// ================channel
func GetConfigBlock(ctx context.Context, conn *grpc.ClientConn, mspID, cryptoPath, certPath, keyPath string, channelName string) (*common.Block, error) {
	id, err := identity.NewPrivateKeySigningIdentity(mspID, readCertificate(certPath), readPrivateKey(keyPath))
	if err != nil {
		return nil, fmt.Errorf("error in get peer %w", err)
	}
	return channel.GetConfigBlock(ctx, conn, id, channelName)
}

func getOrdererCountFromConfigBlock(block *common.Block) (uint64, error) {
	if len(block.Data.Data) == 0 {
		return 0, fmt.Errorf("block data is empty")
	}

	// 1. 解析第一个数据条目（配置交易）
	env := &common.Envelope{}
	if err := proto.Unmarshal(block.Data.Data[0], env); err != nil {
		return 0, err
	}

	// 2. 解析Payload
	payload := &common.Payload{}
	if err := proto.Unmarshal(env.Payload, payload); err != nil {
		return 0, err
	}

	// 3. 解析ChannelHeader
	chdr := &common.ChannelHeader{}
	if err := proto.Unmarshal(payload.Header.ChannelHeader, chdr); err != nil {
		return 0, err
	}

	// 4. 检查是否是配置交易
	if common.HeaderType(chdr.Type) != common.HeaderType_CONFIG {
		return 0, fmt.Errorf("not a config block, type: %s", common.HeaderType(chdr.Type))
	}

	// 5. 解析配置信封
	configEnvelope := &common.ConfigEnvelope{}
	if err := proto.Unmarshal(payload.Data, configEnvelope); err != nil {
		return 0, err
	}

	// 6. 获取Orderer组
	ordererGroup := configEnvelope.Config.ChannelGroup.Groups["Orderer"]
	if ordererGroup == nil {
		return 0, fmt.Errorf("orderer group not found")
	}

	// 7. 获取共识类型配置
	consensusTypeValue := ordererGroup.Values["ConsensusType"]
	if consensusTypeValue == nil {
		return 0, fmt.Errorf("consensus type not found")
	}

	consensusType := &orderer.ConsensusType{}
	if err := proto.Unmarshal(consensusTypeValue.Value, consensusType); err != nil {
		return 0, err
	}

	// 8. 检查是否为etcdraft共识
	if consensusType.Type != "etcdraft" {
		return 0, fmt.Errorf("unsupported consensus type: %s", consensusType.Type)
	}

	// 9. 解析etcdraft元数据获取orderer节点数量
	raftMetadata := &etcdraft.ConfigMetadata{}
	if err := proto.Unmarshal(consensusType.Metadata, raftMetadata); err != nil {
		return 0, fmt.Errorf("failed to unmarshal raft metadata: %v", err)
	}

	return uint64(len(raftMetadata.Consenters)), nil
}

// 获取orderer节点数
func GetOrdererCount(ctx context.Context, conn *grpc.ClientConn, mspID, cryptoPath, certPath, keyPath string, channelName string) (uint64, error) {
	block, err := GetConfigBlock(ctx, conn, mspID, cryptoPath, certPath, keyPath, channelName)
	if err != nil {
		return 0, err
	}
	return getOrdererCountFromConfigBlock(block)
}

// ================discovery
func GetDiscoveryPeer(conn *grpc.ClientConn, mspID, cryptoPath, certPath, keyPath string) (*discovery.Peer, error) {
	id, err := identity.NewPrivateKeySigningIdentity(mspID, readCertificate(certPath), readPrivateKey(keyPath))
	if err != nil {
		return nil, fmt.Errorf("error in get peer %w", err)
	}
	return discovery.NewPeer(conn, id), nil
}

// 获取peer节点总数
func GetPeersCount(discoveryPeer *discovery.Peer, channelName string) (uint64, error) {
	peersResult, err := discoveryPeer.PeerMembershipQuery(context.Background(), channelName, nil)
	if err != nil {
		return 0, fmt.Errorf("error in discovery PeerMembershipQuery,%w", err)
	}
	peersmap := peersResult.PeersByOrg
	count := 0
	for _, peers := range peersmap {
		count += len(peers.Peers)
	}
	return uint64(count), nil
}

// 获取节点（包括peer节点和orderer节点）总数
func GetNodesCount(discoveryPeer *discovery.Peer, channelName string, ctx context.Context, conn *grpc.ClientConn, mspID string, cryptoPath string, certPath string, keyPath string) (uint64, error) {
	peersCount, err := GetPeersCount(discoveryPeer, channelName)
	if err != nil {
		return 0, err
	}
	ordererCount, err := GetOrdererCount(context.Background(), conn, mspID, cryptoPath, certPath, keyPath, channelName)
	if err != nil {
		return 0, err
	}
	return peersCount + ordererCount, nil
}

// ===============chaincode
func GetChaincodePeer(conn *grpc.ClientConn, mspID, cryptoPath, certPath, keyPath string) (*chaincode.Peer, error) {
	id, err := identity.NewPrivateKeySigningIdentity(mspID, readCertificate(certPath), readPrivateKey(keyPath))
	if err != nil {
		return nil, fmt.Errorf("error in getpeer %w", err)
	}
	return chaincode.NewPeer(conn, id), nil
}

func GetChaincodeGateway(conn *grpc.ClientConn, mspID, cryptoPath, certPath, keyPath string) *chaincode.Gateway {
	id, err := identity.NewPrivateKeySigningIdentity(mspID, readCertificate(certPath), readPrivateKey(keyPath))
	if err != nil {
		panic(err)
	}
	return chaincode.NewGateway(conn, id)
}

func GetChaincodeCount(chaincodePeer *chaincode.Peer) (uint64, error) {
	v, err := chaincodePeer.QueryInstalled(context.Background())
	if err != nil {
		return 0, fmt.Errorf("error in QueryInstalled,%w", err)
	}
	return uint64(len(v.InstalledChaincodes)), nil
}

// =============辅助函数
func readCertificate(certPath string) *x509.Certificate {
	certificatePEM, err := readFirstFile(certPath)
	if err != nil {
		panic(err)
	}
	block, _ := pem.Decode([]byte(certificatePEM))
	if block == nil {
		panic("failed to parse certificate PEM")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic("failed to parse certificate: " + err.Error())
	}

	return certificate
}

func readPrivateKey(keyPath string) crypto.PrivateKey {
	privateKeyPEM, err := readFirstFile(keyPath)
	if err != nil {
		panic(err)
	}

	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		panic("failed to parse private key PEM")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		panic("failed to parse PKCS8 encoded private key: " + err.Error())
	}

	return privateKey
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
