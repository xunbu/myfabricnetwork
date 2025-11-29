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
	// v2, _ := gateway.GetTransactionCount(gw, channelName)
	// fmt.Println(v2)
	// m := map[string]any{
	// 	"first name":  "tom",
	// 	"second name": "hanks",
	// }
	// m2 := map[string]any{
	// 	"first name":  "tom",
	// 	"second name": "Janks",
	// }
	// // gateway.PutMap(gw, channelName, "basic", "test", m)
	// // gateway.PutMap(gw, channelName, "basic", "test", m2)
	// // gateway.PutMap(gw, channelName, "basic", "test2", m2)

	// v := make(map[string]string)
	// t, err := json.Marshal(m)
	// if err != nil {
	// 	panic(err)
	// }
	// t2, err := json.Marshal(m2)
	// if err != nil {
	// 	panic(err)
	// }
	// v["test3"] = string(t)
	// v["test4"] = string(t2)
	// fmt.Println(v)
	// gateway.PutKVs(gw, channelName, channelName, v)
	// gateway.PutValue(gw, channelName, "basic", "author", "xunbu")
	// v, err := gateway.EvaluateTransaction(gw, channelName, "basic", "QueryByKey", "author")
	// if err != nil {
	// 	fmt.Printf("error in EvaluateTransaction %v", err)
	// }
	// fmt.Printf("value:%s\n", v)
	// v, _ := gateway.GetTxByID(gw, channelName, "287d49979e733fc0c528b58804ef3f1a29fe1bb47a530b949915e4efdea022d9")
	// fmt.Println(v)

	// v, _ := gateway.GetKeyHistory(gw, channelName, "basic", "author")
	// for _, v2 := range *v {
	// 	fmt.Printf("交易号:%s,值:%s,时间戳%s\n", v2.TxId, v2.Value, v2.Timestamp)
	// }

	// gateway.DeleteKey(gw, channelName, "basic", "author")
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
	v, err := gateway.GetAllData(gw, channelName, "basic")
	if err != nil {
		panic(err)
	}
	fmt.Println(v)

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
