# 如何创建两个orderer节点

## 修改`organizations/cryptogen/crypto-config-orderer.yaml`

在`organizations/cryptogen/crypto-config-orderer.yaml`末尾附加：

```yaml
- Hostname: orderer1
  SANS:
    - localhost
```

## 修改`compose/compose-test-net.yaml`
复制order.guolong.com，修改服务名、容器名、映射

## 修改`config/configtx.yaml`

