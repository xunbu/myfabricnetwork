package com.guolong.gateway.service;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.math.BigInteger;
import java.security.MessageDigest;
import java.time.Instant;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Date;
import java.util.List;

import org.hyperledger.fabric.client.Contract;
import org.hyperledger.fabric.client.Gateway;
import org.hyperledger.fabric.protos.common.Block;
import org.hyperledger.fabric.protos.common.BlockHeader;
import org.hyperledger.fabric.protos.common.BlockchainInfo;
import org.hyperledger.fabric.protos.common.ChannelHeader;
import org.hyperledger.fabric.protos.common.ConfigEnvelope;
import org.hyperledger.fabric.protos.common.ConfigGroup;
import org.hyperledger.fabric.protos.common.Envelope;
import org.hyperledger.fabric.protos.common.HeaderType;
import org.hyperledger.fabric.protos.common.Payload;
import org.hyperledger.fabric.protos.common.SignatureHeader;
import org.hyperledger.fabric.protos.peer.ChaincodeActionPayload;
import org.hyperledger.fabric.protos.peer.ChaincodeInvocationSpec;
import org.hyperledger.fabric.protos.peer.ChaincodeProposalPayload;
import org.hyperledger.fabric.protos.peer.ProcessedTransaction;
import org.hyperledger.fabric.protos.peer.Transaction;
import org.hyperledger.fabric.protos.peer.TransactionAction;

import com.google.protobuf.ByteString;
import com.google.protobuf.InvalidProtocolBufferException;
import com.guolong.gateway.dto.BlockInfo;
import com.guolong.gateway.dto.ChainCodeInfo;
import com.guolong.gateway.dto.TxInfo;

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
        // 倒序：最新的区块在前
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

        // 解析 ChaincodeInfos
        if (channelHeader.getType() == HeaderType.ENDORSER_TRANSACTION_VALUE) {
            Transaction tx = Transaction.parseFrom(payload.getData());
            List<ChainCodeInfo> ccInfos = new ArrayList<>();
            
            for (TransactionAction action : tx.getActionsList()) {
                ChaincodeActionPayload cap = ChaincodeActionPayload.parseFrom(action.getPayload());
                ChaincodeProposalPayload cpp = ChaincodeProposalPayload.parseFrom(cap.getChaincodeProposalPayload());
                
                // 解析 Proposal 中的 Input
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
    // 核心解析逻辑
    // ==========================================
    private BlockInfo parseBlockInfo(byte[] blockBytes, long blockNum, String channelId) throws InvalidProtocolBufferException {
        Block block = Block.parseFrom(blockBytes);
        BlockHeader header = block.getHeader();
        
        BlockInfo info = new BlockInfo();
        info.setBlockNumber(blockNum);
        info.setChannelId(channelId);
        info.setBlockSize((long) blockBytes.length);
        
        // [修复] 计算正确的 Block Hash (ASN.1 DER encoding + SHA256)
        String realBlockHash = calculateBlockHash(header);
        info.setBlockHash(realBlockHash);
        
        info.setPreviousHash(toHex(header.getPreviousHash()));
        info.setMerkleRoot(toHex(header.getDataHash()));
        info.setTxCount((long) block.getData().getDataCount());
        
        List<String> txIds = new ArrayList<>();
        if (block.getData().getDataCount() > 0) {
            for (int i = 0; i < block.getData().getDataCount(); i++) {
                ByteString data = block.getData().getData(i);
                try {
                    Envelope env = Envelope.parseFrom(data);
                    Payload pl = Payload.parseFrom(env.getPayload());
                    ChannelHeader ch = ChannelHeader.parseFrom(pl.getHeader().getChannelHeader());
                    
                    txIds.add(ch.getTxId());
                    
                    // 解析时间戳 (取第一笔交易)
                    if (info.getTimestamp() == null) {
                        info.setTimestamp(Date.from(Instant.ofEpochSecond(ch.getTimestamp().getSeconds(), ch.getTimestamp().getNanos())));
                    }
                    
                    // 解析 Creator (取第一笔交易)
                    if (i == 0 && info.getBlockCreator() == null) {
                        String creator = extractCreatorFromSignature(pl.getHeader().getSignatureHeader());
                        info.setBlockCreator(creator);
                    }
                } catch (Exception e) {
                    // ignore individual tx parse errors
                }
            }
        }
        
        // 兜底：创世区块可能没有交易，补充时间戳
        if (blockNum == 0 && info.getTimestamp() == null) {
            info.setTimestamp(new Date()); 
        }

        info.setTxIds(txIds);
        return info;
    }

    // ==========================================
    // [新增] Fabric 区块哈希算法 (ASN.1 DER Encoding)
    // ==========================================
    private String calculateBlockHash(BlockHeader header) {
        try {
            long number = header.getNumber();
            byte[] previousHash = header.getPreviousHash().toByteArray();
            byte[] dataHash = header.getDataHash().toByteArray();

            // 1. 构造 ASN.1 整数 (Block Number)
            byte[] numBytes = BigInteger.valueOf(number).toByteArray();
            byte[] asn1Num = createAsn1Object(0x02, numBytes);

            // 2. 构造 ASN.1 OctetString (Previous Hash)
            byte[] asn1PrevHash = createAsn1Object(0x04, previousHash);

            // 3. 构造 ASN.1 OctetString (Data Hash)
            byte[] asn1DataHash = createAsn1Object(0x04, dataHash);

            // 4. 拼接 Sequence 内容
            ByteArrayOutputStream seqContent = new ByteArrayOutputStream();
            seqContent.write(asn1Num);
            seqContent.write(asn1PrevHash);
            seqContent.write(asn1DataHash);

            // 5. 构造 ASN.1 Sequence
            byte[] asn1Seq = createAsn1Object(0x30, seqContent.toByteArray());

            // 6. SHA-256 哈希
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hash = digest.digest(asn1Seq);

            return toHex(ByteString.copyFrom(hash));

        } catch (Exception e) {
            e.printStackTrace();
            return "";
        }
    }

    // 简单的 ASN.1 DER 编码辅助方法
    private byte[] createAsn1Object(int tag, byte[] content) throws IOException {
        ByteArrayOutputStream out = new ByteArrayOutputStream();
        out.write(tag);
        writeLength(out, content.length);
        out.write(content);
        return out.toByteArray();
    }

    // ASN.1 长度编码
    private void writeLength(ByteArrayOutputStream out, int length) {
        if (length < 128) {
            out.write(length);
        } else {
            // 长形式：第一个字节是 0x80 | 长度字节数
            int byteCount = 0;
            int temp = length;
            while (temp > 0) {
                temp >>= 8;
                byteCount++;
            }
            out.write(0x80 | byteCount);
            for (int i = byteCount - 1; i >= 0; i--) {
                out.write((length >> (i * 8)) & 0xFF);
            }
        }
    }

    private String extractCreatorFromSignature(ByteString signatureHeaderBytes) throws InvalidProtocolBufferException {
        if (signatureHeaderBytes == null || signatureHeaderBytes.isEmpty()) {
            return "";
        }
        SignatureHeader sh = SignatureHeader.parseFrom(signatureHeaderBytes);
        byte[] creatorBytes = sh.getCreator().toByteArray();
        if (creatorBytes.length > 0) {
            int len = Math.min(8, creatorBytes.length);
            byte[] sub = new byte[len];
            System.arraycopy(creatorBytes, 0, sub, 0, len);
            return "Creator:" + toHex(ByteString.copyFrom(sub));
        }
        return "";
    }

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