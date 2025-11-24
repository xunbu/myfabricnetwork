package com.guolong.gateway.config;

import com.guolong.gateway.utils.FileUtils;
import io.grpc.Grpc;
import io.grpc.ManagedChannel;
import io.grpc.TlsChannelCredentials;
import org.hyperledger.fabric.client.Gateway;
import org.hyperledger.fabric.client.identity.Identities;
import org.hyperledger.fabric.client.identity.Identity;
import org.hyperledger.fabric.client.identity.Signer;
import org.hyperledger.fabric.client.identity.Signers;
import org.hyperledger.fabric.client.identity.X509Identity;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.security.InvalidKeyException; // <--- [新增 import]
import java.security.PrivateKey;
import java.security.cert.CertificateException;
import java.security.cert.X509Certificate;
import java.util.concurrent.TimeUnit;

public class FabricGatewayFactory {

    public static ManagedChannel newGrpcConnection(String tlsCertPath, String peerEndpoint, String overrideAuth) throws IOException {
        // ... (保持不变) ...
        Path tlsCertFile = Paths.get(tlsCertPath);
        if (!Files.exists(tlsCertFile)) {
            throw new IOException("TLS证书不存在: " + tlsCertPath);
        }

        return Grpc.newChannelBuilder(peerEndpoint, TlsChannelCredentials.newBuilder()
                        .trustManager(tlsCertFile.toFile())
                        .build())
                .overrideAuthority(overrideAuth)
                .build();
    }

    // [修改] 添加 InvalidKeyException 到 throws 列表中
    public static Gateway createGateway(ManagedChannel channel, String mspId, String certDir, String keyDir) 
            throws IOException, CertificateException, InvalidKeyException { 
        
        Path certPath = FileUtils.getFirstFile(certDir);
        Path keyPath = FileUtils.getFirstFile(keyDir);

        X509Certificate certificate = Identities.readX509Certificate(Files.newBufferedReader(certPath));
        PrivateKey privateKey = Identities.readPrivateKey(Files.newBufferedReader(keyPath));

        Identity identity = new X509Identity(mspId, certificate);
        Signer signer = Signers.newPrivateKeySigner(privateKey);

        return Gateway.newInstance()
                .identity(identity)
                .signer(signer)
                .connection(channel)
                .evaluateOptions(options -> options.withDeadlineAfter(5, TimeUnit.SECONDS))
                .endorseOptions(options -> options.withDeadlineAfter(15, TimeUnit.SECONDS))
                .submitOptions(options -> options.withDeadlineAfter(5, TimeUnit.SECONDS))
                .commitStatusOptions(options -> options.withDeadlineAfter(1, TimeUnit.MINUTES))
                .connect();
    }
}