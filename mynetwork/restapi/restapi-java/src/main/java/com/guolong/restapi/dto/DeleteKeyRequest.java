package com.guolong.restapi.dto;

import java.util.Objects;

public class DeleteKeyRequest {
    private String key;

    // ==================== 构造函数 ====================

    public DeleteKeyRequest() {
    }

    public DeleteKeyRequest(String key) {
        this.key = key;
    }

    // ==================== Getter / Setter ====================

    public String getKey() {
        return key;
    }

    public void setKey(String key) {
        this.key = key;
    }

    // ==================== equals / hashCode / toString ====================

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        DeleteKeyRequest that = (DeleteKeyRequest) o;
        return Objects.equals(key, that.key);
    }

    @Override
    public int hashCode() {
        return Objects.hash(key);
    }

    @Override
    public String toString() {
        return "DeleteKeyRequest{" +
                "key='" + key + '\'' +
                '}';
    }
}