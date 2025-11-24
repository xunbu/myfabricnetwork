package com.guolong.gateway.dto;

import java.util.Objects;

import com.fasterxml.jackson.annotation.JsonProperty;

public class QueryRichResult {

    @JsonProperty("key")
    private String key;

    @JsonProperty("value")
    private String value;

    // ==================== 构造函数 ====================

    public QueryRichResult() {
    }

    public QueryRichResult(String key, String value) {
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

    // ==================== Builder Pattern ====================

    public static QueryRichResultBuilder builder() {
        return new QueryRichResultBuilder();
    }

    public static class QueryRichResultBuilder {
        private String key;
        private String value;

        QueryRichResultBuilder() {
        }

        public QueryRichResultBuilder key(String key) {
            this.key = key;
            return this;
        }

        public QueryRichResultBuilder value(String value) {
            this.value = value;
            return this;
        }

        public QueryRichResult build() {
            return new QueryRichResult(key, value);
        }

        public String toString() {
            return "QueryRichResult.QueryRichResultBuilder(key=" + this.key + ", value=" + this.value + ")";
        }
    }

    // ==================== equals, hashCode, toString ====================

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        QueryRichResult that = (QueryRichResult) o;
        return Objects.equals(key, that.key) &&
                Objects.equals(value, that.value);
    }

    @Override
    public int hashCode() {
        return Objects.hash(key, value);
    }

    @Override
    public String toString() {
        return "QueryRichResult{" +
                "key='" + key + '\'' +
                ", value='" + value + '\'' +
                '}';
    }
}