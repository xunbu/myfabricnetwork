package chaincode

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

// 统计交易总量(pass)

func (s *SmartContract) KeyExists(ctx contractapi.TransactionContextInterface, key string) (bool, error) {
	value, err := ctx.GetStub().GetState(key)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return value != nil, nil
}

// 根据key查询数据
func (s *SmartContract) QueryByKey(ctx contractapi.TransactionContextInterface, key string) ([]byte, error) {
	v, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if v == nil {
		return nil, fmt.Errorf("key %v not found", key)
	}
	return v, nil
}

func (s *SmartContract) QueryByRange(ctx contractapi.TransactionContextInterface, start string, end string) (map[string]interface{}, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange(start, end)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	results := make(map[string]interface{}, 0) //注意这里string是key，interface{}为value
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		results[queryResponse.Key] = queryResponse.Value
	}
	return results, nil
}

func (s *SmartContract) QueryByRich(ctx contractapi.TransactionContextInterface, richQuery string) ([]byte, error) {
	// query := fmt.Sprintf(`{"selector":{"model_label":"%s"}}`, modelValue)//根据模块（存储、采集、管控）查询相应的数据
	// query := fmt.Sprintf(`{"selector":{"database_label":"%s"}}`, labelValue)// 根据数据库查询相应的表元数据
	resultsIterator, err := ctx.GetStub().GetQueryResult(richQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to iterate query result: %v", err)
	}
	defer resultsIterator.Close()
	results := make([]map[string]interface{}, 0)
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate query result: %v", err)
		}
		var record map[string]interface{}
		if err := json.Unmarshal(queryResponse.Value, &record); err != nil {
			return nil, fmt.Errorf("failed to unmarshal record %v", err)
		}
		results = append(results, record)
	}
	v, err := json.Marshal(results)
	if err != nil {
		return nil, fmt.Errorf("can't marshal data ,%v", err)
	}
	return v, nil
}

// 查询数据并以string形式返回
func (s *SmartContract) QueryByKeyString(ctx contractapi.TransactionContextInterface, key string) (string, error) {
	v, err := s.QueryByKey(ctx, key)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

// 富查询并以string形式返回
func (s *SmartContract) QueryByRichString(ctx contractapi.TransactionContextInterface, key string) (string, error) {
	v, err := s.QueryByRich(ctx, key)
	if err != nil {
		return "", err
	}
	return string(v), nil
}

// string格式数据上链(用于企业数字签名上链)
func (s *SmartContract) PutString(ctx contractapi.TransactionContextInterface, key string, value string) error {
	err := ctx.GetStub().PutState(key, []byte(value))
	if err != nil {
		return fmt.Errorf("error in PutState, key:%v,value:%v", key, value)
	}
	return nil
}

func (s *SmartContract) PutBytes(ctx contractapi.TransactionContextInterface, key string, value []byte) error {
	err := ctx.GetStub().PutState(key, value)
	if err != nil {
		return fmt.Errorf("error in PutState, key:%v,value:%v", key, value)
	}
	return nil
}

// 更新数据 (string)
func (s *SmartContract) UpdateString(ctx contractapi.TransactionContextInterface, key string, value string) error {
	exists, err := s.KeyExists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the key %s does not exist", key)
	}
	return s.PutString(ctx, key, value)
}

// 更新数据 (string)
func (s *SmartContract) UpdateBytes(ctx contractapi.TransactionContextInterface, key string, value []byte) error {
	exists, err := s.KeyExists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the key %s does not exist", key)
	}
	return s.PutBytes(ctx, key, value)
}

func (s *SmartContract) DeleteByKey(ctx contractapi.TransactionContextInterface, key string) error {
	exists, err := s.KeyExists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	err = ctx.GetStub().DelState(key)
	if err != nil {
		return err
	}
	return nil
}
