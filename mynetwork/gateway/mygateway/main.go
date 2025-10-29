package main

import (
	"context"
	"fmt"

	"guolong.com/fabric-gateway/admin"
	"guolong.com/fabric-gateway/gateway"
)

const (
	mspID        = "Org1MSP"
	cryptoPath   = "../../organizations/peerOrganizations/guolong.com"
	certPath     = cryptoPath + "/users/Admin@guolong.com/msp/signcerts"
	keyPath      = cryptoPath + "/users/Admin@guolong.com/msp/keystore"
	tlsCertPath  = cryptoPath + "/peers/peer0.guolong.com/tls/ca.crt"
	peerEndpoint = "dns:///localhost:7051"
	gatewayPeer  = "peer0.guolong.com"
)

func main() {
	clientConnection, err := gateway.NewGrpcConnection(tlsCertPath, gatewayPeer, peerEndpoint)
	if err != nil {
		panic(err)
	}
	defer clientConnection.Close()

	gw, err := gateway.GetGateway(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		panic(err)
	}

	defer gw.Close()
	channelName := "mychannel"
	// gateway.GetTransactionCount(gw, channelName)
	// v, err := gateway.EvaluateTransaction(gw, channelName, "basic", "GetAllAssets")
	// if err != nil {
	// 	fmt.Println("error in EvaluateTransaction")
	// }
	// fmt.Printf("value:%v", v)

	// 以下为admin-sdk
	peer, err := admin.GetDiscoveryPeer(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		panic(err)
	}
	// v, err := peer.QueryInstalled(context.Background())
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Print(len(v.InstalledChaincodes))
	v, err := peer.PeerMembershipQuery(context.Background(), channelName, nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v\n", len(v.PeersByOrg))
}
