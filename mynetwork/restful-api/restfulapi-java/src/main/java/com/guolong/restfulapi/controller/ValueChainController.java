package com.guolong.restfulapi.controller;

import com.guolong.gateway.service.AdminService;
import com.guolong.gateway.service.ChaincodeService;
import com.guolong.gateway.service.LedgerService;
import com.guolong.gateway.utils.FileUtils;
import com.guolong.restfulapi.dto.DeleteKeyRequest;
import com.guolong.restfulapi.dto.PutValueRequest;
import com.guolong.restfulapi.service.DockerMonitorService;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.nio.file.Path;
import java.util.HashMap;
import java.util.Map;

@RestController
@RequestMapping("/valuechain")
@RequiredArgsConstructor
@CrossOrigin("*")
public class ValueChainController {

    private final ChaincodeService chaincodeService;
    private final LedgerService ledgerService;
    private final AdminService adminService;
    private final DockerMonitorService dockerService;

    @Value("${fabric.channel-name}") private String channelName;
    @Value("${fabric.chaincode-name}") private String chaincodeName;
    @Value("${fabric.peer-endpoint}") private String peerEndpoint;
    @Value("${fabric.override-auth}") private String overrideAuth;
    @Value("${fabric.msp-id}") private String mspId;
    @Value("${fabric.msp-config-path}") private String mspConfigPath;
    @Value("${fabric.tls-cert-path}") private String tlsRootCertPath;
    @Value("${fabric.fabric-cfg-path}") private String fabricCfgPath;
    // [删除] fabricBinPath 不需要了
    @Value("${fabric.cert-path}") private String certDir;
    @Value("${fabric.key-path}") private String keyDir;

    @GetMapping("/info")
    public ResponseEntity<?> getInfo() {
        Map<String, Object> res = new HashMap<>();

        // 1. SDK 原生查询
        try {
            res.put("blockHeight", ledgerService.getBlockHeight(channelName));
            res.put("totalTransactionCount", ledgerService.getTransactionCount(channelName));
            res.put("orgCount", ledgerService.getOrganizationCount(channelName));
        } catch (Exception e) {
            return ResponseEntity.internalServerError().body(Map.of("error", "Ledger query failed: " + e.getMessage()));
        }

        // 2. Admin CLI 查询
        
        // 链码数量
        try {
            // [修改] 移除了 fabricBinPath 参数
            int ccCount = adminService.getChaincodeCount(
                channelName, peerEndpoint, overrideAuth, mspId, mspConfigPath, tlsRootCertPath, fabricCfgPath
            );
            res.put("chainCodeCount", ccCount);
        } catch (Exception e) {
            System.err.println("[警告] 获取链码数量失败: " + e.getMessage());
            res.put("chainCodeCount", -1);
        }

        // 节点数量
        try {
            Path keyFile = FileUtils.getFirstFile(keyDir);
            Path certFile = FileUtils.getFirstFile(certDir);
            
            int peers = 0;
            try {
                // [修改] 移除了 fabricBinPath 参数
                peers = adminService.getPeersCount(
                    channelName, peerEndpoint, mspId, 
                    certFile.toString(), keyFile.toString(), tlsRootCertPath
                );
            } catch (Exception ex) {
                System.err.println("[警告] 获取Peer数量失败: " + ex.getMessage());
            }

            int orderers = 0;
            try {
                orderers = adminService.getOrdererCount(channelName);
            } catch (Exception ex) {
                System.err.println("[警告] 获取Orderer数量失败: " + ex.getMessage());
            }
            
            res.put("nodeCount", peers + orderers);
        } catch (Exception e) {
            System.err.println("[警告] 获取节点总数逻辑错误: " + e.getMessage());
            res.put("nodeCount", -1);
        }

        return ResponseEntity.ok(res);
    }

    // ... 其他方法 (getBlocks, getBlockByNum, etc.) 保持不变 ...
    // 为了节省篇幅，这里只列出受影响的 getInfo 方法，其他方法和之前给出的代码完全一致
    
    @GetMapping("/blocks")
    public ResponseEntity<?> getBlocks(@RequestParam(defaultValue = "0") long pageNum,
                                       @RequestParam(defaultValue = "10") long pageSize) {
        try {
            return ResponseEntity.ok(ledgerService.getBlockListByPage(channelName, pageNum, pageSize));
        } catch (Exception e) {
            return ResponseEntity.internalServerError().body(Map.of("error", e.getMessage()));
        }
    }

    @GetMapping("/block")
    public ResponseEntity<?> getBlockByNum(@RequestParam long blockNum) {
        try {
            return ResponseEntity.ok(ledgerService.getBlockByNum(channelName, blockNum));
        } catch (Exception e) {
            return ResponseEntity.internalServerError().body(Map.of("error", e.getMessage()));
        }
    }

    @GetMapping("/cpuHistory")
    public ResponseEntity<?> getCpuHistory() {
        return ResponseEntity.ok(dockerService.getAllCpuHistory());
    }

    @GetMapping("/memoryHistory")
    public ResponseEntity<?> getMemoryHistory() {
        return ResponseEntity.ok(dockerService.getAllMemoryHistory());
    }

    @GetMapping("/data/all")
    public ResponseEntity<?> getAllData() {
        try {
            return ResponseEntity.ok(chaincodeService.getAllData(channelName, chaincodeName));
        } catch (Exception e) {
            return ResponseEntity.internalServerError().body(Map.of("error", e.getMessage()));
        }
    }

    @GetMapping("/data/history")
    public ResponseEntity<?> getKeyHistory(@RequestParam String key) {
        try {
            return ResponseEntity.ok(chaincodeService.getKeyHistory(channelName, chaincodeName, key));
        } catch (Exception e) {
            return ResponseEntity.internalServerError().body(Map.of("error", e.getMessage()));
        }
    }

    @GetMapping("/transaction")
    public ResponseEntity<?> getTxById(@RequestParam String txID) {
        try {
            return ResponseEntity.ok(ledgerService.getTxById(channelName, txID));
        } catch (Exception e) {
            return ResponseEntity.internalServerError().body(Map.of("error", e.getMessage()));
        }
    }

    @PostMapping("/data")
    public ResponseEntity<?> putData(@RequestBody PutValueRequest req) {
        try {
            chaincodeService.putValue(channelName, chaincodeName, req.getKey(), req.getValue());
            return ResponseEntity.ok(Map.of("status", "success"));
        } catch (Exception e) {
            return ResponseEntity.internalServerError().body(Map.of("error", e.getMessage()));
        }
    }

    @PostMapping("/data/delete")
    public ResponseEntity<?> deleteData(@RequestBody DeleteKeyRequest req) {
        try {
            chaincodeService.deleteKey(channelName, chaincodeName, req.getKey());
            return ResponseEntity.ok(Map.of("status", "success"));
        } catch (Exception e) {
            return ResponseEntity.internalServerError().body(Map.of("error", e.getMessage()));
        }
    }
}