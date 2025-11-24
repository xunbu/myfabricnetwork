package com.guolong.gateway.service;

import com.google.protobuf.ByteString;
import com.google.protobuf.InvalidProtocolBufferException;
import com.guolong.gateway.dto.BlockInfo;
import com.guolong.gateway.dto.ChainCodeInfo;
import com.guolong.gateway.dto.TxInfo;
import org.hyperledger.fabric.client.Contract;
import org.hyperledger.fabric.client.Gateway;

// ==========================================
// 核心修正：直接导入具体的 Protobuf 类
// ==========================================
import org.hyperledger.fabric.protos.common.Block;
import org.hyperledger.fabric.protos.common.BlockHeader;
import org.hyperledger.fabric.protos.common.BlockchainInfo;
import org.hyperledger.fabric.protos.common.ChannelHeader;
import org.hyperledger.fabric.protos.common.ConfigEnvelope;
import org.hyperledger.fabric.protos.common.ConfigGroup;
import org.hyperledger.fabric.protos.common.Envelope;
import org.hyperledger.fabric.protos.common.HeaderType;
import org.hyperledger.fabric.protos.common.Payload;
import org.hyperledger.fabric.protos.peer.ChaincodeActionPayload;
import org.hyperledger.fabric.protos.peer.ChaincodeInvocationSpec;
import org.hyperledger.fabric.protos.peer.ChaincodeProposalPayload;
import org.hyperledger.fabric.protos.peer.ProcessedTransaction;
import org.hyperledger.fabric.protos.peer.Transaction;
import org.hyperledger.fabric.protos.peer.TransactionAction;

import java.time.Instant;
import java.util.ArrayList;
import java.util.Base64;
import java.util.Collections;
import java.util.Date;
import java.util.List;

public class LedgerService {
    private final Gateway gateway;

    public LedgerService(Gateway gateway) {
        this.gateway = gateway;
    }

    private Contract getQscc(String channelName) {
        return gateway.getNetwork(channelName).getContract("qscc");
    }

    private Contract getCscc(String channelName) {
        return gateway.getNetwork(channelName).getContract("cscc");
    }

    // 获取区块高度
    public long getBlockHeight(String channelName) throws Exception {
        byte[] bytes = getQscc(channelName).evaluateTransaction("GetChainInfo", channelName);
        // 修正：直接使用 BlockchainInfo，去掉 Ledger 前缀
        BlockchainInfo info = BlockchainInfo.parseFrom(bytes);
        return info.getHeight();
    }

    // 获取交易总数
    public long getTransactionCount(String channelName) throws Exception {
        long height = getBlockHeight(channelName);
        long totalTx = 0;
        for (long i = 0; i < height; i++) {
            byte[] blockBytes = getQscc(channelName).evaluateTransaction("GetBlockByNumber", channelName, String.valueOf(i));
            // 修正：直接使用 Block，去掉 Common 前缀
            Block block = Block.parseFrom(blockBytes);
            totalTx += block.getData().getDataCount();
        }
        return totalTx;
    }

    // 获取组织数量 (解析 ConfigBlock)
    public int getOrganizationCount(String channelName) throws Exception {
        byte[] configBlockBytes = getCscc(channelName).evaluateTransaction("GetConfigBlock", channelName);
        Block configBlock = Block.parseFrom(configBlockBytes);
        
        if (configBlock.getData().getDataCount() == 0) return 0;

        Envelope envelope = Envelope.parseFrom(configBlock.getData().getData(0));
        Payload payload = Payload.parseFrom(envelope.getPayload());
        // 修正：直接使用 ConfigEnvelope，去掉 Configtx 前缀
        ConfigEnvelope configEnvelope = ConfigEnvelope.parseFrom(payload.getData());
        
        ConfigGroup channelGroup = configEnvelope.getConfig().getChannelGroup();
        ConfigGroup orgsGroup = null;

        if (channelGroup.getGroupsMap().containsKey("Application")) {
            orgsGroup = channelGroup.getGroupsMap().get("Application");
        } else if (channelGroup.getGroupsMap().containsKey("Consortiums")) {
            for (ConfigGroup group : channelGroup.getGroupsMap().get("Consortiums").getGroupsMap().values()) {
                orgsGroup = group;
                break;
            }
        }

        if (orgsGroup == null) return 0;
        return orgsGroup.getGroupsCount();
    }

    // 根据区块号获取区块信息
    public BlockInfo getBlockByNum(String channelName, long blockNum) throws Exception {
        byte[] blockBytes = getQscc(channelName).evaluateTransaction("GetBlockByNumber", channelName, String.valueOf(blockNum));
        return parseBlockInfo(blockBytes, blockNum, channelName);
    }

    // 分页获取区块列表
    public List<BlockInfo> getBlockListByPage(String channelName, long pageNum, long pageSize) throws Exception {
        long height = getBlockHeight(channelName);
        if (height == 0) return Collections.emptyList();

        long startIdx = pageNum * pageSize;
        if (startIdx >= height) return Collections.emptyList();

        long endIdx = startIdx + pageSize - 1;
        if (endIdx >= height) endIdx = height - 1;

        List<BlockInfo> list = new ArrayList<>();
        for (long i = endIdx; i >= startIdx; i--) {
            list.add(getBlockByNum(channelName, i));
        }
        return list;
    }
    
    // 根据交易ID查询详情
    public TxInfo getTxById(String channelName, String txId) throws Exception {
        byte[] bytes = getQscc(channelName).evaluateTransaction("GetTransactionByID", channelName, txId);
        // 修正：直接使用 ProcessedTransaction，去掉 TransactionPackage 前缀
        ProcessedTransaction processedTx = ProcessedTransaction.parseFrom(bytes);
        
        TxInfo txInfo = new TxInfo();
        txInfo.setValidationCode(processedTx.getValidationCode());
        txInfo.setSize(bytes.length);

        Envelope envelope = processedTx.getTransactionEnvelope();
        Payload payload = Payload.parseFrom(envelope.getPayload());
        ChannelHeader channelHeader = ChannelHeader.parseFrom(payload.getHeader().getChannelHeader());

        txInfo.setTxId(channelHeader.getTxId());
        txInfo.setChannelId(channelHeader.getChannelId());
        // 修正：直接使用 HeaderType
        txInfo.setType(HeaderType.forNumber(channelHeader.getType()).name());
        txInfo.setTimestamp(Date.from(Instant.ofEpochSecond(channelHeader.getTimestamp().getSeconds(), channelHeader.getTimestamp().getNanos())));

        // 修正：直接使用 HeaderType.ENDORSER_TRANSACTION_VALUE
        if (channelHeader.getType() == HeaderType.ENDORSER_TRANSACTION_VALUE) {
            Transaction tx = Transaction.parseFrom(payload.getData());
            
            List<ChainCodeInfo> ccInfos = new ArrayList<>();
            for (TransactionAction action : tx.getActionsList()) {
                ChaincodeActionPayload cap = ChaincodeActionPayload.parseFrom(action.getPayload());
                ChaincodeProposalPayload cpp = ChaincodeProposalPayload.parseFrom(cap.getChaincodeProposalPayload());
                ChaincodeInvocationSpec cis = ChaincodeInvocationSpec.parseFrom(cpp.getInput());
                
                ChainCodeInfo ccInfo = new ChainCodeInfo();
                ccInfo.setChainCodeName(cis.getChaincodeSpec().getChaincodeId().getName());
                List<String> args = new ArrayList<>();
                for (ByteString arg : cis.getChaincodeSpec().getInput().getArgsList()) {
                    args.add(arg.toStringUtf8());
                }
                ccInfo.setArgs(args);
                ccInfos.add(ccInfo);
            }
            txInfo.setChainCodeInfos(ccInfos);
        }
        
        return txInfo;
    }

    // 解析区块辅助方法
    private BlockInfo parseBlockInfo(byte[] blockBytes, long blockNum, String channelId) throws InvalidProtocolBufferException {
        Block block = Block.parseFrom(blockBytes);
        BlockHeader header = block.getHeader();
        
        BlockInfo info = new BlockInfo();
        info.setBlockNumber(blockNum);
        info.setChannelId(channelId);
        info.setBlockHash(Base64.getEncoder().encodeToString(header.getDataHash().toByteArray()));
        info.setPreviousHash(Base64.getEncoder().encodeToString(header.getPreviousHash().toByteArray()));
        info.setTxCount((long) block.getData().getDataCount());
        
        if (block.getData().getDataCount() > 0) {
            try {
                Envelope env = Envelope.parseFrom(block.getData().getData(0));
                Payload pl = Payload.parseFrom(env.getPayload());
                ChannelHeader ch = ChannelHeader.parseFrom(pl.getHeader().getChannelHeader());
                info.setTimestamp(Date.from(Instant.ofEpochSecond(ch.getTimestamp().getSeconds(), ch.getTimestamp().getNanos())));
            } catch (Exception e) {
                // ignore
            }
        }
        return info;
    }
}