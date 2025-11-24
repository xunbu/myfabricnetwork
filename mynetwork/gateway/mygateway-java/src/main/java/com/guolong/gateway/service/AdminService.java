package com.guolong.gateway.service;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.guolong.gateway.utils.ShellUtils;
import org.hyperledger.fabric.client.Gateway;
import org.hyperledger.fabric.protos.common.*;

import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class AdminService {
    private final Gateway gateway;
    private final ObjectMapper objectMapper;
    // [新增] 保存 bin 目录路径
    private final String binPath;

    // [修改] 构造函数接收 binPath
    public AdminService(Gateway gateway, String binPath) {
        this.gateway = gateway;
        this.objectMapper = new ObjectMapper();
        this.binPath = binPath;
    }

    // [私有辅助] 拼接绝对路径
    private String getBinaryPath(String command) {
        if (binPath == null || binPath.isEmpty()) {
            return command; // 回退到系统 PATH
        }
        return Paths.get(binPath, command).toString();
    }

    // ==========================================
    // 1. Channel / ConfigBlock 相关 (SDK 原生)
    // ==========================================

    public Block getConfigBlock(String channelName) throws Exception {
        byte[] blockBytes = gateway.getNetwork(channelName)
                .getContract("cscc")
                .evaluateTransaction("GetConfigBlock", channelName);
        return Block.parseFrom(blockBytes);
    }

    public int getOrdererCount(String channelName) throws Exception {
        Block configBlock = getConfigBlock(channelName);
        if (configBlock.getData().getDataCount() == 0) return 0;

        Envelope envelope = Envelope.parseFrom(configBlock.getData().getData(0));
        Payload payload = Payload.parseFrom(envelope.getPayload());
        ConfigEnvelope configEnvelope = ConfigEnvelope.parseFrom(payload.getData());
        ConfigGroup channelGroup = configEnvelope.getConfig().getChannelGroup();

        ConfigValue ordererAddressesValue = null;
        
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
    // 2. Chaincode 相关 (CLI 实现)
    // ==========================================

    // [修改] 移除了 fabricBinPath 参数
    public int getChaincodeCount(String channelName, 
                                 String peerEndpoint,
                                 String overrideAuth,
                                 String mspId, 
                                 String mspConfigPath,
                                 String tlsRootCertPath,
                                 String fabricCfgPath) throws Exception {
        
        // 自动拼接绝对路径
        String peerCmd = getBinaryPath("peer");

        List<String> cmd = Arrays.asList(
            peerCmd, "lifecycle", "chaincode", "querycommitted",
            "--channelID", channelName,
            "--output", "json",
            "--peerAddresses", peerEndpoint,
            "--tlsRootCertFiles", tlsRootCertPath,
            "--tls"
        );

        Map<String, String> env = new HashMap<>();
        env.put("FABRIC_CFG_PATH", fabricCfgPath); 
        env.put("CORE_PEER_LOCALMSPID", mspId);
        env.put("CORE_PEER_MSPCONFIGPATH", mspConfigPath);
        env.put("CORE_PEER_TLS_ENABLED", "true");
        
        if (overrideAuth != null && !overrideAuth.isEmpty()) {
            env.put("CORE_PEER_TLS_SERVERHOSTOVERRIDE", overrideAuth);
        }
        
        String output = ShellUtils.exec(cmd, env);
        
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

    // [修改] 移除了 fabricBinPath 参数
    public int getPeersCount(String channelName, 
                             String peerEndpoint, 
                             String mspId, 
                             String userCertPath, 
                             String userKeyPath,
                             String peerTlsCaPath) throws Exception {
        
        // 自动拼接绝对路径
        String discoverCmd = getBinaryPath("discover");

        List<String> cmd = new ArrayList<>();
        cmd.add(discoverCmd);
        cmd.add("--peerTLSCA");
        cmd.add(peerTlsCaPath);
        cmd.add("--userKey");
        cmd.add(userKeyPath);
        cmd.add("--userCert");
        cmd.add(userCertPath);
        cmd.add("--MSP");
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
            // 兼容两种 JSON 格式
            if (root.size() > 0 && root.get(0).has("Endpoint")) {
                totalPeers = root.size();
            } else {
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