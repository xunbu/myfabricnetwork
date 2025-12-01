package gateway

import (
	"crypto/x509"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NewGrpcConnection 创建到网关服务器的gRPC连接
func NewGrpcConnection(tlsCertPath, gatewayPeer, peerEndpoint string) (*grpc.ClientConn, error) {
	certificatePEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		return nil, fmt.Errorf("读取TLS证书文件失败: %w", err)
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		return nil, fmt.Errorf("创建gRPC连接失败: %w", err)
	}

	return connection, nil
}

// GetGateway 创建并返回Gateway实例
func GetGateway(clientConnection *grpc.ClientConn, mspID, cryptoPath, certPath, keyPath string) (*client.Gateway, error) {
	id, err := newIdentity(certPath, mspID)
	if err != nil {
		return nil, fmt.Errorf("创建身份失败: %w", err)
	}

	sign, err := newSign(keyPath)
	if err != nil {
		return nil, fmt.Errorf("创建签名器失败: %w", err)
	}

	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(clientConnection),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("连接到gateway失败: %w", err)
	}

	return gw, nil
}

func newIdentity(certPath, mspID string) (*identity.X509Identity, error) {
	certificatePEM, err := readFirstFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("读取证书文件失败: %w", err)
	}
	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}
	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		return nil, fmt.Errorf("创建X509身份失败: %w", err)
	}
	return id, nil
}

func newSign(keyPath string) (identity.Sign, error) {
	privateKeyPEM, err := readFirstFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("读取私钥文件失败: %w", err)
	}
	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %w", err)
	}
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, fmt.Errorf("创建签名器失败: %w", err)
	}
	return sign, nil
}

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
