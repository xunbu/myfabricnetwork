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

	// [优化] 启动全量区块监控 (后台维护内存缓存)
	if err := gateway.StartChainMonitor(gw, channelName); err != nil {
		log.Printf("启动区块链监控失败: %v", err)
	}

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

	// 4. 定义 API 路由
	v1 := r.Group("/valuechain")
	{
		// 概览信息 (内存直读)
		v1.GET("/info", getValueChainInfo)

		// [优化] 趋势数据 (内存直读)
		v1.GET("/trend", getTrendData)

		v1.GET("/blocks", getBlockListByPage)
		v1.GET("/block", getBlockByNum)
		// 修正：这里调用本地实现的 Handler 函数
		v1.GET("/block/transactions", getBlockTransactionsByPage)
		v1.GET("/cpuHistory", getCpuHistory)
		v1.GET("/memoryHistory", getMemoryHistory)
		v1.GET("/data/all", GetAllDataByPageWithLimit)
		v1.GET("/data/history", getKeyHistory)
		v1.GET("/transaction", getTxByID)

		v1.POST("/data", putValue)
		v1.POST("/data/delete", deleteKey)
	}

	log.Printf("服务器正在启动，监听端口: %s", serverPort)
	if err := r.Run(":" + serverPort); err != nil {
		log.Fatalf("启动 Gin 服务器失败: %v", err)
	}
}

// --- Handler Functions ---

func getValueChainInfo(c *gin.Context) {
	chaincodeGateway := c.MustGet("chaincodeGateway").(*chaincode.Gateway)
	discoveryPeer := c.MustGet("discoveryPeer").(*discovery.Peer)
	clientConnection := c.MustGet("connection").(*grpc.ClientConn)
	channelName := c.MustGet("channelName").(string)

	response := gin.H{}
	var err error

	// 从缓存获取核心指标 (O(1))
	stats := gateway.GetChainStats()

	response["blockHeight"] = stats.Height
	response["totalTransactionCount"] = stats.TxCount
	response["orgCount"] = stats.OrgCount
	response["lastBlockTime"] = stats.LastBlockTime

	// 其它低频数据 (容错处理)
	response["chainCodeCount"], err = admin.GetChaincodeCount(chaincodeGateway, channelName)
	if err != nil {
		response["chainCodeCount"] = 0
	}
	response["nodeCount"], err = admin.GetNodesCount(discoveryPeer, channelName, context.Background(), clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		response["nodeCount"] = 0
	}

	c.JSON(http.StatusOK, response)
}

// [超级优化] 直接从内存返回，0 网络开销
func getTrendData(c *gin.Context) {
	trendData := gateway.GetTrendData()

	// 构造成 BlockPage 格式返回给前端
	response := &gateway.BlockPage{
		Results:  trendData,
		Page:     0,
		PageSize: len(trendData),
		Total:    len(trendData),
	}

	c.JSON(http.StatusOK, response)
}

func getBlockListByPage(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	pageNumStr := c.DefaultQuery("pageNum", "0")
	pageNum, _ := strconv.ParseUint(pageNumStr, 10, 64)
	pageSizeStr := c.DefaultQuery("pageSize", "10")
	pageSize, _ := strconv.ParseUint(pageSizeStr, 10, 64)

	// 列表页包含交易详情
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
	blockNum, _ := strconv.ParseUint(blockNumStr, 10, 64)

	response, err := gateway.GetBlockByNum(gw, channelName, blockNum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取区块详情失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, response)
}

// getBlockTransactionsByPage 是根据区块号分页获取该区块内交易的 Handler
func getBlockTransactionsByPage(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)

	// 1. 获取 blockNum
	blockNumStr := c.Query("blockNum")
	if blockNumStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数 blockNum 不能为空"})
		return
	}
	blockNum, err := strconv.ParseUint(blockNumStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 blockNum 格式"})
		return
	}

	// 2. 获取分页参数
	pageNumStr := c.DefaultQuery("pageNum", "0")
	pageNum, _ := strconv.Atoi(pageNumStr)

	pageSizeStr := c.DefaultQuery("pageSize", "10")
	pageSize, _ := strconv.Atoi(pageSizeStr)

	// 3. 调用 Gateway 逻辑
	response, err := gateway.GetBlockTransactionsByPage(gw, channelName, blockNum, pageNum, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取区块交易失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

func getCpuHistory(c *gin.Context) {
	c.JSON(http.StatusOK, docker.GetAllCPUHistory())
}

func getMemoryHistory(c *gin.Context) {
	c.JSON(http.StatusOK, docker.GetAllMemoryHistory())
}

func GetAllDataByPageWithLimit(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodeName := c.MustGet("chaincodeName").(string)
	pageNumStr := c.DefaultQuery("pageNum", "0")
	pageNum, _ := strconv.ParseUint(pageNumStr, 10, 64)
	pageSizeStr := c.DefaultQuery("pageSize", "10")
	pageSize, _ := strconv.ParseUint(pageSizeStr, 10, 64)

	v, err := gateway.GetAllDataByPageWithLimit(gw, channelName, chaincodeName, int(pageNum), int(pageSize))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取数据失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, v)
}

func getKeyHistory(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodeName := c.MustGet("chaincodeName").(string)
	key := c.Query("key")
	keyHistory, err := gateway.GetKeyHistory(gw, channelName, chaincodeName, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取历史失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, keyHistory)
}

func getTxByID(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	txID := c.Query("txID")
	txInfo, err := gateway.GetTxByID(gw, channelName, txID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取交易详情失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, txInfo)
}

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
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求: " + err.Error()})
		return
	}
	_, err := gateway.PutValue(gw, channelName, chaincodeName, req.Key, req.Value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

type DeleteKeyRequest struct {
	Key string `json:"key" binding:"required"`
}

func deleteKey(c *gin.Context) {
	gw := c.MustGet("gateway").(*client.Gateway)
	channelName := c.MustGet("channelName").(string)
	chaincodeName := c.MustGet("chaincodeName").(string)
	var req DeleteKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求: " + err.Error()})
		return
	}
	_, err := gateway.DeleteKey(gw, channelName, chaincodeName, req.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
