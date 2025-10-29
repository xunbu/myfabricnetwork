package main

import (
	"fmt"

	"guolong.com/fabric-gateway/gateway"
)

const (
	mspID        = "Org1MSP"
	cryptoPath   = "../../organizations/peerOrganizations/guolong.com"
	certPath     = cryptoPath + "/users/User1@guolong.com/msp/signcerts"
	keyPath      = cryptoPath + "/users/User1@guolong.com/msp/keystore"
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
	gateway.GetTransactionCount(gw, channelName)
	v, err := gateway.EvaluateTransaction(gw, channelName, "basic", "GetAllAssets")
	if err != nil {
		fmt.Println("error in EvaluateTransaction")
	}
	fmt.Printf("value:%v", v)
}
