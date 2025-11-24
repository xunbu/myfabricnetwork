package com.guolong.gateway.service;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.guolong.gateway.dto.KeyHistory;
import com.guolong.gateway.dto.QueryRichResult;
import com.guolong.gateway.dto.RichPageResult;
import org.hyperledger.fabric.client.Contract;
import org.hyperledger.fabric.client.Gateway;
import org.hyperledger.fabric.client.Network;

import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class ChaincodeService {
    private final Gateway gateway;
    private final ObjectMapper objectMapper = new ObjectMapper();

    public ChaincodeService(Gateway gateway) {
        this.gateway = gateway;
    }

    private Contract getContract(String channelName, String chaincodeName) {
        Network network = gateway.getNetwork(channelName);
        return network.getContract(chaincodeName);
    }

    // Evaluate 包装
    public byte[] evaluateTransaction(String channelName, String chaincodeName, String func, String... args) throws Exception {
        return getContract(channelName, chaincodeName).evaluateTransaction(func, args);
    }

    // Submit 包装
    public byte[] submitTransaction(String channelName, String chaincodeName, String func, String... args) throws Exception {
        return getContract(channelName, chaincodeName).submitTransaction(func, args);
    }

    // 写入数据
    public byte[] putValue(String channelName, String chaincodeName, String key, String data) throws Exception {
        return submitTransaction(channelName, chaincodeName, "PutValue", key, data);
    }

    // 写入 Map
    public byte[] putMap(String channelName, String chaincodeName, String key, Map<String, Object> jsonMap) throws Exception {
        String jsonString = objectMapper.writeValueAsString(jsonMap);
        return submitTransaction(channelName, chaincodeName, "PutValue", key, jsonString);
    }

    // 写入 KVs
    public byte[] putKvs(String channelName, String chaincodeName, Map<String, String> kvMap) throws Exception {
        String jsonString = objectMapper.writeValueAsString(kvMap);
        return submitTransaction(channelName, chaincodeName, "PutKVs", jsonString);
    }

    // 获取数据
    public String getValue(String channelName, String chaincodeName, String key) throws Exception {
        byte[] bytes = evaluateTransaction(channelName, chaincodeName, "QueryByKey", key);
        return new String(bytes, StandardCharsets.UTF_8);
    }

    // 删除数据
    public byte[] deleteKey(String channelName, String chaincodeName, String key) throws Exception {
        return submitTransaction(channelName, chaincodeName, "DeleteByKey", key);
    }

    // 获取所有数据 (Rich Query)
    public List<QueryRichResult> getAllData(String channelName, String chaincodeName) throws Exception {
        Map<String, Object> query = new HashMap<>();
        query.put("selector", new HashMap<>()); // 空选择器匹配所有
        String queryString = objectMapper.writeValueAsString(query);

        byte[] bytes = evaluateTransaction(channelName, chaincodeName, "QueryByRich", queryString);
        if (bytes == null || bytes.length == 0) {
            return Collections.emptyList();
        }
        return objectMapper.readValue(bytes, new TypeReference<List<QueryRichResult>>() {});
    }

    // 分页查询 (对应 Go 的 GetAllDataByPageWithLimit)
    public RichPageResult getAllDataByPageWithLimit(String channelName, String chaincodeName, int page, int pageSize) throws Exception {
        int skip = (page - 1) * pageSize;
        
        // 构建 CouchDB 查询语句
        Map<String, Object> query = new HashMap<>();
        query.put("selector", new HashMap<>());
        query.put("limit", pageSize);
        query.put("skip", skip);
        
        // [新增] 实现 Sort 逻辑，与 Go 代码一致: "sort": [{"_id": "asc"}]
        List<Map<String, String>> sortList = new ArrayList<>();
        Map<String, String> sortId = new HashMap<>();
        sortId.put("_id", "asc");
        sortList.add(sortId);
        query.put("sort", sortList);
        
        String queryString = objectMapper.writeValueAsString(query);
        byte[] bytes = evaluateTransaction(channelName, chaincodeName, "QueryByRich", queryString);
        
        List<QueryRichResult> results = Collections.emptyList();
        if (bytes != null && bytes.length > 0) {
            results = objectMapper.readValue(bytes, new TypeReference<List<QueryRichResult>>() {});
        }

        boolean hasMore = results.size() == pageSize;
        // 近似计算 total，因为 CouchDB 在复杂查询下获取精确 total 性能开销大
        int estimatedTotal = skip + results.size(); 
        
        return new RichPageResult(results, hasMore, page, estimatedTotal);
    }
    
    // 获取历史
    public List<KeyHistory> getKeyHistory(String channelName, String chaincodeName, String key) throws Exception {
        byte[] bytes = evaluateTransaction(channelName, chaincodeName, "GetKeyHistory", key);
        return objectMapper.readValue(bytes, new TypeReference<List<KeyHistory>>() {});
    }
}