package com.guolong.gateway.dto;

import java.util.Date;
import java.util.Objects;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;

public class KeyHistory {

    @JsonProperty("txId")
    private String txId;

    @JsonProperty("timestamp")
    @JsonFormat(shape = JsonFormat.Shape.STRING, pattern = "yyyy-MM-dd'T'HH:mm:ss.SSSX", timezone = "UTC")
    private Date timestamp;

    @JsonProperty("isDelete")
    private Boolean isDelete;

    @JsonProperty("value")
    private String value;

    // ==================== 构造函数 ====================

    public KeyHistory() {
    }

    public KeyHistory(String txId, Date timestamp, Boolean isDelete, String value) {
        this.txId = txId;
        this.timestamp = timestamp;
        this.isDelete = isDelete;
        this.value = value;
    }

    // ==================== Getter / Setter ====================

    public String getTxId() {
        return txId;
    }

    public void setTxId(String txId) {
        this.txId = txId;
    }

    public Date getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(Date timestamp) {
        this.timestamp = timestamp;
    }

    public Boolean getIsDelete() {
        return isDelete;
    }

    public void setIsDelete(Boolean isDelete) {
        this.isDelete = isDelete;
    }

    public String getValue() {
        return value;
    }

    public void setValue(String value) {
        this.value = value;
    }

    // ==================== Builder Pattern ====================

    public static KeyHistoryBuilder builder() {
        return new KeyHistoryBuilder();
    }

    public static class KeyHistoryBuilder {
        private String txId;
        private Date timestamp;
        private Boolean isDelete;
        private String value;

        KeyHistoryBuilder() {
        }

        public KeyHistoryBuilder txId(String txId) {
            this.txId = txId;
            return this;
        }

        public KeyHistoryBuilder timestamp(Date timestamp) {
            this.timestamp = timestamp;
            return this;
        }

        public KeyHistoryBuilder isDelete(Boolean isDelete) {
            this.isDelete = isDelete;
            return this;
        }

        public KeyHistoryBuilder value(String value) {
            this.value = value;
            return this;
        }

        public KeyHistory build() {
            return new KeyHistory(txId, timestamp, isDelete, value);
        }

        public String toString() {
            return "KeyHistory.KeyHistoryBuilder(txId=" + this.txId + ", timestamp=" + this.timestamp + ", isDelete=" + this.isDelete + ", value=" + this.value + ")";
        }
    }

    // ==================== equals, hashCode, toString ====================

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        KeyHistory that = (KeyHistory) o;
        return Objects.equals(txId, that.txId) &&
                Objects.equals(timestamp, that.timestamp) &&
                Objects.equals(isDelete, that.isDelete) &&
                Objects.equals(value, that.value);
    }

    @Override
    public int hashCode() {
        return Objects.hash(txId, timestamp, isDelete, value);
    }

    @Override
    public String toString() {
        return "KeyHistory{" +
                "txId='" + txId + '\'' +
                ", timestamp=" + timestamp +
                ", isDelete=" + isDelete +
                ", value='" + value + '\'' +
                '}';
    }
}