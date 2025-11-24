package com.guolong.restfulapi.config;

import com.guolong.gateway.config.FabricGatewayFactory;
import com.guolong.gateway.service.AdminService;
import com.guolong.gateway.service.ChaincodeService;
import com.guolong.gateway.service.LedgerService;
import io.grpc.ManagedChannel;
import org.hyperledger.fabric.client.Gateway;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class FabricBeanConfig {

    @Value("${fabric.tls-cert-path}") private String tlsCertPath;
    @Value("${fabric.peer-endpoint}") private String peerEndpoint;
    @Value("${fabric.override-auth}") private String overrideAuth;
    @Value("${fabric.msp-id}") private String mspId;
    @Value("${fabric.cert-path}") private String certDir;
    @Value("${fabric.key-path}") private String keyDir;
    
    // [新增] 注入 bin 路径
    @Value("${fabric.fabric-bin-path}") private String fabricBinPath;

    @Bean(destroyMethod = "shutdownNow")
    public ManagedChannel managedChannel() throws Exception {
        return FabricGatewayFactory.newGrpcConnection(tlsCertPath, peerEndpoint, overrideAuth);
    }

    @Bean(destroyMethod = "close")
    public Gateway gateway(ManagedChannel channel) throws Exception {
        return FabricGatewayFactory.createGateway(channel, mspId, certDir, keyDir);
    }

    @Bean
    public ChaincodeService chaincodeService(Gateway gateway) {
        return new ChaincodeService(gateway);
    }

    @Bean
    public LedgerService ledgerService(Gateway gateway) {
        return new LedgerService(gateway);
    }

    // [修改] 核心改动：将 fabricBinPath 传入构造函数
    @Bean
    public AdminService adminService(Gateway gateway) {
        return new AdminService(gateway, fabricBinPath);
    }
}