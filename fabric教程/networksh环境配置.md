# 使用networksh建立网络

目标是使用`fabric-samples/test-network/network.sh`建立一个有3个peer节点（其中org1两个peer节点，org2一个peer节点）、一个排序节点的网络(可选：使用couchDB)。

以下所有操作在fabric-samples/test-network下进行
`cd /home/xunbu/fabric/fabric-samples/test-network`

## 确保环境干净

`./network.sh down`
`export PATH=$PATH:$PWD/../bin`

## 一、修改为peer节点生成加密材料的配置

### 修改`organizations/cryptogen/crypto-config-org1.yaml`

找到 PeerOrgs: 部分，将 Template 下的 Count 从 1 修改为 2。

保存并关闭文件。这样，当我们后续运行 cryptogen 时（通常由 ./network.sh 脚本自动调用），它就会在 organizations/peerOrganizations/org1.example.com/peers/ 目录下创建 peer0.org1.example.com 和 peer1.org1.example.com 两个子目录，并包含它们各自的证书和密钥。

## 二、在 Docker Compose 中定义新 Peer 服务

### 编辑 `compose/compose-test-net.yaml` 配置文件

打开`compose/compose-test-net.yaml`

将`peer0.org1.example.com`复制一份到下方。并做以下修改

1. 顶层的`volumes`添加`peer1.org1.example.com:`
2. 服务名改为`peer1.org1.example.com`
3. `container_name`改为`peer1.org1.example.com`
4. `CORE_PEER_ID`改为`peer1.org1.example.com`
5. `CORE_PEER_ADDRESS`改为`peer1.org1.example.com:8051`
6. `CORE_PEER_LISTENADDRESS`改为`0.0.0.0:8051`
7. `CORE_PEER_CHAINCODEADDRESS`改为`peer1.org1.example.com:8052`
8. `CORE_PEER_CHAINCODELISTENADDRESS`改为`0.0.0.0:8052`
9. `CORE_PEER_GOSSIP_EXTERNALENDPOINT`改为`peer1.org1.example.com:8051`
10. `CORE_PEER_OPERATIONS_LISTENADDRESS`改为`peer1.org1.example.com:10051`
11. `volumes`：将所有路径中的 `peer0.org1.example.com` 修改为 `peer1.org1.example.com`。
12. `ports`：修改端口映射，将 `7051:7051` 修改为 `8051:8051`，将 `9444:9444` 修改为 `10051:10051`

### 编辑 `compose/docker/docker-compose-test-net.yaml` 配置文件

打开`compose/docker/docker-compose-test-net.yaml`

将`peer0.org1.example.com`复制一份到下方。并将`peer0`改为`peer1`

### 启用couchDB（可选）

 `/compose/compose-couch.yaml` 配置文件中新增`couchdb2`,并作对应修改，`ports`改成`"9984:5984"`

编辑`compose/docker/docker-compose-couch.yaml`配置文件

## 三、启动网络并加入通道

启动网络并创建通道
`./network.sh up createChannel`

`./network.sh up createChannel -s couchdb`（使用couchDB）

脚本只会将peer0.org1和peer0.org2加入网络，我们需要手动将peer1.org1加入通道。

```bash
# 切换到 fabric-samples/test-network 目录

# 设置环境变量指向 Org1
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp

# 关键：将地址指向我们的新 Peer
export CORE_PEER_ADDRESS=localhost:8051 


#将core.yaml添加到环境变量
export FABRIC_CFG_PATH=$PWD/../config/

# 执行加入通道的命令
peer channel join -b ./channel-artifacts/mychannel.block
```

## 验证

`peer channel getinfo -c mychannel`

## 关闭网络

### 编辑`bft`配置文件

`network.sh`脚本中的`networkDown`函数存在一个Bug，它没有检查你启动时用的是哪个模式，而是硬编码、错误地去使用`/compose/docker/docker-compose-bft-test-net.yaml`和`compose/compose-bft-test-net.yaml`文件来执行关闭操作。因此也需要做相应修改。

#### 编辑`/compose/docker/docker-compose-bft-test-net.yaml`配置文件

打开`fabric-samples/test-network/compose/docker/docker-compose-bft-test-net.yaml`
将`peer0.org1.example.com`复制一份到下方。并将`peer0`改为`peer1`

#### 编辑`compose/compose-bft-test-net.yaml`配置文件

在`volumes`中添加`peer1.org1.example.com:`

## 关闭

```bash
./network.sh down
# docker volume rm compose_peer1.org1.example.com
```
