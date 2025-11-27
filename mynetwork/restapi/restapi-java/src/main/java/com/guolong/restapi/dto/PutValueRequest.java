package com.guolong.restapi.dto;

import java.util.Objects;

public class PutValueRequest {
    private String key;
    private String value;

    // ==================== 构造函数 ====================

    public PutValueRequest() {
    }

    public PutValueRequest(String key, String value) {
        this.key = key;
        this.value = value;
    }

    // ==================== Getter / Setter ====================

    public String getKey() {
        return key;
    }

    public void setKey(String key) {
        this.key = key;
    }

    public String getValue() {
        return value;
    }

    public void setValue(String value) {
        this.value = value;
    }

    // ==================== equals / hashCode / toString ====================

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        PutValueRequest that = (PutValueRequest) o;
        return Objects.equals(key, that.key) &&
                Objects.equals(value, that.value);
    }

    @Override
    public int hashCode() {
        return Objects.hash(key, value);
    }

    @Override
    public String toString() {
        return "PutValueRequest{" +
                "key='" + key + '\'' +
                ", value='" + value + '\'' +
                '}';
    }
}