package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
	clientConnection := gateway.NewGrpcConnection(tlsCertPath, gatewayPeer, peerEndpoint)

	defer clientConnection.Close()
	gw, err := gateway.GetGateWay(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		panic(err)
	}

	defer gw.Close()
	channelName := "mychannel"

	r := gin.Default()
	r.GET("/valuechain", func(c *gin.Context) {
		// 返回 JSON 响应
		c.JSON(http.StatusOK, gin.H{
			"blockHeight": gateway.GetBlockHeight(gw, channelName),
			// "nodeCount":             gateway.GetNodeCount(gw, channelName),
			"totalTransactionCount": gateway.GetTransactionCount(gw, channelName),
			// "chaincodeNum":          gateway.GetChaincodeCount(gw, channelName),
			"orgCount": gateway.GetOrgCount(gw, channelName),
		})
	})

	// 默认端口 8080 启动服务器
	// 监听 0.0.0.0:8080（Windows 下为 localhost:8080）
	r.Run()
}
