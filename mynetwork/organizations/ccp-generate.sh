#!/bin/bash

function one_line_pem {
    echo "`awk 'NF {sub(/\\n/, ""); printf "%s\\\\\\\n",$0;}' $1`"
}

function yaml_ccp {
    local PP=$(one_line_pem $1)
    local CP=$(one_line_pem $2)
    # 将模板中的占位符替换为实际的证书内容
    sed -e "s#\${PEERPEM}#$PP#" \
        -e "s#\${CAPEM}#$CP#" \
        ccp-template.yaml | sed -e $'s/\\\\n/\\\n          /g'
}

# 1. 定义证书路径 (基于你的 tree 结构，相对路径)
# Peer 组织的 TLS CA 证书
PEERPEM=peerOrganizations/guolong.com/tlsca/tlsca.guolong.com-cert.pem
# CA 组织的证书
CAPEM=peerOrganizations/guolong.com/ca/ca.guolong.com-cert.pem

# 2. 生成文件
echo "Generating connection profile for Org1..."
# 输出到 organizations/peerOrganizations/guolong.com/connection-org1.yaml
echo "$(yaml_ccp $PEERPEM $CAPEM)" > peerOrganizations/guolong.com/connection-org1.yaml

echo "Done. File saved to: peerOrganizations/guolong.com/connection-org1.yaml"