// https://pkg.go.dev/github.com/hyperledger/fabric-admin-sdk/pkg
package admin

import (
	"context"
	"fmt"

	"github.com/hyperledger/fabric-admin-sdk/pkg/chaincode"
	"github.com/hyperledger/fabric-admin-sdk/pkg/channel"
	"github.com/hyperledger/fabric-admin-sdk/pkg/discovery"
	"github.com/hyperledger/fabric-admin-sdk/pkg/identity"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"google.golang.org/grpc"
)

// ================channel
func GetConfigBlock(ctx context.Context, conn *grpc.ClientConn, mspID, cryptoPath, certPath, keyPath string, channelName string) (*common.Block, error) {
	id, err := identity.NewPrivateKeySigningIdentity(mspID, readCertificate(certPath), readPrivateKey(keyPath))
	if err != nil {
		return nil, fmt.Errorf("error in get peer %w", err)
	}
	return channel.GetConfigBlock(ctx, conn, id, channelName)
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
		return nil, fmt.Errorf("error in get peer %w", err)
	}
	return chaincode.NewPeer(conn, id), nil
}

func GetChaincodeGateway(conn *grpc.ClientConn, mspID, cryptoPath, certPath, keyPath string) (*chaincode.Gateway, error) {
	id, err := identity.NewPrivateKeySigningIdentity(mspID, readCertificate(certPath), readPrivateKey(keyPath))
	if err != nil {
		return nil, fmt.Errorf("error in get gateway %w", err)
	}
	return chaincode.NewGateway(conn, id), nil
}

func GetChaincodeCount(chaincodeGateway *chaincode.Gateway, channelName string) (uint64, error) {
	v, err := chaincodeGateway.QueryCommitted(context.Background(), channelName)
	if err != nil {
		return 0, fmt.Errorf("error in QueryInstalled,%w", err)
	}
	return uint64(len(v.ChaincodeDefinitions)), nil
}

// func GetChaincodeInstalledCount(chaincodePeer *chaincode.Peer) (uint64, error) {
// 	v, err := chaincodePeer.QueryInstalled(context.Background())
// 	if err != nil {
// 		return 0, fmt.Errorf("error in QueryInstalled,%w", err)
// 	}
// 	return uint64(len(v.InstalledChaincodes)), nil
// }
