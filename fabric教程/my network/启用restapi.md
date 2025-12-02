# 启用mynetwork的监控界面

## 预先步骤

### 下载安装go
```bash
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.25.4.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin #在`$HOME/.profile` 或 `/etc/profile` 添加环境变量 
go version #检查是否安装成功
```


### 下载安装mysql
```bash
sudo apt update
sudo apt install mysql-server
sudo systemctl start mysql #启动mysql服务
sudo systemctl enable mysql #开机自启
```

### mysql新建用户并创建一个数据库
```bash
#进入mysql
sudo mysql
#新增用户
CREATE USER '用户名'@'主机名' IDENTIFIED WITH mysql_native_password BY '密码';
##CREATE USER 'testuser'@'localhost' IDENTIFIED BY '123456';
#赋予权限
GRANT ALL PRIVILEGES ON *.* TO '用户名'@'主机名' WITH GRANT OPTION;
#刷新权限
FLUSH PRIVILEGES;
#创建数据库
CREATE DATABASE mynetwork;
```

## 开启界面

```bash
cd /home/qinhan/fabric/mynetwork/restapi/restapi-go
# 修改main.go中mysql对应的变量
go run .
```
