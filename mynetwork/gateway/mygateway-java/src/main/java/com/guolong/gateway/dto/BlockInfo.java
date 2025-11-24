package com.guolong.gateway.dto;

import java.util.Date;
import java.util.List;
import java.util.Objects;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonInclude;
import com.fasterxml.jackson.annotation.JsonProperty;

// 对应 Go 的 omitempty：如果字段为 null，不包含在 JSON 中
@JsonInclude(JsonInclude.Include.NON_NULL)
public class BlockInfo {

    @JsonProperty("blockHash")
    private String blockHash;

    @JsonProperty("previousHash")
    private String previousHash;

    // Go代码中 Struct 字段是 MerkleRoot, 但 json tag 是 "dataHash"
    @JsonProperty("dataHash")
    private String merkleRoot;

    @JsonProperty("blockNumber")
    private Long blockNumber; // 使用 Long 对应 uint64

    @JsonProperty("txCount")
    private Long txCount;

    @JsonProperty("blockSize")
    private Long blockSize;

    @JsonProperty("timestamp")
    @JsonFormat(shape = JsonFormat.Shape.STRING, pattern = "yyyy-MM-dd'T'HH:mm:ss.SSSX", timezone = "UTC")
    private Date timestamp;

    @JsonProperty("channelId")
    private String channelId;

    @JsonProperty("blockCreator")
    private String blockCreator;

    @JsonProperty("txIds")
    private List<String> txIds;

    // ==================== 构造函数 ====================

    public BlockInfo() {
    }

    public BlockInfo(String blockHash, String previousHash, String merkleRoot, Long blockNumber, Long txCount, Long blockSize, Date timestamp, String channelId, String blockCreator, List<String> txIds) {
        this.blockHash = blockHash;
        this.previousHash = previousHash;
        this.merkleRoot = merkleRoot;
        this.blockNumber = blockNumber;
        this.txCount = txCount;
        this.blockSize = blockSize;
        this.timestamp = timestamp;
        this.channelId = channelId;
        this.blockCreator = blockCreator;
        this.txIds = txIds;
    }

    // ==================== Getter / Setter ====================

    public String getBlockHash() {
        return blockHash;
    }

    public void setBlockHash(String blockHash) {
        this.blockHash = blockHash;
    }

    public String getPreviousHash() {
        return previousHash;
    }

    public void setPreviousHash(String previousHash) {
        this.previousHash = previousHash;
    }

    public String getMerkleRoot() {
        return merkleRoot;
    }

    public void setMerkleRoot(String merkleRoot) {
        this.merkleRoot = merkleRoot;
    }

    public Long getBlockNumber() {
        return blockNumber;
    }

    public void setBlockNumber(Long blockNumber) {
        this.blockNumber = blockNumber;
    }

    public Long getTxCount() {
        return txCount;
    }

    public void setTxCount(Long txCount) {
        this.txCount = txCount;
    }

    public Long getBlockSize() {
        return blockSize;
    }

    public void setBlockSize(Long blockSize) {
        this.blockSize = blockSize;
    }

    public Date getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(Date timestamp) {
        this.timestamp = timestamp;
    }

    public String getChannelId() {
        return channelId;
    }

    public void setChannelId(String channelId) {
        this.channelId = channelId;
    }

    public String getBlockCreator() {
        return blockCreator;
    }

    public void setBlockCreator(String blockCreator) {
        this.blockCreator = blockCreator;
    }

    public List<String> getTxIds() {
        return txIds;
    }

    public void setTxIds(List<String> txIds) {
        this.txIds = txIds;
    }

    // ==================== Builder Pattern ====================

    public static BlockInfoBuilder builder() {
        return new BlockInfoBuilder();
    }

    public static class BlockInfoBuilder {
        private String blockHash;
        private String previousHash;
        private String merkleRoot;
        private Long blockNumber;
        private Long txCount;
        private Long blockSize;
        private Date timestamp;
        private String channelId;
        private String blockCreator;
        private List<String> txIds;

        BlockInfoBuilder() {
        }

        public BlockInfoBuilder blockHash(String blockHash) {
            this.blockHash = blockHash;
            return this;
        }

        public BlockInfoBuilder previousHash(String previousHash) {
            this.previousHash = previousHash;
            return this;
        }

        public BlockInfoBuilder merkleRoot(String merkleRoot) {
            this.merkleRoot = merkleRoot;
            return this;
        }

        public BlockInfoBuilder blockNumber(Long blockNumber) {
            this.blockNumber = blockNumber;
            return this;
        }

        public BlockInfoBuilder txCount(Long txCount) {
            this.txCount = txCount;
            return this;
        }

        public BlockInfoBuilder blockSize(Long blockSize) {
            this.blockSize = blockSize;
            return this;
        }

        public BlockInfoBuilder timestamp(Date timestamp) {
            this.timestamp = timestamp;
            return this;
        }

        public BlockInfoBuilder channelId(String channelId) {
            this.channelId = channelId;
            return this;
        }

        public BlockInfoBuilder blockCreator(String blockCreator) {
            this.blockCreator = blockCreator;
            return this;
        }

        public BlockInfoBuilder txIds(List<String> txIds) {
            this.txIds = txIds;
            return this;
        }

        public BlockInfo build() {
            return new BlockInfo(blockHash, previousHash, merkleRoot, blockNumber, txCount, blockSize, timestamp, channelId, blockCreator, txIds);
        }

        public String toString() {
            return "BlockInfo.BlockInfoBuilder(blockHash=" + this.blockHash + ", previousHash=" + this.previousHash + ", merkleRoot=" + this.merkleRoot + ", blockNumber=" + this.blockNumber + ", txCount=" + this.txCount + ", blockSize=" + this.blockSize + ", timestamp=" + this.timestamp + ", channelId=" + this.channelId + ", blockCreator=" + this.blockCreator + ", txIds=" + this.txIds + ")";
        }
    }

    // ==================== equals, hashCode, toString ====================

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        BlockInfo blockInfo = (BlockInfo) o;
        return Objects.equals(blockHash, blockInfo.blockHash) &&
                Objects.equals(previousHash, blockInfo.previousHash) &&
                Objects.equals(merkleRoot, blockInfo.merkleRoot) &&
                Objects.equals(blockNumber, blockInfo.blockNumber) &&
                Objects.equals(txCount, blockInfo.txCount) &&
                Objects.equals(blockSize, blockInfo.blockSize) &&
                Objects.equals(timestamp, blockInfo.timestamp) &&
                Objects.equals(channelId, blockInfo.channelId) &&
                Objects.equals(blockCreator, blockInfo.blockCreator) &&
                Objects.equals(txIds, blockInfo.txIds);
    }

    @Override
    public int hashCode() {
        return Objects.hash(blockHash, previousHash, merkleRoot, blockNumber, txCount, blockSize, timestamp, channelId, blockCreator, txIds);
    }

    @Override
    public String toString() {
        return "BlockInfo{" +
                "blockHash='" + blockHash + '\'' +
                ", previousHash='" + previousHash + '\'' +
                ", merkleRoot='" + merkleRoot + '\'' +
                ", blockNumber=" + blockNumber +
                ", txCount=" + txCount +
                ", blockSize=" + blockSize +
                ", timestamp=" + timestamp +
                ", channelId='" + channelId + '\'' +
                ", blockCreator='" + blockCreator + '\'' +
                ", txIds=" + txIds +
                '}';
    }
}