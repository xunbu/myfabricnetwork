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
	"guolong.com/fabric-restful-api/docker"
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
	channelName := "mychannel"
	ChaincodeName := "basic"

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

	chaincodeGateway, err := admin.GetChaincodeGateway(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		panic(err)
	}
	chaincodePeer, err := admin.GetChaincodePeer(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		panic(err)
	}
	discoveryPeer, err := admin.GetDiscoveryPeer(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		panic(err)
	}

	containerNames := []string{"peer0.guolong.com", "peer1.guolong.com", "peer2.guolong.com", "orderer.guolong.com"}

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
		c.Set("chaincodeName", ChaincodeName)
		c.Set("chaincodeGateway", chaincodeGateway)
		c.Set("chaincodePeer", chaincodePeer)
		c.Set("discoveryPeer", discoveryPeer)
		c.Next()
	})

	r.GET("/valuechain", getValueChainInfo)
	r.GET("/valuechain/getBlockByPage", getBlockListByPage)
	r.GET("/valuechain/getBlockByNum", getBlockByNum)
	r.GET("/valuechain/cpuHistory", getCpuHistory)
	r.GET("/valuechain/memoryHistory", getMemoryHistory)
	r.GET("/valuechain/getAllData", getAllData)
	r.GET("/valuechain/getKeyHistory", getKeyHistory)
	r.GET("/valuechain/getTxByID", getTxByID)
	r.GET("/valuechain/putValue", PutValue)
	// 默认端口 8080 启动服务器
	// 监听 0.0.0.0:8080（Windows 下为 localhost:8080）
	r.Run()
}
func getValueChainInfo(c *gin.Context) {

	conn := c.MustGet("connection").(*grpc.ClientConn)
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodeGateway := c.MustGet("chaincodeGateway").(*chaincode.Gateway)
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

	response["chainCodeCount"], err = admin.GetChaincodeCount(chaincodeGateway, channelName)
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

func getAllData(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodeName := c.MustGet("chaincodeName").(string)
	// func GetAllData(gw *client.Gateway, channelName string) ([]chaincode.QueryRichResult, error) {
	// 	v, err := EvaluateTransaction(gw, channelName, "basic", "QueryByRange", "", "")
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	var results []chaincode.QueryRichResult
	// 	err = json.Unmarshal(v, &results)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	return results, nil
	// }
	// 	type QueryRichResult struct {
	// 	Key    string      `json:"key"`
	// 	Value  interface{} `json:"value"`//Value有string和json([]byte)两种
	// 	IsJSON bool        `json:"isJson"`
	// }
	v, err := gateway.GetAllData(gw, channelName, chaincodeName)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, v)

}

func getKeyHistory(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodeName := c.MustGet("chaincodeName").(string)
	key := c.Query("key")
	// 	type KeyHistory struct {
	// 	TxId      string    `json:"txId"`
	// 	Timestamp time.Time `json:"timestamp"`
	// 	IsDelete  bool      `json:"isDelete"`
	// 	Value     []byte    `json:"value"`//应该显示string(Value)，string后为string或json
	// }
	// func gateway.GetKeyHistory(gw *client.Gateway, channelName string, chaincodeName string, key string) (*[]chaincode.KeyHistory, error)
	keyHistory, err := gateway.GetKeyHistory(gw, channelName, chaincodeName, key)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, keyHistory)
}

func getTxByID(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	TxID := c.Query("TxID")
	// 	type Txinfo struct {
	// 	TxId      string
	// 	ChannelId string
	// 	Type      string
	// 	Timestamp time.Time

	// 	Size           int
	// 	ValidationCode int
	// 	ChainCodeInfos []*ChainCodeInfo
	// }

	// type ChainCodeInfo struct {
	// 	ChainCodeName string
	// 	Args          []string
	// }

	// func (txInfo *Txinfo) String() string {
	// 	return fmt.Sprintf("TxId:%v\nChannelId:%v\nType:%v\nTimestamp:%v\nValidationCode:%v\nChainCodeInfos:\n%v\n", txInfo.TxId, txInfo.ChannelId, txInfo.Type, txInfo.Timestamp, txInfo.ValidationCode, txInfo.ChainCodeInfos)
	// }

	// func (chainCodeInfo *ChainCodeInfo) String() string {
	// 	return fmt.Sprintf("ChainCodeName:%v,args:%v", chainCodeInfo.ChainCodeName, chainCodeInfo.Args)
	// }
	Txinfo, err := gateway.GetTxByID(gw, channelName, TxID)
	if err != nil {
		return
	}
	c.JSON(http.StatusOK, Txinfo)
}

func PutValue(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodeName := c.MustGet("chaincodeName").(string)
	key := c.Query("key")
	value := c.Query("value")

	_, err := gateway.PutString(gw, channelName, chaincodeName, key, value)
	if err != nil {
		return
	}
	c.Status(http.StatusOK)
}
