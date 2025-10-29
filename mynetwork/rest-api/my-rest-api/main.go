package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-admin-sdk/pkg/chaincode"
	"github.com/hyperledger/fabric-admin-sdk/pkg/discovery"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"google.golang.org/grpc"
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
	chaincodePeer, err := admin.GetChaincodePeer(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		panic(err)
	}
	discoveryPeer, err := admin.GetDiscoveryPeer(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})
	r.Use(func(c *gin.Context) {
		c.Set("connection", clientConnection)
		c.Set("gateway", gw)
		c.Set("channelName", channelName)
		c.Set("chaincodePeer", chaincodePeer)
		c.Set("discoveryPeer", discoveryPeer)
		c.Next()
	})

	r.GET("/valuechain", getValueChainInfo)
	r.GET("/valuechain/getBlockByPage", getBlockListByPage)

	// 默认端口 8080 启动服务器
	// 监听 0.0.0.0:8080（Windows 下为 localhost:8080）
	r.Run()
}
func getValueChainInfo(c *gin.Context) {

	conn := c.MustGet("connection").(*grpc.ClientConn)
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodePeer := c.MustGet("chaincodePeer").(*chaincode.Peer)
	discoveryPeer := c.MustGet("discoveryPeer").(*discovery.Peer)
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

	response["chainCodeCount"], err = admin.GetChaincodeCount(chaincodePeer)
	if err != nil {
		c.Error(err)
		return
	}

	response["nodeCount"], err = admin.GetNodesCount(discoveryPeer, channelName, context.Background(), conn, mspID, cryptoPath, certPath, keyPath)

	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response)
}

func getBlockListByPage(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	pageNumStr := c.DefaultQuery("pageNum", "0")
	pageNum, err := strconv.ParseUint(pageNumStr, 10, 64)
	if err != nil {
		fmt.Println("转换错误:", err)
		return
	}
	pageSizeStr := c.DefaultQuery("pageSize", "1")
	pageSize, err := strconv.ParseUint(pageSizeStr, 10, 64)
	if err != nil {
		fmt.Println("转换错误:", err)
		return
	}
	//func GetBlockListByPage(gw *client.Gateway, channelName string, pageNum uint64, pageSize uint64, includeTxDetails bool) ([]*BlockInfo, error)
	//type BlockInfo struct {
	// 	BlockHash    string    `json:"blockHash"`
	// 	PreviousHash string    `json:"previousHash"`
	// 	MerkleRoot   string    `json:"dataHash"`
	// 	BlockNumber  uint64    `json:"blockNumber"`
	// 	TxCount      uint64    `json:"txCount"`
	// 	BlockSize    int64     `json:"blockSize"`
	// 	Timestamp    time.Time `json:"timestamp"`
	// 	ChannelID    string    `json:"channelId"`
	// 	BlockCreator string    `json:"blockCreator,omitempty"` // 可选字段
	// 	TxIDs        []string  `json:"txIds,omitempty"`        // 可选字段，交易ID列表
	// }
	response, err := gateway.GetBlockListByPage(gw, channelName, pageNum, pageSize, true)
	if err != nil {
		fmt.Println("获取BlockList错误,%w", err)
		return
	}
	c.JSON(http.StatusOK, response)

}
