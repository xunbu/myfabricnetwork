package com.guolong.gateway.dto;

import java.util.Date;
import java.util.List;
import java.util.Objects;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;

public class TxInfo {

    @JsonProperty("txId")
    private String txId;

    @JsonProperty("channelId")
    private String channelId;

    @JsonProperty("type")
    private String type;

    @JsonProperty("timestamp")
    @JsonFormat(shape = JsonFormat.Shape.STRING, pattern = "yyyy-MM-dd'T'HH:mm:ss.SSSX", timezone = "UTC")
    private Date timestamp;

    @JsonProperty("size")
    private Integer size;

    @JsonProperty("validationCode")
    private Integer validationCode;

    @JsonProperty("chainCodeInfos")
    private List<ChainCodeInfo> chainCodeInfos;

    // ==================== 构造函数 ====================

    public TxInfo() {
    }

    public TxInfo(String txId, String channelId, String type, Date timestamp, Integer size, Integer validationCode, List<ChainCodeInfo> chainCodeInfos) {
        this.txId = txId;
        this.channelId = channelId;
        this.type = type;
        this.timestamp = timestamp;
        this.size = size;
        this.validationCode = validationCode;
        this.chainCodeInfos = chainCodeInfos;
    }

    // ==================== Getter / Setter ====================

    public String getTxId() {
        return txId;
    }

    public void setTxId(String txId) {
        this.txId = txId;
    }

    public String getChannelId() {
        return channelId;
    }

    public void setChannelId(String channelId) {
        this.channelId = channelId;
    }

    public String getType() {
        return type;
    }

    public void setType(String type) {
        this.type = type;
    }

    public Date getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(Date timestamp) {
        this.timestamp = timestamp;
    }

    public Integer getSize() {
        return size;
    }

    public void setSize(Integer size) {
        this.size = size;
    }

    public Integer getValidationCode() {
        return validationCode;
    }

    public void setValidationCode(Integer validationCode) {
        this.validationCode = validationCode;
    }

    public List<ChainCodeInfo> getChainCodeInfos() {
        return chainCodeInfos;
    }

    public void setChainCodeInfos(List<ChainCodeInfo> chainCodeInfos) {
        this.chainCodeInfos = chainCodeInfos;
    }

    // ==================== Builder Pattern ====================

    public static TxInfoBuilder builder() {
        return new TxInfoBuilder();
    }

    public static class TxInfoBuilder {
        private String txId;
        private String channelId;
        private String type;
        private Date timestamp;
        private Integer size;
        private Integer validationCode;
        private List<ChainCodeInfo> chainCodeInfos;

        TxInfoBuilder() {
        }

        public TxInfoBuilder txId(String txId) {
            this.txId = txId;
            return this;
        }

        public TxInfoBuilder channelId(String channelId) {
            this.channelId = channelId;
            return this;
        }

        public TxInfoBuilder type(String type) {
            this.type = type;
            return this;
        }

        public TxInfoBuilder timestamp(Date timestamp) {
            this.timestamp = timestamp;
            return this;
        }

        public TxInfoBuilder size(Integer size) {
            this.size = size;
            return this;
        }

        public TxInfoBuilder validationCode(Integer validationCode) {
            this.validationCode = validationCode;
            return this;
        }

        public TxInfoBuilder chainCodeInfos(List<ChainCodeInfo> chainCodeInfos) {
            this.chainCodeInfos = chainCodeInfos;
            return this;
        }

        public TxInfo build() {
            return new TxInfo(txId, channelId, type, timestamp, size, validationCode, chainCodeInfos);
        }

        public String toString() {
            return "TxInfo.TxInfoBuilder(txId=" + this.txId + ", channelId=" + this.channelId + ", type=" + this.type + ", timestamp=" + this.timestamp + ", size=" + this.size + ", validationCode=" + this.validationCode + ", chainCodeInfos=" + this.chainCodeInfos + ")";
        }
    }

    // ==================== equals, hashCode, toString ====================

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        TxInfo txInfo = (TxInfo) o;
        return Objects.equals(txId, txInfo.txId) &&
                Objects.equals(channelId, txInfo.channelId) &&
                Objects.equals(type, txInfo.type) &&
                Objects.equals(timestamp, txInfo.timestamp) &&
                Objects.equals(size, txInfo.size) &&
                Objects.equals(validationCode, txInfo.validationCode) &&
                Objects.equals(chainCodeInfos, txInfo.chainCodeInfos);
    }

    @Override
    public int hashCode() {
        return Objects.hash(txId, channelId, type, timestamp, size, validationCode, chainCodeInfos);
    }

    @Override
    public String toString() {
        return "TxInfo{" +
                "txId='" + txId + '\'' +
                ", channelId='" + channelId + '\'' +
                ", type='" + type + '\'' +
                ", timestamp=" + timestamp +
                ", size=" + size +
                ", validationCode=" + validationCode +
                ", chainCodeInfos=" + chainCodeInfos +
                '}';
    }
}