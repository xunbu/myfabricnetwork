package com.guolong.gateway;

import com.guolong.gateway.config.FabricGatewayFactory;
import com.guolong.gateway.dto.BlockInfo;
import com.guolong.gateway.dto.KeyHistory;
import com.guolong.gateway.dto.RichPageResult;
import com.guolong.gateway.dto.TxInfo;
import com.guolong.gateway.service.AdminService;
import com.guolong.gateway.service.ChaincodeService;
import com.guolong.gateway.service.LedgerService;
import com.guolong.gateway.utils.FileUtils;
import io.grpc.ManagedChannel;
import org.hyperledger.fabric.client.Gateway;
import org.hyperledger.fabric.protos.common.Block;

import java.nio.file.Path;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;

public class Main {

    // =========================================================================
    // 1. 基础网络配置
    // =========================================================================
    private static final String MSP_ID = "Org1MSP";
    private static final String CHANNEL_NAME = "mychannel";
    private static final String CHAINCODE_NAME = "basic";

    // 连接地址
    // 注意：CLI 工具(discover)对 TLS 校验非常严格。如果 /etc/hosts 没有配置映射，
    // discover 连接 localhost 可能会失败。建议配置 hosts: 127.0.0.1 peer0.guolong.com
    private static final String PEER_ENDPOINT = "localhost:7051";
    
    // 域名覆盖 (用于 TLS 握手时验证证书)
    private static final String OVERRIDE_AUTH = "peer0.guolong.com";

    // =========================================================================
    // 2. 证书路径配置 (请确保路径真实存在)
    // =========================================================================
    private static final String CRYPTO_PATH = "/home/qinhan/fabric/mynetwork/organizations/peerOrganizations/guolong.com";
    
    // 用户身份 (用于 SDK 连接和 CLI 调用)
    private static final String CERT_PATH = CRYPTO_PATH + "/users/User1@guolong.com/msp/signcerts";
    private static final String KEY_PATH = CRYPTO_PATH + "/users/User1@guolong.com/msp/keystore";
    
    // Peer 节点的 TLS CA 证书 (用于建立可信连接)
    private static final String TLS_CERT_PATH = CRYPTO_PATH + "/peers/peer0.guolong.com/tls/ca.crt";

    // [CLI 专用] MSP 文件夹路径 (指向 msp 目录，而不是 signcerts)
    private static final String MSP_CONFIG_PATH = CRYPTO_PATH + "/users/User1@guolong.com/msp";
    
    // [CLI 专用] Fabric Config 路径 (指向包含 core.yaml 的文件夹)
    // ⚠️ 请确认此路径下存在 core.yaml (通常在 fabric-samples/config)
    private static final String FABRIC_CFG_PATH = "/home/qinhan/fabric/mynetwork/config";

    public static void main(String[] args) {
        ManagedChannel channel = null;
        Gateway gateway = null;

        try {
            System.out.println(">>> 1. 正在初始化 gRPC 连接...");
            channel = FabricGatewayFactory.newGrpcConnection(TLS_CERT_PATH, PEER_ENDPOINT, OVERRIDE_AUTH);
            
            System.out.println(">>> 2. 正在连接 Fabric Gateway...");
            gateway = FabricGatewayFactory.createGateway(channel, MSP_ID, CERT_PATH, KEY_PATH);

            // 初始化所有服务
            ChaincodeService chaincodeService = new ChaincodeService(gateway);
            LedgerService ledgerService = new LedgerService(gateway);
            AdminService adminService = new AdminService(gateway);

            // // ==========================================
            // // A. 业务链码测试
            // // ==========================================
            // System.out.println("\n========== A. 业务链码测试 ==========");
            
            // String testKey = "asset_" + System.currentTimeMillis();
            
            // // 1. 上链
            // System.out.printf("正在写入数据 Key: %s...%n", testKey);
            // chaincodeService.putValue(CHANNEL_NAME, CHAINCODE_NAME, testKey, "{\"color\":\"red\",\"size\":10}");
            // System.out.println("写入成功");

            // // 2. 查询
            // System.out.print("正在查询数据: ");
            // String value = chaincodeService.getValue(CHANNEL_NAME, CHAINCODE_NAME, testKey);
            // System.out.println("查询结果: " + value);

            // // 3. Map写入
            // Map<String, Object> mapData = new HashMap<>();
            // mapData.put("name", "GatewayJava");
            // mapData.put("version", 1.0);
            // chaincodeService.putMap(CHANNEL_NAME, CHAINCODE_NAME, testKey, mapData);
            // System.out.println("Map 数据更新成功");

            // // 4. 历史记录
            // List<KeyHistory> history = chaincodeService.getKeyHistory(CHANNEL_NAME, CHAINCODE_NAME, testKey);
            // System.out.printf("Key历史记录条数: %d%n", history.size());

            // // 5. 分页查询
            // RichPageResult pageResult = chaincodeService.getAllDataByPageWithLimit(CHANNEL_NAME, CHAINCODE_NAME, 1, 5);
            // System.out.printf("分页查询第1页，共 %d 条数据%n", pageResult.getResults().size());

            // // ==========================================
            // // B. 系统账本测试 (QSCC/CSCC)
            // // ==========================================
            // System.out.println("\n========== B. 系统账本测试 (QSCC/CSCC) ==========");

            // // 1. 区块高度
            // long height = ledgerService.getBlockHeight(CHANNEL_NAME);
            // System.out.println("当前区块高度: " + height);
            
            // // 2. 最新区块详情
            // if (height > 0) {
            //     BlockInfo blockInfo = ledgerService.getBlockByNum(CHANNEL_NAME, height - 1);
            //     System.out.printf("最新区块 [#%d] Hash: %s...%n", blockInfo.getBlockNumber(), blockInfo.getBlockHash().substring(0, 10));
                
            //     // 3. 交易详情
            //     if (!history.isEmpty()) {
            //         String lastTxId = history.get(0).getTxId();
            //         System.out.println("正在查询交易详情 TxId: " + lastTxId);
            //         TxInfo txInfo = ledgerService.getTxById(CHANNEL_NAME, lastTxId);
            //         System.out.println("交易详情: " + txInfo);
            //     }
            // }

            // // 4. 组织数量
            // int orgCount = ledgerService.getOrganizationCount(CHANNEL_NAME);
            // System.out.println("通道组织数量: " + orgCount);

            // ==========================================
            // C. Admin 管理功能测试 (CLI + SDK)
            // ==========================================
            System.out.println("\n========== C. Admin 管理功能测试 ==========");

            // 1. 获取已安装链码数量 (CLI 实现 - peer lifecycle)
            System.out.print("1. 查询已提交链码数量... ");
            try {
                int ccCount = adminService.getChaincodeCount(
                    CHANNEL_NAME,
                    PEER_ENDPOINT,
                    OVERRIDE_AUTH,    // [关键] 传入域名覆盖，解决 localhost TLS 报错
                    MSP_ID,
                    MSP_CONFIG_PATH,
                    TLS_CERT_PATH,
                    FABRIC_CFG_PATH
                );
                System.out.println("结果: " + ccCount);
            } catch (Exception e) {
                System.out.println("\n[错误] 链码查询失败: " + e.getMessage());
                System.out.println("提示: 请检查 FABRIC_CFG_PATH 和 TLS 配置是否正确");
            }

            // 2. 获取 Orderer 数量 (SDK 原生 - 解析 ConfigBlock)
            System.out.print("2. 查询 Orderer 节点数量... ");
            int ordererCount = adminService.getOrdererCount(CHANNEL_NAME);
            System.out.println("结果: " + ordererCount);

            // 3. 获取 ConfigBlock (SDK 原生)
            System.out.print("3. 获取配置区块... ");
            Block configBlock = adminService.getConfigBlock(CHANNEL_NAME);
            System.out.println("成功，配置区块序号: " + configBlock.getHeader().getNumber());

            // 4. 获取 Peer 数量 (CLI 实现 - discover)
            System.out.print("4. 使用 CLI 发现 Peer 节点... ");
            try {
                // 动态获取私钥文件名 (keystore 文件名是随机的)
                Path userKeyPath = FileUtils.getFirstFile(KEY_PATH);
                Path userCertPath = FileUtils.getFirstFile(CERT_PATH);

                int peerCount = adminService.getPeersCount(
                        CHANNEL_NAME, 
                        PEER_ENDPOINT, 
                        MSP_ID, 
                        userCertPath.toAbsolutePath().toString(), 
                        userKeyPath.toAbsolutePath().toString(),
                        TLS_CERT_PATH // [新增] 传入 peerTLSCA 路径，对应 discover --peerTLSCA
                );
                System.out.println("在线 Peer 数量: " + peerCount);
                System.out.println("网络总节点数 (Peers + Orderers): " + (peerCount + ordererCount));
            } catch (Exception e) {
                System.out.println("\n[警告] Peer发现失败: " + e.getMessage());
                if (e.getMessage().contains("deadline exceeded") || e.getMessage().contains("connection refused")) {
                     System.out.println("提示: 如果遇到连接错误，请检查 /etc/hosts 是否映射了 127.0.0.1 peer0.guolong.com");
                }
            }

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