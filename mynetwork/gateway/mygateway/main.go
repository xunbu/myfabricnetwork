package main

import (
	"fmt"

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
	// m := map[string]any{
	// 	"first name":  "tom",
	// 	"second name": "hanks",
	// }

	// v, err := gateway.PutJson(gw, channelName, "test", m)
	// gateway.PutString(gw, channelName, "author", "xunbu")
	// v, err := gateway.EvaluateTransaction(gw, channelName, "basic", "QueryByKey", "test")
	// if err != nil {
	// 	fmt.Printf("error in EvaluateTransaction %v", err)
	// }
	// fmt.Printf("value:%s\n", v)
	v, _ := gateway.GetTxByID(gw, channelName, "287d49979e733fc0c528b58804ef3f1a29fe1bb47a530b949915e4efdea022d9")
	fmt.Println(v)
	// v, _ := gateway.EvaluateTransaction(gw, channelName, "basic", "GetKeyHistory", "author")
	// kh := &[]chaincode.KeyHistory{}
	// json.Unmarshal(v, &kh)
	// for _, v2 := range *kh {
	// 	fmt.Printf("交易号:%s,值%s,时间戳%s\n", v2.TxId, v2.Value, v2.Timestamp)
	// }

	// 以下为admin-sdk
	// peer, err := admin.GetDiscoveryPeer(clientConnection, mspID, cryptoPath, certPath, keyPath)
	// if err != nil {
	// 	panic(err)
	// }
	// v, err := peer.QueryInstalled(context.Background())
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Print(len(v.InstalledChaincodes))
	// v, err := peer.PeerMembershipQuery(context.Background(), channelName, nil)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("%v\n", len(v.PeersByOrg))
	// v, err := admin.GetOrdererCount(context.Background(), clientConnection, mspID, cryptoPath, certPath, keyPath, channelName)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(v)
	// v, err := admin.GetOrdererCount(context.Background(), clientConnection, mspID, cryptoPath, certPath, keyPath, channelName)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(v)
	// v, err := gateway.GetAllStates(gw, channelName)
	// if err != nil {
	// 	panic(err)
	// }

	// v2, err := json.Marshal(v)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(string(v2))

	// v3, err := gateway.EvaluateTransaction(gw, channelName, "basic", "QueryByKeyAsBytes", "author")
	// if err != nil {
	// 	panic(err)
	// }
	// var v string
	// err = json.Unmarshal(v3, &v)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(string(v3))
}
