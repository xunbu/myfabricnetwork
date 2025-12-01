package gateway

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"guolong.com/basic-chaincode/chaincode"
)

type RichPageResult struct {
	Results  []chaincode.QueryRichResult `json:"results"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"pageSize"`
	Total    int                         `json:"total"`
}

// EvaluateTransaction 执行查询交易 (只读)
func EvaluateTransaction(gw *client.Gateway, channelName, chainCodeName, funcName string, args ...string) ([]byte, error) {
	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chainCodeName)
	return contract.EvaluateTransaction(funcName, args...)
}

// SubmitTransaction 执行提交交易 (写账本)
func SubmitTransaction(gw *client.Gateway, channelName, chainCodeName, funcName string, args ...string) ([]byte, error) {
	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chainCodeName)
	return contract.SubmitTransaction(funcName, args...)
}

func PutValue(gw *client.Gateway, channelName string, chaincodeName string, key string, data string) ([]byte, error) {
	return SubmitTransaction(gw, channelName, chaincodeName, "PutValue", key, data)
}

func PutMap(gw *client.Gateway, channelName string, chaincodeName string, key string, jsonMap map[string]interface{}) ([]byte, error) {
	jsonBytes, err := json.Marshal(jsonMap)
	if err != nil {
		return nil, fmt.Errorf("JSON 序列化失败: %w", err)
	}
	return SubmitTransaction(gw, channelName, chaincodeName, "PutValue", key, string(jsonBytes))
}

func PutKVs(gw *client.Gateway, channelName string, chaincodeName string, KVMap map[string]string) ([]byte, error) {
	v, err := json.Marshal(KVMap)
	if err != nil {
		return nil, err
	}
	return SubmitTransaction(gw, channelName, chaincodeName, "PutKVs", string(v))
}

func GetValue(gw *client.Gateway, channelName string, chaincodeName string, key string) (string, error) {
	v, err := EvaluateTransaction(gw, channelName, chaincodeName, "QueryByKey", key)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

func DeleteKey(gw *client.Gateway, channelName string, chaincodeName string, key string) ([]byte, error) {
	return SubmitTransaction(gw, channelName, chaincodeName, "DeleteByKey", key)
}

func GetAllData(gw *client.Gateway, channelName string, chaincodeName string) ([]chaincode.QueryRichResult, error) {
	query := map[string]interface{}{"selector": map[string]interface{}{}}
	queryBytes, _ := json.Marshal(query)
	v, err := EvaluateTransaction(gw, channelName, chaincodeName, "QueryByRich", string(queryBytes))
	if err != nil {
		return nil, err
	}
	if len(v) == 0 {
		return []chaincode.QueryRichResult{}, nil
	}
	var richResults []chaincode.QueryRichResult
	json.Unmarshal(v, &richResults)
	return richResults, nil
}

func GetAllDataByPageWithLimit(gw *client.Gateway, channelName string, chaincodeName string, page int, pageSize int) (*RichPageResult, error) {
	skip := page * pageSize
	query := map[string]interface{}{"selector": map[string]interface{}{}, "skip": skip}
	queryBytes, _ := json.Marshal(query)
	v, err := EvaluateTransaction(gw, channelName, chaincodeName, "QueryByRichWithLimit", string(queryBytes), strconv.Itoa(pageSize))
	if err != nil {
		return nil, err
	}
	var results []chaincode.QueryRichResult
	if len(v) > 0 {
		json.Unmarshal(v, &results)
	}
	return &RichPageResult{Results: results, Page: page, PageSize: pageSize, Total: skip + len(results)}, nil
}

func GetKeyHistory(gw *client.Gateway, channelName string, chaincodeName, key string) (*[]chaincode.KeyHistory, error) {
	v, err := EvaluateTransaction(gw, channelName, chaincodeName, "GetKeyHistory", key)
	if err != nil {
		return nil, err
	}
	kh := &[]chaincode.KeyHistory{}
	json.Unmarshal(v, &kh)
	return kh, nil
}
