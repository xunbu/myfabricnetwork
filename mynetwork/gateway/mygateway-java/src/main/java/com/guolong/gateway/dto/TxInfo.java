package com.guolong.gateway.dto;

import java.util.Date;
import java.util.List;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;

public class TxInfo {

    // [修复] Go struct 默认为大写开头，前端可能依赖此格式
    @JsonProperty("TxId")
    private String txId;

    @JsonProperty("ChannelId")
    private String channelId;

    @JsonProperty("Type")
    private String type;

    @JsonProperty("Timestamp")
    @JsonFormat(shape = JsonFormat.Shape.STRING, pattern = "yyyy-MM-dd'T'HH:mm:ss.SSSX", timezone = "UTC")
    private Date timestamp;

    @JsonProperty("Size")
    private Integer size;

    @JsonProperty("ValidationCode")
    private Integer validationCode;

    @JsonProperty("ChainCodeInfos")
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
    
    // toString, equals, hashCode 可以保持标准生成，不需要改
    @Override
    public String toString() {
        return "TxInfo{" + "txId='" + txId + '\'' + ", channelId='" + channelId + '\'' + '}';
    }
}