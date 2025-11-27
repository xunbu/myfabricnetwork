package main

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-admin-sdk/pkg/chaincode"
	"github.com/hyperledger/fabric-admin-sdk/pkg/discovery"
	"github.com/hyperledger/fabric-gateway/pkg/client"
	"google.golang.org/grpc"
	"guolong.com/fabric-gateway/admin"
	"guolong.com/fabric-gateway/gateway"
	"guolong.com/fabric-restapi/docker"
)

// --- 配置信息 ---
const (
	mspID         = "Org1MSP"
	cryptoPath    = "../../organizations/peerOrganizations/guolong.com"
	certPath      = cryptoPath + "/users/Admin@guolong.com/msp/signcerts"
	keyPath       = cryptoPath + "/users/Admin@guolong.com/msp/keystore"
	tlsCertPath   = cryptoPath + "/peers/peer0.guolong.com/tls/ca.crt"
	peerEndpoint  = "dns:///localhost:7051"
	gatewayPeer   = "peer0.guolong.com"
	channelName   = "mychannel"
	chaincodeName = "basic"
	serverPort    = "8081"
)

func main() {
	// 1. 初始化 Fabric 连接
	clientConnection, err := gateway.NewGrpcConnection(tlsCertPath, gatewayPeer, peerEndpoint)
	if err != nil {
		log.Fatalf("创建 gRPC 连接失败: %v", err)
	}
	defer clientConnection.Close()

	gw, err := gateway.GetGateway(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		log.Fatalf("获取 Gateway 实例失败: %v", err)
	}
	defer gw.Close()

	chaincodeGateway, err := admin.GetChaincodeGateway(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		log.Fatalf("获取 Chaincode Gateway 实例失败: %v", err)
	}

	discoveryPeer, err := admin.GetDiscoveryPeer(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		log.Fatalf("获取 Discovery Peer 实例失败: %v", err)
	}

	// 2. 初始化 Docker 监控
	containerNames := []string{"peer0.guolong.com", "peer1.guolong.com", "peer2.guolong.com", "orderer.guolong.com"}
	if err := docker.GetCpuHistoryByContainerNames(containerNames); err != nil {
		log.Printf("启动 CPU 监控失败: %v\n", err)
	}
	if err := docker.GetMemoryHistoryByContainerNames(containerNames); err != nil {
		log.Printf("启动内存监控失败: %v\n", err)
	}
	defer docker.StopMonitoring()
	defer docker.ClearAllHistory()

	// 3. 设置 Gin 路由器
	r := gin.Default()

	// 静态文件服务
	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	// 使用中间件注入所有共享的实例和配置
	r.Use(func(c *gin.Context) {
		c.Set("gateway", gw)
		c.Set("chaincodeGateway", chaincodeGateway)
		c.Set("discoveryPeer", discoveryPeer)
		c.Set("connection", clientConnection)
		c.Set("channelName", channelName)
		c.Set("chaincodeName", chaincodeName)
		c.Next()
	})

	// 4. 定义 API 路由 (仅使用 GET 和 POST)
	v1 := r.Group("/valuechain")
	{
		// 查询类接口 (GET)
		v1.GET("/info", getValueChainInfo)
		v1.GET("/blocks", getBlockListByPage)
		v1.GET("/block", getBlockByNum)
		v1.GET("/cpuHistory", getCpuHistory)
		v1.GET("/memoryHistory", getMemoryHistory)
		v1.GET("/data/all", getAllData)
		v1.GET("/data/history", getKeyHistory)
		v1.GET("/transaction", getTxByID)

		// 写入类接口 (POST)
		// 数据通过 JSON 请求体传递: {"key": "someKey", "value": "someValue"}
		v1.POST("/data", putValue)

		// 删除类接口 (POST)
		// 数据通过 JSON 请求体传递: {"key": "key_to_delete"}
		v1.POST("/data/delete", deleteKey)
	}

	// 5. 启动服务器
	log.Printf("服务器正在启动，监听端口: %s", serverPort)
	if err := r.Run(":" + serverPort); err != nil {
		log.Fatalf("启动 Gin 服务器失败: %v", err)
	}
}

// --- Handler Functions ---

func getValueChainInfo(c *gin.Context) {
	// 从上下文中获取依赖项
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodeGateway := c.MustGet("chaincodeGateway").(*chaincode.Gateway)
	discoveryPeer := c.MustGet("discoveryPeer").(*discovery.Peer)
	clientConnection := c.MustGet("connection").(*grpc.ClientConn)

	response := gin.H{}
	var err error

	response["blockHeight"], err = gateway.GetBlockHeight(gw, channelName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取区块高度失败: " + err.Error()})
		return
	}

	response["totalTransactionCount"], err = gateway.GetTransactionCount(gw, channelName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取交易总数失败: " + err.Error()})
		return
	}

	response["orgCount"], err = gateway.GetOrganizationCount(gw, channelName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取组织数量失败: " + err.Error()})
		return
	}

	response["chainCodeCount"], err = admin.GetChaincodeCount(chaincodeGateway, channelName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取链码数量失败: " + err.Error()})
		return
	}

	response["nodeCount"], err = admin.GetNodesCount(discoveryPeer, channelName, context.Background(), clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取节点数量失败: " + err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 pageNum 参数"})
		return
	}

	pageSizeStr := c.DefaultQuery("pageSize", "10")
	pageSize, err := strconv.ParseUint(pageSizeStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 pageSize 参数"})
		return
	}

	response, err := gateway.GetBlockListByPage(gw, channelName, pageNum, pageSize, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取区块列表失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}

func getBlockByNum(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)

	blockNumStr := c.Query("blockNum")
	if blockNumStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 blockNum 参数"})
		return
	}

	blockNum, err := strconv.ParseUint(blockNumStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 blockNum 参数"})
		return
	}

	response, err := gateway.GetBlockByNum(gw, channelName, blockNum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取区块详情失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}

func getCpuHistory(c *gin.Context) {
	allHistory := docker.GetAllCPUHistory()
	c.JSON(http.StatusOK, allHistory)
}

func getMemoryHistory(c *gin.Context) {
	allHistory := docker.GetAllMemoryHistory()
	c.JSON(http.StatusOK, allHistory)
}

func getAllData(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodeName := c.MustGet("chaincodeName").(string)

	v, err := gateway.GetAllData(gw, channelName, chaincodeName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取所有数据失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, v)
}

func getKeyHistory(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodeName := c.MustGet("chaincodeName").(string)

	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 key 参数"})
		return
	}

	keyHistory, err := gateway.GetKeyHistory(gw, channelName, chaincodeName, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取 key 历史记录失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, keyHistory)
}

func getTxByID(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)

	txID := c.Query("txID")
	if txID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 txID 参数"})
		return
	}

	txInfo, err := gateway.GetTxByID(gw, channelName, txID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取交易详情失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, txInfo)
}

// 定义用于接收 POST (PutValue) 请求体的结构体
type PutValueRequest struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

func putValue(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodeName := c.MustGet("chaincodeName").(string)

	var req PutValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体: " + err.Error()})
		return
	}

	_, err := gateway.PutValue(gw, channelName, chaincodeName, req.Key, req.Value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入数据失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// 定义用于接收 POST (DeleteKey) 请求体的结构体
type DeleteKeyRequest struct {
	Key string `json:"key" binding:"required"`
}

func deleteKey(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodeName := c.MustGet("chaincodeName").(string)

	var req DeleteKeyRequest
	// 从 JSON 请求体中绑定 key
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求体，需要 { \"key\": \"...\" }: " + err.Error()})
		return
	}

	_, err := gateway.DeleteKey(gw, channelName, chaincodeName, req.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除数据失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
