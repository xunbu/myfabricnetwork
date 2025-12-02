package main

import (
	"fmt"
	"log"
	"time"

	"guolong.com/fabric-gateway/gateway"
)

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

	// [新增] 数据库配置 (必须与 restapi 中或者实际环境保持一致)
	mysqlDSN = "root:password@tcp(127.0.0.1:3306)/fabric_data?charset=utf8mb4&parseTime=True&loc=Local"
)

func main() {
	log.Println("=== 开始测试 Fabric Gateway 功能 ===")

	// [新增] 0. 初始化数据库 (Monitor 依赖数据库)
	log.Println("0. 初始化数据库连接...")
	if err := gateway.InitStore(mysqlDSN); err != nil {
		log.Fatalf("❌ 初始化数据库失败: %v", err)
	}
	log.Println("✅ 数据库连接成功")

	// 1. 建立连接
	log.Println("1. 正在连接到 Fabric 网络...")
	clientConnection, err := gateway.NewGrpcConnection(tlsCertPath, gatewayPeer, peerEndpoint)
	if err != nil {
		log.Fatalf("❌ 连接失败: %v", err)
	}
	defer clientConnection.Close()

	gw, err := gateway.GetGateway(clientConnection, mspID, cryptoPath, certPath, keyPath)
	if err != nil {
		log.Fatalf("❌ 获取 Gateway 实例失败: %v", err)
	}
	defer gw.Close()
	log.Println("✅ 连接成功")

	// 2. 启动监控 (monitor.go)
	// 现在监控服务会检查 MySQL 中的创世块哈希，如果不对会自动重置 DB
	log.Println("2. 启动后台区块监控...")
	if err := gateway.StartChainMonitor(gw, channelName); err != nil {
		log.Fatalf("❌ 监控启动失败: %v", err)
	}
	// 给监控一点时间同步初始数据或断点续传
	time.Sleep(2 * time.Second)

	// 3. 测试内存缓存统计 (GetChainStats)
	stats := gateway.GetChainStats()
	fmt.Printf("   [内存缓存] 高度: %d, 交易总数: %d, 组织数: %d, 最新区块时间: %v\n",
		stats.Height, stats.TxCount, stats.OrgCount, stats.LastBlockTime)

	// 4. 链码交互测试: 写入数据 (PutValue)
	testKey := "test_monitor_key"
	testVal := fmt.Sprintf("value_created_at_%v", time.Now().Format(time.RFC3339))
	log.Printf("3. 测试写入数据 Key=%s, Val=%s", testKey, testVal)

	if _, err := gateway.PutValue(gw, channelName, chaincodeName, testKey, testVal); err != nil {
		log.Fatalf("❌ 写入数据失败: %v", err)
	}
	log.Println("✅ 数据写入提交成功 (等待2秒让区块生成并被监控捕获)...")
	time.Sleep(3 * time.Second) // 等待出块

	// 5. 再次检查监控数据 (验证是否自动更新)
	newStats := gateway.GetChainStats()
	fmt.Printf("   [更新后缓存] 高度: %d (+%d), 交易总数: %d\n",
		newStats.Height, newStats.Height-stats.Height, newStats.TxCount)

	// 6. 链码交互测试: 读取数据 (GetValue)
	log.Println("4. 测试读取数据...")
	val, err := gateway.GetValue(gw, channelName, chaincodeName, testKey)
	if err != nil {
		log.Printf("❌ 读取失败: %v", err)
	} else {
		log.Printf("✅ 读取成功: %s", val)
	}

	// 7. 账本查询: 获取最新区块详情 (GetBlockByNum)
	currentHeight := newStats.Height
	if currentHeight > 0 {
		log.Printf("5. 查询最新区块详情 (区块号: %d)...", currentHeight-1)
		blockInfo, err := gateway.GetBlockByNum(gw, channelName, currentHeight-1)
		if err != nil {
			log.Printf("❌ 获取区块失败: %v", err)
		} else {
			fmt.Printf("   区块哈希: %s\n", blockInfo.BlockHash)
			fmt.Printf("   交易数量: %d\n", blockInfo.TxCount)
			fmt.Printf("   交易ID列表: %v\n", blockInfo.TxIDs)

			// 8. 账本查询: 获取交易详情 (GetTxByID)
			if len(blockInfo.TxIDs) > 0 {
				txID := blockInfo.TxIDs[0]
				log.Printf("6. 查询交易详情 (TxID: %s)...", txID)
				txInfo, err := gateway.GetTxByID(gw, channelName, txID)
				if err != nil {
					log.Printf("❌ 获取交易详情失败: %v", err)
				} else {
					fmt.Printf("   交易时间: %v\n", txInfo.Timestamp)
					fmt.Printf("   验证码: %d (0=Valid)\n", txInfo.ValidationCode)
					if len(txInfo.ChainCodeInfos) > 0 {
						fmt.Printf("   调用链码: %s, 参数: %v\n",
							txInfo.ChainCodeInfos[0].ChainCodeName,
							txInfo.ChainCodeInfos[0].Args)
					}
				}
			}
		}
	}

	// 9. 测试趋势数据缓存 (GetTrendData)
	// [修改] GetTrendData 现在从数据库读取，需要传入通道名和限制数量
	log.Println("7. 测试趋势数据 (GetTrendData - 读数据库)...")
	trendData, err := gateway.GetTrendData(channelName, 10) // 获取最近10条
	if err != nil {
		log.Printf("❌ 获取趋势数据失败: %v", err)
	} else {
		fmt.Printf("   获取到的最近区块数量: %d\n", len(trendData))
		if len(trendData) > 0 {
			latest := trendData[0]
			fmt.Printf("   数据库中最新区块: #%d, Hash=%s...\n", latest.BlockNumber, latest.BlockHash[:10])
		}
	}

	// 10. 测试 Key 历史记录 (GetKeyHistory)
	log.Printf("8. 查询 Key 历史记录: %s", testKey)
	history, err := gateway.GetKeyHistory(gw, channelName, chaincodeName, testKey)
	if err != nil {
		log.Printf("❌ 获取历史失败: %v", err)
	} else {
		for i, h := range *history {
			fmt.Printf("   [%d] TxID: %s, Value: %s, IsDelete: %v\n", i, h.TxId, h.Value, h.IsDelete)
		}
	}

	// 11. 测试分页查询区块 (GetBlockListByPage)
	log.Println("9. 测试分页查询区块列表 (第0页, 5条)...")
	blockPage, err := gateway.GetBlockListByPage(gw, channelName, 0, 5, false, "DESC")
	if err != nil {
		log.Printf("❌ 分页查询失败: %v", err)
	} else {
		fmt.Printf("   获取到 %d 个区块, 总高度: %d\n", len(blockPage.Results), blockPage.Total)
	}

	// 12. 测试富查询 (GetAllData / QueryByRich)
	log.Println("10. 测试富查询 (获取所有数据)...")
	allData, err := gateway.GetAllData(gw, channelName, chaincodeName)
	if err != nil {
		log.Printf("❌ 富查询失败: %v", err)
	} else {
		fmt.Printf("   当前状态数据库记录数: %d\n", len(allData))
		if len(allData) > 0 {
			fmt.Printf("   第一条数据: Key=%s, Value=%s\n", allData[0].Key, string(allData[0].Value))
		}
	}

	// 13. 清理测试数据
	log.Println("11. 清理测试数据 (DeleteKey)...")
	_, err = gateway.DeleteKey(gw, channelName, chaincodeName, testKey)
	if err != nil {
		log.Printf("❌ 删除失败: %v", err)
	} else {
		log.Println("✅ 删除成功")
	}

	log.Println("=== 测试结束 ===")
}
