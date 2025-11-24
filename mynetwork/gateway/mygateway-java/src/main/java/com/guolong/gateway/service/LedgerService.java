package com.guolong.gateway.service;

import com.google.protobuf.ByteString;
import com.google.protobuf.InvalidProtocolBufferException;
import com.guolong.gateway.dto.BlockInfo;
import com.guolong.gateway.dto.ChainCodeInfo;
import com.guolong.gateway.dto.TxInfo;
import org.hyperledger.fabric.client.Contract;
import org.hyperledger.fabric.client.Gateway;

// 引入 Protobuf 类
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

    public long getBlockHeight(String channelName) throws Exception {
        byte[] bytes = getQscc(channelName).evaluateTransaction("GetChainInfo", channelName);
        BlockchainInfo info = BlockchainInfo.parseFrom(bytes);
        return info.getHeight();
    }

    public long getTransactionCount(String channelName) throws Exception {
        long height = getBlockHeight(channelName);
        long totalTx = 0;
        for (long i = 0; i < height; i++) {
            // 为了性能，这里只获取区块头可能会更快，但 QSCC GetBlockByNumber 返回全量块
            // 在生产环境中，通常会缓存这个总数，而不是每次遍历
            byte[] blockBytes = getQscc(channelName).evaluateTransaction("GetBlockByNumber", channelName, String.valueOf(i));
            Block block = Block.parseFrom(blockBytes);
            totalTx += block.getData().getDataCount();
        }
        return totalTx;
    }

    public int getOrganizationCount(String channelName) throws Exception {
        byte[] configBlockBytes = getCscc(channelName).evaluateTransaction("GetConfigBlock", channelName);
        Block configBlock = Block.parseFrom(configBlockBytes);
        
        if (configBlock.getData().getDataCount() == 0) return 0;

        Envelope envelope = Envelope.parseFrom(configBlock.getData().getData(0));
        Payload payload = Payload.parseFrom(envelope.getPayload());
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

    public BlockInfo getBlockByNum(String channelName, long blockNum) throws Exception {
        byte[] blockBytes = getQscc(channelName).evaluateTransaction("GetBlockByNumber", channelName, String.valueOf(blockNum));
        return parseBlockInfo(blockBytes, blockNum, channelName);
    }

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
    
    public TxInfo getTxById(String channelName, String txId) throws Exception {
        byte[] bytes = getQscc(channelName).evaluateTransaction("GetTransactionByID", channelName, txId);
        ProcessedTransaction processedTx = ProcessedTransaction.parseFrom(bytes);
        
        TxInfo txInfo = new TxInfo();
        txInfo.setValidationCode(processedTx.getValidationCode());
        txInfo.setSize(bytes.length);

        Envelope envelope = processedTx.getTransactionEnvelope();
        Payload payload = Payload.parseFrom(envelope.getPayload());
        ChannelHeader channelHeader = ChannelHeader.parseFrom(payload.getHeader().getChannelHeader());

        txInfo.setTxId(channelHeader.getTxId());
        txInfo.setChannelId(channelHeader.getChannelId());
        txInfo.setType(HeaderType.forNumber(channelHeader.getType()).name());
        txInfo.setTimestamp(Date.from(Instant.ofEpochSecond(channelHeader.getTimestamp().getSeconds(), channelHeader.getTimestamp().getNanos())));

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

    // ==========================================
    // 核心解析逻辑更新
    // ==========================================
    private BlockInfo parseBlockInfo(byte[] blockBytes, long blockNum, String channelId) throws InvalidProtocolBufferException {
        Block block = Block.parseFrom(blockBytes);
        BlockHeader header = block.getHeader();
        
        BlockInfo info = new BlockInfo();
        info.setBlockNumber(blockNum);
        info.setChannelId(channelId);
        
        // [新增] 区块大小
        info.setBlockSize((long) blockBytes.length);
        
        // [更新] 使用 Hex 格式 (更符合区块链浏览器习惯)，如果想用 Base64 可以换回去
        info.setBlockHash(toHex(header.getDataHash()));
        info.setPreviousHash(toHex(header.getPreviousHash()));
        
        // [新增] Merkle Root (即 DataHash)
        info.setMerkleRoot(toHex(header.getDataHash()));
        
        info.setTxCount((long) block.getData().getDataCount());
        
        // [新增] 解析所有交易 ID 和时间戳
        List<String> txIds = new ArrayList<>();
        if (block.getData().getDataCount() > 0) {
            for (ByteString data : block.getData().getDataList()) {
                try {
                    Envelope env = Envelope.parseFrom(data);
                    Payload pl = Payload.parseFrom(env.getPayload());
                    ChannelHeader ch = ChannelHeader.parseFrom(pl.getHeader().getChannelHeader());
                    
                    // 收集 TxID
                    txIds.add(ch.getTxId());
                    
                    // 使用第一笔交易的时间作为区块时间
                    if (info.getTimestamp() == null) {
                        info.setTimestamp(Date.from(Instant.ofEpochSecond(ch.getTimestamp().getSeconds(), ch.getTimestamp().getNanos())));
                    }
                } catch (Exception e) {
                    // ignore invalid tx in block parsing
                }
            }
        }
        info.setTxIds(txIds);
        
        return info;
    }

    // 辅助方法：ByteString 转 Hex 字符串
    private String toHex(ByteString byteString) {
        if (byteString == null) return "";
        byte[] bytes = byteString.toByteArray();
        StringBuilder sb = new StringBuilder();
        for (byte b : bytes) {
            sb.append(String.format("%02x", b));
        }
        return sb.toString();
    }
}