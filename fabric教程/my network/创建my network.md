# 创建my network

本章介绍如何基于`my network`文件夹，创建一个单组织(单order组织和单peer组织)，包含单order节点，三个peer节点，使用`cryptogen`与`couchDB`的hyperledger网络

## 将bin目录、FABRIC_CFG_PATH添加到系统路径

```bash
cd ~/fabric/mynetwork
export PATH=$PATH:$PWD/bin
export FABRIC_CFG_PATH=${PWD}/config
```

## 启动网络（couchDB）

```bash
export DOCKER_SOCK=/var/run/docker.sock

sudo -E docker-compose -f compose/compose-test-net.yaml -f compose/docker/docker-compose-test-net.yaml -f compose/compose-couch.yaml -f compose/docker/docker-compose-couch.yaml up -d
```

可以采用以下方法不使用sudo运行docker

```bash
# 新建docker组
# 1. 创建 docker 用户组（如果不存在），docker组默认有`/var/run/docker.sock`的权限
sudo groupadd docker

# 2. 将当前用户添加到 docker 组（-aG 表示追加，不影响用户其他组）
sudo usermod -aG docker $USER

# 3. 重启 Docker 服务，使组权限更改生效
sudo systemctl restart docker

# 4. 放宽 /var/run/docker.sock 的权限（允许所有用户读写，可能有安全风险，一般前三步即可）
sudo chmod a+rw /var/run/docker.sock
```

## 删除网络

```bash
export DOCKER_SOCK=/var/run/docker.sock
# -v选项清除用以持久化的卷，下次再启动时是一个全新的网络
docker-compose -f compose/compose-test-net.yaml -f compose/docker/docker-compose-test-net.yaml -f compose/compose-couch.yaml -f compose/docker/docker-compose-couch.yaml down -v
```

## 设置orderer管理员环境变量

```bash
export ORDERER_CA=${PWD}/organizations/ordererOrganizations/guolong.com/orderers/orderer.guolong.com/msp/tlscacerts/tlsca.guolong.com-cert.pem
export ORDERER_ADMIN_TLS_SIGN_CERT=${PWD}/organizations/ordererOrganizations/guolong.com/orderers/orderer.guolong.com/tls/server.crt
export ORDERER_ADMIN_TLS_PRIVATE_KEY=${PWD}/organizations/ordererOrganizations/guolong.com/orderers/orderer.guolong.com/tls/server.key
```

## 生成通道mychannel

```bash
#生成应用通道创世块
configtxgen -profile ChannelUsingRaft -outputBlock ./channel-artifacts/mychannel.block -channelID mychannel
#orderer加入通道
osnadmin channel join --channelID mychannel --config-block ./channel-artifacts/mychannel.block -o localhost:7053 --ca-file "$ORDERER_CA" --client-cert "$ORDERER_ADMIN_TLS_SIGN_CERT" --client-key "$ORDERER_ADMIN_TLS_PRIVATE_KEY"
```

## 查看orderer上的通道

```bash
osnadmin channel list -o localhost:7053 --ca-file "$ORDERER_CA" --client-cert "$ORDERER_ADMIN_TLS_SIGN_CERT" --client-key "$ORDERER_ADMIN_TLS_PRIVATE_KEY"
```

## 设置org1管理员环境变量(peer0)

```bash
export CORE_PEER_TLS_ENABLED=true

export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/guolong.com/peers/peer0.guolong.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/guolong.com/users/Admin@guolong.com/msp
export CORE_PEER_ADDRESS=localhost:7051
```

## 将peer0节点加入mychannel

```bash
peer channel join -b ./channel-artifacts/mychannel.block
```

## 将peer1/2节点加入mychannel

```bash
export CORE_PEER_ADDRESS=localhost:8051
peer channel join -b ./channel-artifacts/mychannel.block
export CORE_PEER_ADDRESS=localhost:9051
peer channel join -b ./channel-artifacts/mychannel.block
```

## 将peer0设为锚节点

```bash
# 使用 peer channel fetch 命令拉取最新的通道配置区块
peer channel fetch config channel-artifacts/config_block.pb -o localhost:7050 --ordererTLSHostnameOverride orderer.guolong.com -c mychannel --tls --cafile "$ORDERER_CA"
# 切换到channel-artifacts
cd channel-artifacts
#将区块从 protobuf 格式解码为可读取和编辑的 JSON 对象。同时我们会剥离不必要的区块数据，仅保留通道配置部分。
configtxlator proto_decode --input config_block.pb --type common.Block --output config_block.json
jq '.data.data[0].payload.data.config' config_block.json > config.json
cp config.json config_copy.json
#将 org1的锚节点配置写入通道配置
jq '.channel_group.groups.Application.groups.Org1MSP.values += {"AnchorPeers":{"mod_policy": "Admins","value":{"anchor_peers": [{"host": "peer0.guolong.com","port": 7051}]},"version": "0"}}' config_copy.json > modified_config.json
#将原始和修改后的通道配置重新转换为 protobuf 格式，并计算它们之间的差异。
configtxlator proto_encode --input config.json --type common.Config --output config.pb
configtxlator proto_encode --input modified_config.json --type common.Config --output modified_config.pb
configtxlator compute_update --channel_id mychannel --original config.pb --updated modified_config.pb --output config_update.pb
#将配置更新包装在一个交易信封中
configtxlator proto_decode --input config_update.pb --type common.ConfigUpdate --output config_update.json
echo '{"payload":{"header":{"channel_header":{"channel_id":"mychannel", "type":2}},"data":{"config_update":'$(cat config_update.json)'}}}' | jq . > config_update_in_envelope.json
configtxlator proto_encode --input config_update_in_envelope.json --type common.Envelope --output config_update_in_envelope.pb
#提交通道配置更新
cd ..
peer channel update -f channel-artifacts/config_update_in_envelope.pb -c mychannel -o localhost:7050  --ordererTLSHostnameOverride orderer.guolong.com --tls --cafile "${PWD}/organizations/ordererOrganizations/guolong.com/orderers/orderer.guolong.com/msp/tlscacerts/tlsca.guolong.com-cert.pem"
```
