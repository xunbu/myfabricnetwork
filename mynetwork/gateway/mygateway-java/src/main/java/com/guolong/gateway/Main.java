package com.guolong.gateway;

import com.guolong.gateway.config.FabricGatewayFactory;
import com.guolong.gateway.dto.BlockInfo;
import com.guolong.gateway.dto.KeyHistory;
import com.guolong.gateway.dto.RichPageResult;
import com.guolong.gateway.dto.TxInfo;
import com.guolong.gateway.service.ChaincodeService;
import com.guolong.gateway.service.LedgerService;
import io.grpc.ManagedChannel;
import org.hyperledger.fabric.client.Gateway;

import java.nio.file.Paths;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;

public class Main {

    // =========================================================================
    // 配置部分：请根据你的实际环境（Fabric Test Network 或 生产环境）修改以下路径
    // =========================================================================
    
    // 组织 MSP ID
    private static final String MSP_ID = "Org1MSP";
    
    // 通道与链码名称
    private static final String CHANNEL_NAME = "mychannel";
    private static final String CHAINCODE_NAME = "basic";

    // Peer 节点连接信息
    private static final String PEER_ENDPOINT = "localhost:7051";
    private static final String OVERRIDE_AUTH = "peer0.guolong.com";

    // 证书路径 (这里以 fabric-samples/test-network 为例，请修改为你的绝对路径)
    private static final String CRYPTO_PATH = "/home/qinhan/fabric/mynetwork/organizations/peerOrganizations/guolong.com";
    private static final String CERT_PATH = CRYPTO_PATH + "/users/User1@guolong.com/msp/signcerts";
    private static final String KEY_PATH = CRYPTO_PATH + "/users/User1@guolong.com/msp/keystore";
    private static final String TLS_CERT_PATH = CRYPTO_PATH + "/peers/peer0.guolong.com/tls/ca.crt";

    public static void main(String[] args) {
        ManagedChannel channel = null;
        Gateway gateway = null;

        try {
            System.out.println(">>> 1. 正在初始化 gRPC 连接...");
            channel = FabricGatewayFactory.newGrpcConnection(TLS_CERT_PATH, PEER_ENDPOINT, OVERRIDE_AUTH);

            System.out.println(">>> 2. 正在连接 Fabric Gateway...");
            gateway = FabricGatewayFactory.createGateway(channel, MSP_ID, CERT_PATH, KEY_PATH);

            // 初始化服务
            ChaincodeService chaincodeService = new ChaincodeService(gateway);
            LedgerService ledgerService = new LedgerService(gateway);

            // ==========================================
            // A. 业务链码测试 (Put/Get/History)
            // ==========================================
            System.out.println("\n========== 业务链码测试 ==========");
            
            String testKey = "asset_" + System.currentTimeMillis();
            
            // 1. 上链 (PutValue)
            System.out.printf("正在写入数据 Key: %s...%n", testKey);
            chaincodeService.putValue(CHANNEL_NAME, CHAINCODE_NAME, testKey, "{\"color\":\"red\",\"size\":10}");
            System.out.println("写入成功");

            // 2. 查询 (GetValue)
            System.out.print("正在查询数据: ");
            String value = chaincodeService.getValue(CHANNEL_NAME, CHAINCODE_NAME, testKey);
            System.out.println("查询结果: " + value);

            // 3. 写入 Map (PutMap)
            Map<String, Object> mapData = new HashMap<>();
            mapData.put("name", "GatewayJava");
            mapData.put("version", 1.0);
            chaincodeService.putMap(CHANNEL_NAME, CHAINCODE_NAME, testKey, mapData);
            System.out.println("Map 数据更新成功");

            // 4. 获取历史 (KeyHistory)
            List<KeyHistory> history = chaincodeService.getKeyHistory(CHANNEL_NAME, CHAINCODE_NAME, testKey);
            System.out.printf("Key历史记录条数: %d%n", history.size());
            history.forEach(h -> System.out.printf(" - TxId: %s, IsDelete: %s, Value: %s%n", h.getTxId(), h.getIsDelete(), h.getValue()));

            // 5. 分页查询 (Pagination)
            RichPageResult pageResult = chaincodeService.getAllDataByPageWithLimit(CHANNEL_NAME, CHAINCODE_NAME, 1, 10);
            System.out.printf("分页查询第1页，共 %d 条数据%n", pageResult.getResults().size());

            // ==========================================
            // B. 系统账本测试 (Ledger/QSCC)
            // ==========================================
            System.out.println("\n========== 系统账本测试 (QSCC/CSCC) ==========");

            // 1. 获取区块高度
            long height = ledgerService.getBlockHeight(CHANNEL_NAME);
            System.out.println("当前区块高度: " + height);

            // 2. 获取交易总数
            long txCount = ledgerService.getTransactionCount(CHANNEL_NAME);
            System.out.println("通道总交易数: " + txCount);

            // 3. 获取最新区块详情
            if (height > 0) {
                BlockInfo blockInfo = ledgerService.getBlockByNum(CHANNEL_NAME, height - 1);
                System.out.printf("最新区块 [#%d] Hash: %s...%n", blockInfo.getBlockNumber(), blockInfo.getBlockHash().substring(0, 10));
                
                // 4. 获取该区块内的第一笔交易详情 (如果有)
                // 注意：需要确保 BlockInfo 在 DTO 转换时填充了 txIds，或者我们直接拿刚刚写入的交易ID来查
                String lastTxId = history.get(0).getTxId(); // 获取最近一次操作的 TxId
                System.out.println("正在查询交易详情 TxId: " + lastTxId);
                TxInfo txInfo = ledgerService.getTxById(CHANNEL_NAME, lastTxId);
                System.out.println("交易详情: " + txInfo);
            }

            // 5. 获取组织数量
            int orgCount = ledgerService.getOrganizationCount(CHANNEL_NAME);
            System.out.println("通道组织数量: " + orgCount);

        } catch (Exception e) {
            System.err.println("发生错误: " + e.getMessage());
            e.printStackTrace();
        } finally {
            // 资源清理
            if (gateway != null) {
                gateway.close();
            }
            if (channel != null) {
                try {
                    channel.shutdownNow().awaitTermination(5, TimeUnit.SECONDS);
                } catch (InterruptedException e) {
                    // ignore
                }
            }
            System.out.println("\n>>> 程序结束");
        }
    }
}