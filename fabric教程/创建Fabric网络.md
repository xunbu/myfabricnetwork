# 创建Fabric网络

以下所有操作在fabric-samples/test-network下进行
`cd /home/xunbu/fabric/fabric-samples/test-network`

## 预备操作

### 将bin目录、FABRIC_CFG_PATH添加到系统路径

```bash
export PATH=$PATH:$PWD/../bin
export FABRIC_CFG_PATH=${PWD}/configtx
```

> 这里FABRIC_CFG_PATH包含configtx.yaml用于通道配置

## 使用cryptogen创建order和peer组织

### 创建Order组织及身份证书

```bash
cryptogen generate --config=./organizations/cryptogen/crypto-config-orderer.yaml --output=organizations
```

### 创建org1、org2的组织及身份证书

```bash
cryptogen generate --config=./organizations/cryptogen/crypto-config-org1.yaml --output=organizations
cryptogen generate --config=./organizations/cryptogen/crypto-config-org2.yaml --output=organizations
```

## 网络配置与启动网络
>
> test-network的网络配置文件是`compose/compose-test-net.yaml`

启动网络（levelDB）

```bash
DOCKER_SOCK=/var/run/docker.sock docker-compose -f compose/compose-test-net.yaml -f compose/docker/docker-compose-test-net.yaml up -d
```

### 启动网络（couchDB）

```bash
export DOCKER_SOCK=/var/run/docker.sock
docker-compose -f compose/compose-test-net.yaml -f compose/docker/docker-compose-test-net.yaml -f compose/compose-couch.yaml -f compose/docker/docker-compose-couch.yaml up -d
```

### 关闭网络

```bash
export DOCKER_SOCK=/var/run/docker.sock
docker-compose -f compose/compose-test-net.yaml -f compose/docker/docker-compose-test-net.yaml -f compose/compose-couch.yaml -f compose/docker/docker-compose-couch.yaml down -v
```

## 应用通道配置与创世块生成

> configtxgen使用configtx.yaml中定义的通道配置创建创世块

## 创建应用通道

### 设置orderer管理员环境变量

```bash
export ORDERER_CA=${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem
export ORDERER_ADMIN_TLS_SIGN_CERT=${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/server.crt
export ORDERER_ADMIN_TLS_PRIVATE_KEY=${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/server.key
```

### 生成应用通道的创世块

```bash
configtxgen -profile ChannelUsingRaft -outputBlock ./channel-artifacts/mychannel.block -channelID mychannel
```

### 在orderer上创建mychannel

```bash
osnadmin channel join --channelID mychannel --config-block ./channel-artifacts/mychannel.block -o localhost:7053 --ca-file "$ORDERER_CA" --client-cert "$ORDERER_ADMIN_TLS_SIGN_CERT" --client-key "$ORDERER_ADMIN_TLS_PRIVATE_KEY"
```

### 查看orderer上的通道

```bash
osnadmin channel list -o localhost:7053 --ca-file "$ORDERER_CA" --client-cert "$ORDERER_ADMIN_TLS_SIGN_CERT" --client-key "$ORDERER_ADMIN_TLS_PRIVATE_KEY"
```

## 将peer加入到mychannel

### 将core.yaml添加到环境变量（设置FABRIC_CFG_PATH）

> core.yaml包含org1、org2的对等节点配置

```bash
export FABRIC_CFG_PATH=$PWD/../config/
```

### 设置org1管理员环境变量

```bash
export CORE_PEER_TLS_ENABLED=true

export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
```

> 以上命令可以使用test-network下的`setOrgEnv.sh`脚本完成：`source ./setOrgEnv.sh org1`

### 将peer0.org1节点加入mychannel

```bash
peer channel join -b ./channel-artifacts/mychannel.block
```

### 加入org2的对等节点

```bash
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051

peer channel join -b ./channel-artifacts/mychannel.block
```

## 设置锚节点

### 设置org1管理员环境变量

```bash
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
```

### 使用 peer channel fetch 命令拉取最新的通道配置区块

```bash
peer channel fetch config channel-artifacts/config_block.pb -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com -c mychannel --tls --cafile "$ORDERER_CA"
```

>通道配置块 config_block.pb 存储在 channel-artifacts 文件夹中

### 切换到channel-artifacts

`cd channel-artifacts`

### 使用 configtxlator 工具来处理通道配置

将区块从 protobuf 格式解码为可读取和编辑的 JSON 对象。同时我们会剥离不必要的区块数据，仅保留通道配置部分。

```bash
configtxlator proto_decode --input config_block.pb --type common.Block --output config_block.json
jq '.data.data[0].payload.data.config' config_block.json > config.json
```

复制一份config.json文件

```bash
cp config.json config_copy.json
```

将 org1的锚节点配置写入通道配置

```bash
jq '.channel_group.groups.Application.groups.Org1MSP.values += {"AnchorPeers":{"mod_policy": "Admins","value":{"anchor_peers": [{"host": "peer0.org1.example.com","port": 7051}]},"version": "0"}}' config_copy.json > modified_config.json
```

将原始和修改后的通道配置重新转换为 protobuf 格式，并计算它们之间的差异。

```bash
configtxlator proto_encode --input config.json --type common.Config --output config.pb

configtxlator proto_encode --input modified_config.json --type common.Config --output modified_config.pb

configtxlator compute_update --channel_id mychannel --original config.pb --updated modified_config.pb --output config_update.pb
```

将配置更新包装在一个交易信封中

```bash
configtxlator proto_decode --input config_update.pb --type common.ConfigUpdate --output config_update.json
echo '{"payload":{"header":{"channel_header":{"channel_id":"mychannel", "type":2}},"data":{"config_update":'$(cat config_update.json)'}}}' | jq . > config_update_in_envelope.json
configtxlator proto_encode --input config_update_in_envelope.json --type common.Envelope --output config_update_in_envelope.pb
```

### 通道配置更新

```bash
cd ..

peer channel update -f channel-artifacts/config_update_in_envelope.pb -c mychannel -o localhost:7050  --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
```

### 同样的方法设置org2的锚节点

```bash
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051

peer channel fetch config channel-artifacts/config_block.pb -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com -c mychannel --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"

cd channel-artifacts
configtxlator proto_decode --input config_block.pb --type common.Block --output config_block.json
jq '.data.data[0].payload.data.config' config_block.json > config.json
cp config.json config_copy.json
jq '.channel_group.groups.Application.groups.Org2MSP.values += {"AnchorPeers":{"mod_policy": "Admins","value":{"anchor_peers": [{"host": "peer0.org2.example.com","port": 9051}]},"version": "0"}}' config_copy.json > modified_config.json
configtxlator proto_encode --input config.json --type common.Config --output config.pb
configtxlator proto_encode --input modified_config.json --type common.Config --output modified_config.pb
configtxlator compute_update --channel_id mychannel --original config.pb --updated modified_config.pb --output config_update.pb

configtxlator proto_decode --input config_update.pb --type common.ConfigUpdate --output config_update.json
echo '{"payload":{"header":{"channel_header":{"channel_id":"mychannel", "type":2}},"data":{"config_update":'$(cat config_update.json)'}}}' | jq . > config_update_in_envelope.json
configtxlator proto_encode --input config_update_in_envelope.json --type common.Envelope --output config_update_in_envelope.pb

cd ..
peer channel update -f channel-artifacts/config_update_in_envelope.pb -c mychannel -o localhost:7050  --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
```

### 使用`peer channel info`查看通道是否更新（区块高度变化）

```bash
peer channel getinfo -c mychannel
```

## 关闭网络

### 关闭docker

```bash
DOCKER_SOCK=/var/run/docker.sock docker-compose -f compose/compose-test-net.yaml -f compose/docker/docker-compose-test-net.yaml down --volumes --remove-orphans
```

## 删除相关文件

```bash
rm -rf channel-artifacts/
rm -rf ./organizations/ordererOrganizations/
rm -rf ./organizations/peerOrganizations/
```

## 脚本合集

### 创建网络（总）

```bash
#将二进制程序、通道配置写在路径中
cd /home/xunbu/fabric/fabric-samples/test-network
export PATH=$PATH:$PWD/../bin
export FABRIC_CFG_PATH=${PWD}/configtx
#使用cryptogen为组织生成加密材料
cryptogen generate --config=./organizations/cryptogen/crypto-config-orderer.yaml --output=organizations
cryptogen generate --config=./organizations/cryptogen/crypto-config-org1.yaml --output=organizations
cryptogen generate --config=./organizations/cryptogen/crypto-config-org2.yaml --output=organizations
#启动网络（levelDB）
# DOCKER_SOCK=/var/run/docker.sock docker-compose -f compose/compose-test-net.yaml -f compose/docker/docker-compose-test-net.yaml up -d
#启动网络（couchDB）
DOCKER_SOCK=/var/run/docker.sock docker-compose -f compose/compose-test-net.yaml -f compose/docker/docker-compose-test-net.yaml -f compose/compose-couch.yaml -f compose/docker/docker-compose-couch.yaml up -d
#生成创世块
configtxgen -profile ChannelUsingRaft -outputBlock ./channel-artifacts/mychannel.block -channelID mychannel

#设置orderer管理员环境变量
export ORDERER_CA=${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem
export ORDERER_ADMIN_TLS_SIGN_CERT=${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/server.crt
export ORDERER_ADMIN_TLS_PRIVATE_KEY=${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/tls/server.key
#将order节点加到通道中
osnadmin channel join --channelID mychannel --config-block ./channel-artifacts/mychannel.block -o localhost:7053 --ca-file "$ORDERER_CA" --client-cert "$ORDERER_ADMIN_TLS_SIGN_CERT" --client-key "$ORDERER_ADMIN_TLS_PRIVATE_KEY"
#等待一会儿
sleep 12
#将peer节点加入通道
##设置org1管理员环境变量
##将org1的peer节点加入通道
export FABRIC_CFG_PATH=$PWD/../config/
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
peer channel join -b ./channel-artifacts/mychannel.block

##将org2的peer节点加入通道
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051
peer channel join -b ./channel-artifacts/mychannel.block
#设置组织的锚节点
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp
export CORE_PEER_ADDRESS=localhost:7051
peer channel fetch config channel-artifacts/config_block.pb -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com -c mychannel --tls --cafile "$ORDERER_CA"
cd channel-artifacts
configtxlator proto_decode --input config_block.pb --type common.Block --output config_block.json
jq '.data.data[0].payload.data.config' config_block.json > config.json
cp config.json config_copy.json
jq '.channel_group.groups.Application.groups.Org1MSP.values += {"AnchorPeers":{"mod_policy": "Admins","value":{"anchor_peers": [{"host": "peer0.org1.example.com","port": 7051}]},"version": "0"}}' config_copy.json > modified_config.json
configtxlator proto_encode --input config.json --type common.Config --output config.pb
configtxlator proto_encode --input modified_config.json --type common.Config --output modified_config.pb
configtxlator compute_update --channel_id mychannel --original config.pb --updated modified_config.pb --output config_update.pb
configtxlator proto_decode --input config_update.pb --type common.ConfigUpdate --output config_update.json
echo '{"payload":{"header":{"channel_header":{"channel_id":"mychannel", "type":2}},"data":{"config_update":'$(cat config_update.json)'}}}' | jq . > config_update_in_envelope.json
configtxlator proto_encode --input config_update_in_envelope.json --type common.Envelope --output config_update_in_envelope.pb
cd ..
peer channel update -f channel-artifacts/config_update_in_envelope.pb -c mychannel -o localhost:7050  --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
export CORE_PEER_LOCALMSPID="Org2MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp
export CORE_PEER_ADDRESS=localhost:9051

peer channel fetch config channel-artifacts/config_block.pb -o localhost:7050 --ordererTLSHostnameOverride orderer.example.com -c mychannel --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"

cd channel-artifacts
configtxlator proto_decode --input config_block.pb --type common.Block --output config_block.json
jq '.data.data[0].payload.data.config' config_block.json > config.json
cp config.json config_copy.json
jq '.channel_group.groups.Application.groups.Org2MSP.values += {"AnchorPeers":{"mod_policy": "Admins","value":{"anchor_peers": [{"host": "peer0.org2.example.com","port": 9051}]},"version": "0"}}' config_copy.json > modified_config.json
configtxlator proto_encode --input config.json --type common.Config --output config.pb
configtxlator proto_encode --input modified_config.json --type common.Config --output modified_config.pb
configtxlator compute_update --channel_id mychannel --original config.pb --updated modified_config.pb --output config_update.pb

configtxlator proto_decode --input config_update.pb --type common.ConfigUpdate --output config_update.json
echo '{"payload":{"header":{"channel_header":{"channel_id":"mychannel", "type":2}},"data":{"config_update":'$(cat config_update.json)'}}}' | jq . > config_update_in_envelope.json
configtxlator proto_encode --input config_update_in_envelope.json --type common.Envelope --output config_update_in_envelope.pb

cd ..
peer channel update -f channel-artifacts/config_update_in_envelope.pb -c mychannel -o localhost:7050  --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem"
```

### 关闭网络（总）

```bash
# DOCKER_SOCK=/var/run/docker.sock docker-compose -f compose/compose-test-net.yaml -f compose/docker/docker-compose-test-net.yaml down --volumes --remove-orphans

#关闭couchDB
# DOCKER_SOCK=/var/run/docker.sock docker-compose -f compose/compose-test-net.yaml -f compose/docker/docker-compose-test-net.yaml -f compose/compose-couch.yaml -f compose/docker/docker-compose-couch.yaml down --volumes --remove-orphans

#关闭所有运行中的容器
docker stop $(docker ps -aq)

rm -rf channel-artifacts/
rm -rf ./organizations/ordererOrganizations/
rm -rf ./organizations/peerOrganizations/
```

或者使用
`./network.sh down`
