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
	"guolong.com/fabric-rest-api/docker"
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

	containerNames := []string{"peer0.guolong.com", "peer1.guolong.com", "peer2.guolong.com"}

	err = docker.GetCpuHistoryByContainerNames(containerNames)
	if err != nil {
		fmt.Printf("启动cpu监控失败: %v\n", err)
		return
	}
	err = docker.GetMemoryHistoryByContainerNames(containerNames)
	if err != nil {
		fmt.Printf("启动内存监控失败: %v\n", err)
		return
	}

	defer docker.StopMonitoring()
	defer docker.ClearAllHistory()

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
	r.GET("/valuechain/getBlockByNum", getBlockByNum)
	r.GET("/valuechain/cpuHistory", getCpuHistory)
	r.GET("/valuechain/memoryHistory", getMemoryHistory)
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

// func GetBlockByNum(gw *client.Gateway, channelName string, blockNum uint64) (*BlockInfo, error) {
func getBlockByNum(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	blockNumStr := c.DefaultQuery("blockNum", "0")
	blockNum, err := strconv.ParseUint(blockNumStr, 10, 64)
	if err != nil {
		fmt.Println("转换错误:", err)
		return
	}
	response, err := gateway.GetBlockByNum(gw, channelName, blockNum)
	if err != nil {
		fmt.Println("获取Block错误,%w", err)
		return
	}
	c.JSON(http.StatusOK, response)

}

func getCpuHistory(c *gin.Context) {
	//func docker.GetAllCPUHistory() map[string][]docker.CPUMetric
	//type CPUMetric struct {
	// 	Timestamp time.Time
	// 	CPUUsage  float64 (百分比数字，如5.0表示5%)
	// }
	// 默认每两秒获取一次数据，最多一个容器存储1000条cpu记录
	allHistory := docker.GetAllCPUHistory()
	c.JSON(http.StatusOK, allHistory)
}

func getMemoryHistory(c *gin.Context) {
	// func docker.GetAllMemoryHistory() map[string][]docker.MemoryMetric
	// MemoryMetric 存储单个时间点的内存使用情况
	// type MemoryMetric struct {
	// 	Timestamp   time.Time
	// 	UsedMemory  uint64 // 已使用内存(字节)
	// 	TotalMemory uint64 // 总内存限制(字节)
	// }
	allHistory := docker.GetAllMemoryHistory()
	c.JSON(http.StatusOK, allHistory)
}
