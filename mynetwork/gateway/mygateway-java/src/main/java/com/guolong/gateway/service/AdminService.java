package com.guolong.gateway.service;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.guolong.gateway.utils.ShellUtils;
import org.hyperledger.fabric.client.Gateway;
import org.hyperledger.fabric.protos.common.*;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class AdminService {
    private final Gateway gateway;
    private final ObjectMapper objectMapper;

    public AdminService(Gateway gateway) {
        this.gateway = gateway;
        this.objectMapper = new ObjectMapper();
    }

    // ==========================================
    // 1. Channel / ConfigBlock 相关 (SDK 原生)
    // ==========================================

    public Block getConfigBlock(String channelName) throws Exception {
        // 调用 CSCC 系统链码获取配置区块
        byte[] blockBytes = gateway.getNetwork(channelName)
                .getContract("cscc")
                .evaluateTransaction("GetConfigBlock", channelName);
        return Block.parseFrom(blockBytes);
    }

    public int getOrdererCount(String channelName) throws Exception {
        Block configBlock = getConfigBlock(channelName);
        if (configBlock.getData().getDataCount() == 0) return 0;

        // 层层解析 Protobuf 结构
        Envelope envelope = Envelope.parseFrom(configBlock.getData().getData(0));
        Payload payload = Payload.parseFrom(envelope.getPayload());
        ConfigEnvelope configEnvelope = ConfigEnvelope.parseFrom(payload.getData());
        ConfigGroup channelGroup = configEnvelope.getConfig().getChannelGroup();

        ConfigValue ordererAddressesValue = null;
        
        // 尝试获取 OrdererAddresses 配置
        if (channelGroup.getValuesMap().containsKey("OrdererAddresses")) {
            ordererAddressesValue = channelGroup.getValuesMap().get("OrdererAddresses");
        } else if (channelGroup.getGroupsMap().containsKey("Orderer")) {
             ConfigGroup ordererGroup = channelGroup.getGroupsMap().get("Orderer");
             if (ordererGroup.getValuesMap().containsKey("OrdererAddresses")) {
                 ordererAddressesValue = ordererGroup.getValuesMap().get("OrdererAddresses");
             }
        }

        if (ordererAddressesValue != null) {
            OrdererAddresses addresses = OrdererAddresses.parseFrom(ordererAddressesValue.getValue());
            return addresses.getAddressesCount();
        }
        return 0;
    }

    // ==========================================
    // 2. Chaincode 相关 (CLI 实现 - 绕过 ACL)
    // ==========================================

    public int getChaincodeCount(String channelName, 
                                 String peerEndpoint,
                                 String overrideAuth, // TLS Host Override
                                 String mspId, 
                                 String mspConfigPath,
                                 String tlsRootCertPath,
                                 String fabricCfgPath) throws Exception {
        
        // 使用 peer lifecycle 命令查询已提交链码
        List<String> cmd = Arrays.asList(
            "peer", "lifecycle", "chaincode", "querycommitted",
            "--channelID", channelName,
            "--output", "json",
            "--peerAddresses", peerEndpoint,       // 指定 Peer 地址
            "--tlsRootCertFiles", tlsRootCertPath, // 指定 Peer TLS CA
            "--tls"
        );

        // 设置环境变量
        Map<String, String> env = new HashMap<>();
        env.put("FABRIC_CFG_PATH", fabricCfgPath); 
        env.put("CORE_PEER_LOCALMSPID", mspId);
        env.put("CORE_PEER_MSPCONFIGPATH", mspConfigPath);
        env.put("CORE_PEER_TLS_ENABLED", "true");
        
        // 关键：如果在本地连接 localhost，必须设置此变量以匹配证书域名
        if (overrideAuth != null && !overrideAuth.isEmpty()) {
            env.put("CORE_PEER_TLS_SERVERHOSTOVERRIDE", overrideAuth);
        }
        
        String output = ShellUtils.exec(cmd, env);
        
        // 解析: { "chaincode_definitions": [ ... ] }
        Map<String, Object> result = objectMapper.readValue(output, new TypeReference<Map<String, Object>>(){});
        
        if (result.containsKey("chaincode_definitions")) {
            List<?> list = (List<?>) result.get("chaincode_definitions");
            return list.size();
        }
        return 0;
    }

    // ==========================================
    // 3. Discovery / Peer 相关 (CLI 实现)
    // ==========================================

    public int getPeersCount(String channelName, 
                             String peerEndpoint, 
                             String mspId, 
                             String userCertPath, 
                             String userKeyPath,
                             String peerTlsCaPath) throws Exception {
        
        List<String> cmd = new ArrayList<>();
        cmd.add("discover");
        cmd.add("--peerTLSCA"); // 关键参数：Peer 的 TLS CA
        cmd.add(peerTlsCaPath);
        cmd.add("--userKey");
        cmd.add(userKeyPath);
        cmd.add("--userCert");
        cmd.add(userCertPath);
        cmd.add("--MSP");       // 关键参数：大写 MSP
        cmd.add(mspId);
        cmd.add("peers");
        cmd.add("--channel");
        cmd.add(channelName);
        cmd.add("--server");
        cmd.add(peerEndpoint);
        
        String output = ShellUtils.exec(cmd, new HashMap<>());
        
        JsonNode root = objectMapper.readTree(output);
        int totalPeers = 0;

        if (root.isArray()) {
            // [修复逻辑] 兼容两种 JSON 格式
            
            // 格式 A: 扁平列表 (你当前环境的输出)
            // 示例: [{"MSPID": "...", "Endpoint": "..."}, ...]
            if (root.size() > 0 && root.get(0).has("Endpoint")) {
                totalPeers = root.size();
                System.out.println("    -> [调试] 解析模式: 直接列表 (Flat List)");
                for (JsonNode peer : root) {
                    System.out.println("    -> 发现节点: " + peer.path("Endpoint").asText() + " (" + peer.path("MSPID").asText() + ")");
                }
            } 
            // 格式 B: 按组织分组 (旧版本或特定配置)
            // 示例: [{"msp_id": "...", "peers": [...]}]
            else {
                System.out.println("    -> [调试] 解析模式: 组织分组 (Nested Org)");
                for (JsonNode item : root) {
                    if (item.has("peers")) {
                        totalPeers += item.get("peers").size();
                    }
                }
            }
        }
        
        return totalPeers;
    }
}