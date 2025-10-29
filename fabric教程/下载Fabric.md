# 下载Fabric

以下代码都运行在linux中

## 前置安装

### 升级apt

`apt update`

### 安装git

`sudo apt install git`

### 安装curl

`sudo apt install curl`

### 安装docker并运行

`sudo apt -y install docker-compose`

`sudo systemctl start docker`

### 安装jq

`sudo apt install jq`

### 安装go

在[https://go.dev/dl/](https://go.dev/dl/)下载安装包go1.24.4.linux-amd64.tar.gz放在服务器任意目录内

进入安装包所在目录，运行以下命令：

```bash
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.24.4.linux-amd64.tar.gz
```

添加go二进制文件环境变量,在`~/.profile`或`/etc/profile`末尾加上：

`export PATH=$PATH:/usr/local/go/bin`

## 下载fabric v2.5

获取下载脚本

```bash
curl -sSLO https://raw.githubusercontent.com/hyperledger/fabric/main/scripts/install-fabric.sh && chmod +x install-fabric.sh
```

下载hyperledger-fabric v2.5.12的镜像、示例和二进制文件

```bash
./install-fabric.sh --fabric-version 2.5.12 docker samples binary
```

hyperledger-fabric即下载到`fabric-samples`中
