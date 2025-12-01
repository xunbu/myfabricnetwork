# 如何下载运行caliper

## 使用nodejs

### 预先步骤（下载安装nodejs\nvm 下载caliper模板配置）
```bash
#下载安装nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.5/install.sh | bash
source ~/.bashrc
#安装nodejs
nvm install --lts
```

### 安装caliper

```bash
#下载caliper模板配置
git clone https://github.com/hyperledger-caliper/caliper-benchmarks.git
#后面的命令都要在该目录下完成
cd caliper-benchmarks
#全局安装
npm install --only=prod @hyperledger/caliper-cli@0.7.1
#下载fabric需要的sdk
npx caliper bind --caliper-bind-sut fabric:fabric-gateway
```

第一次使用要在`/etc/hosts`文件中添加所有节点的DNS

```bash
sudo nano /etc/hosts
# 添加一行127.0.0.1 peer0.guolong.com peer1.guolong.com peer2.guolong.com orderer.guolong.com
```

### 使用caliper（配置文件略）
```bash
# 注意修改对应路径
npx caliper launch manager \
    --caliper-workspace . \
    --caliper-benchconfig benchmarks/scenario/my-test/config.yaml \
    --caliper-networkconfig networks/fabric/test-network.yaml
```