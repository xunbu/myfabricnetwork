package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-admin-sdk/pkg/chaincode"
	"github.com/hyperledger/fabric-gateway/pkg/client"
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
	peer, err := admin.GetPeer(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		panic(err)
	}
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Set("gateway", gw)
		c.Set("channelName", channelName)
		c.Set("peer", peer)
		c.Next()
	})

	r.GET("/valuechain", getValueChainInfo)

	// 默认端口 8080 启动服务器
	// 监听 0.0.0.0:8080（Windows 下为 localhost:8080）
	r.Run()
}
func getValueChainInfo(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	peer := c.MustGet("peer").(*chaincode.Peer)
	response := gin.H{}
	var err error

	response["blockHeight"], err = gateway.GetBlockHeight(gw, channelName)
	if err != nil {
		c.Error(err)
		return
	}

	response["totalTransactionCount"], err = gateway.GetTransactionCount(gw, channelName)
	if err != nil {
		c.Error(err)
		return
	}

	response["orgCount"], err = gateway.GetOrganizationCount(gw, channelName)
	if err != nil {
		c.Error(err)
		return
	}
	response["chainCodeCount"], err = admin.GetChaincodeCount(peer)
	if err != nil {
		c.Error(err)
		return
	}

	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, response)
}
