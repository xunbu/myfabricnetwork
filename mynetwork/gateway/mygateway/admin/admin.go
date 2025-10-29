package admin

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path"

	"github.com/hyperledger/fabric-admin-sdk/pkg/chaincode"
	"github.com/hyperledger/fabric-admin-sdk/pkg/identity"
	"google.golang.org/grpc"
)

// 以下为获取peer和gatewayadmin的代码
func GetPeer(conn *grpc.ClientConn, mspID, cryptoPath, certPath, keyPath string) (*chaincode.Peer, error) {
	id, err := identity.NewPrivateKeySigningIdentity(mspID, readCertificate(certPath), readPrivateKey(keyPath))
	if err != nil {
		return nil, fmt.Errorf("error in getpeer %w", err)
	}
	return chaincode.NewPeer(conn, id), nil
}
func GetGatewayAdmin(conn *grpc.ClientConn, mspID, cryptoPath, certPath, keyPath string) *chaincode.Gateway {
	id, err := identity.NewPrivateKeySigningIdentity(mspID, readCertificate(certPath), readPrivateKey(keyPath))
	if err != nil {
		panic(err)
	}
	return chaincode.NewGateway(conn, id)
}
func readCertificate(certPath string) *x509.Certificate {
	certificatePEM, err := readFirstFile(certPath)
	if err != nil {
		panic(err)
	}
	block, _ := pem.Decode([]byte(certificatePEM))
	if block == nil {
		panic("failed to parse certificate PEM")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic("failed to parse certificate: " + err.Error())
	}

	return certificate
}

func readPrivateKey(keyPath string) crypto.PrivateKey {
	privateKeyPEM, err := readFirstFile(keyPath)
	if err != nil {
		panic(err)
	}

	block, _ := pem.Decode(privateKeyPEM)
	if block == nil {
		panic("failed to parse private key PEM")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		panic("failed to parse PKCS8 encoded private key: " + err.Error())
	}

	return privateKey
}

// readFirstFile 读取目录中的第一个文件
func readFirstFile(dirPath string) ([]byte, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, fmt.Errorf("打开目录失败: %w", err)
	}

	fileNames, err := dir.Readdirnames(1)
	if err != nil {
		return nil, fmt.Errorf("读取目录内容失败: %w", err)
	}

	if len(fileNames) == 0 {
		return nil, fmt.Errorf("目录中没有文件")
	}

	data, err := os.ReadFile(path.Join(dirPath, fileNames[0]))
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	return data, nil
}

// 以下为获取信息的代码
func GetChaincodeCount(peer *chaincode.Peer) (uint64, error) {
	v, err := peer.QueryInstalled(context.Background())
	if err != nil {
		return 0, fmt.Errorf("error in QueryInstalled,%w", err)
	}
	return uint64(len(v.InstalledChaincodes)), nil
}
