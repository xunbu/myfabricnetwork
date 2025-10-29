# 智能合约部署、更新与CLI调用

以下所有代码在fabric-samples/test-network下进行
`cd /home/xunbu/fabric/fabric-samples/test-network`

## 下载链码依赖

```bash
#切换到chaincode-go文件夹，使用pushd因为方便返回
pushd ../asset-transfer-basic/chaincode-go/
#打开go module模式
go env -w GO111MODULE=on
go mod tidy
go mod vendor
popd
```

## 打包并安装智能合约

### 设置环境变量

```bash
cd /home/xunbu/fabric/fabric-samples/test-network
export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/../config/
export CORE_PEER_TLS_ENABLED=true
```

### 打包智能合约

```bash
peer lifecycle chaincode package basic.tar.gz --path ../asset-transfer-basic/chaincode-go/ --lang golang --label basic_1.0

# peer lifecycle chaincode package basic.tar.gz --path /home/xunbu/fabric/fabric-samples/mycode/chaincode-go --lang golang --label basic_1.0

# peer lifecycle chaincode package myassetcontract.tar.gz --path /home/xunbu/fabric/fabric-samples/mycode/mychaincode --lang java --label myassetcontract_1.0
```

此时生成链码包basic.tar.gz位于test-network下

### 安装链码(在org1中)

切换org1身份

```bash
source ./setOrgEnv.sh org1
export CORE_PEER_LOCALMSPID CORE_PEER_TLS_ROOTCERT_FILE CORE_PEER_MSPCONFIGPATH CORE_PEER_ADDRESS

#等价于以下：

# export CORE_PEER_TLS_ENABLED=true
# export CORE_PEER_LOCALMSPID="Org1MSP"
# export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
# export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
# export CORE_PEER_ADDRESS=localhost:7051



```

安装链码

```bash
peer lifecycle chaincode install basic.tar.gz
# peer lifecycle chaincode install myassetcontract.tar.gz
```

### 在org2中安装链码

> 由于交易的背书策略指定为需要org1和org2的背书，所以需要在两个org组织的peer节点都安装链码

```bash
#切换身份
source ./setOrgEnv.sh org2
export CORE_PEER_LOCALMSPID CORE_PEER_TLS_ROOTCERT_FILE CORE_PEER_MSPCONFIGPATH CORE_PEER_ADDRESS
#安装链码
peer lifecycle chaincode install basic.tar.gz
# peer lifecycle chaincode install myassetcontract.tar.gz
```

## 查询链码安装信息获取链码包id

```bash
#查询链码安装信息
peer lifecycle chaincode queryinstalled
#导出链码包id，以下为示例
#export CC_PACKAGE_ID=basic_1.0:6a6f743366675481339c7ddda5f281a441c9ad8c13e32a9a5c07b892e44de105
export CC_PACKAGE_ID=$(peer lifecycle chaincode queryinstalled | grep -oP 'Package ID: \K[^,]+')
```

## 链码审批

### 以组织身份（以org2组织为例）进行审批

```bash
peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID mychannel --name basic --version 1.0 --package-id $CC_PACKAGE_ID --sequence 1 --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"

# peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID mychannel --name mycontract --version 1.0 --package-id $CC_PACKAGE_ID --sequence 1 --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
```

>链码包的名称在这里由--name指定
>注意CC_PACKAGE_ID必须是之前查询到的packageID，注意是否设置

以下命令可以查看有哪些组织批准了该链码

```bash
peer lifecycle chaincode checkcommitreadiness --channelID mychannel --name basic --version 1.0 --sequence 1 --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --output json

```

### 切换org1进行批准

```bash
#切换身份
source ./setOrgEnv.sh org1
export CORE_PEER_LOCALMSPID CORE_PEER_TLS_ROOTCERT_FILE CORE_PEER_MSPCONFIGPATH CORE_PEER_ADDRESS
#批准
peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID mychannel --name basic --version 1.0 --package-id $CC_PACKAGE_ID --sequence 1 --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
# peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID mychannel --name mycontract --version 2.0 --package-id $CC_PACKAGE_ID --sequence 2 --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
```

### 提交链码至通道

```bash
peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID mychannel --name basic --version 1.0 --sequence 1 --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt"

# peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID mychannel --name mycontract --version 1.0 --sequence 1 --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt"
```

### 检查链码定义信息

```bash
peer lifecycle chaincode querycommitted --channelID mychannel --name basic
# peer lifecycle chaincode querycommitted --channelID mychannel --name mycontract
```

## CLI调用链码

### CLI调用链码函数

org1调用mychannel的basic链码包的InitLedger函数创建一组初始资产

```bash
peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n basic --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" -c '{"function":"InitLedger","Args":[]}'
# peer chaincode invoke -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" -C mychannel -n mycontract --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt" -c '{"function":"createMyAsset","Args":["name","qinhan"]}'
```

> 因为链码的背书策略要求两个节点都要背书，因此需要两个证书

调用链码包GetAllAssets函数查询创建的资产集合

```bash
peer chaincode query -C mychannel -n basic -c '{"Args":["GetAllAssets"]}'
```

### CLI调用系统链码

设置必要的环境变量

```bash
#设置必要的环境变量，在test-network目录下
export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/../config/
export CORE_PEER_TLS_ENABLED=true

#设置peer身份的环境变量
source ./setOrgEnv.sh org1
export CORE_PEER_LOCALMSPID CORE_PEER_TLS_ROOTCERT_FILE CORE_PEER_MSPCONFIGPATH CORE_PEER_ADDRESS
```

> 以下命令返回的是protobuf二进制数据，因此显示的是乱码（使用gateway-api时也要对protobuf做相应的解析才行）

#### QSCC (Query System Chaincode)

```bash
# 获取区块信息
peer chaincode query -C mychannel -n qscc -c '{"Args":["GetBlockByNumber","mychannel","5"]}'

# 获取交易信息
peer chaincode query -C mychannel -n qscc -c '{"Args":["GetTransactionByID","mychannel","tx_id_here"]}'

# 获取区块链信息
peer chaincode query -C mychannel -n qscc -c '{"Args":["GetChainInfo","mychannel"]}'
```

#### LSCC (Lifecycle System Chaincode)

```bash
# 获取已安装的链码
peer chaincode query -C mychannel -n lscc -c '{"Args":["getinstalledchaincodes"]}'

# 获取通道上的链码
peer chaincode query -C mychannel -n lscc -c '{"Args":["getchaincodes"]}'

# 获取链码部署规范
peer chaincode query -C mychannel -n lscc -c '{"Args":["getdepspec","mychannel","my_chaincode"]}'
```

#### CSCC (Configuration System Chaincode)

```bash
# 获取通道配置
peer chaincode query -C mychannel -n cscc -c '{"Args":["GetConfigBlock","mychannel"]}'

# 获取当前peer加入的通道
peer chaincode query -C mychannel -n cscc -c '{"Args":["GetChannels"]}'
```

## 智能合约更新

略

## 脚本合集

### 智能合约部署(总)

```bash
cd /home/xunbu/fabric/fabric-samples/test-network
#下载链码依赖
pushd ../asset-transfer-basic/chaincode-go/
#打开go module模式
go env -w GO111MODULE=on
go mod tidy
go mod vendor
popd

export PATH=${PWD}/../bin:$PATH
export FABRIC_CFG_PATH=${PWD}/../config/
export CORE_PEER_TLS_ENABLED=true

peer lifecycle chaincode package basic.tar.gz --path ../asset-transfer-basic/chaincode-go/ --lang golang --label basic_1.0

source ./setOrgEnv.sh org1
export CORE_PEER_LOCALMSPID CORE_PEER_TLS_ROOTCERT_FILE CORE_PEER_MSPCONFIGPATH CORE_PEER_ADDRESS
peer lifecycle chaincode install basic.tar.gz

source ./setOrgEnv.sh org2
export CORE_PEER_LOCALMSPID CORE_PEER_TLS_ROOTCERT_FILE CORE_PEER_MSPCONFIGPATH CORE_PEER_ADDRESS
peer lifecycle chaincode install basic.tar.gz

export CC_PACKAGE_ID=$(peer lifecycle chaincode queryinstalled | grep -oP 'Package ID: \K[^,]+')

peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID mychannel --name basic --version 1.0 --package-id $CC_PACKAGE_ID --sequence 1 --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"

source ./setOrgEnv.sh org1
export CORE_PEER_LOCALMSPID CORE_PEER_TLS_ROOTCERT_FILE CORE_PEER_MSPCONFIGPATH CORE_PEER_ADDRESS
peer lifecycle chaincode approveformyorg -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID mychannel --name basic --version 1.0 --package-id $CC_PACKAGE_ID --sequence 1 --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"

peer lifecycle chaincode commit -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --channelID mychannel --name basic --version 1.0 --sequence 1 --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt"

peer lifecycle chaincode querycommitted --channelID mychannel --name basic
```
