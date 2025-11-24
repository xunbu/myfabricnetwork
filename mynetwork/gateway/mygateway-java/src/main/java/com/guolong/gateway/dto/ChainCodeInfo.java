package com.guolong.gateway.dto;

import java.util.List;
import java.util.Objects;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ChainCodeInfo {

    @JsonProperty("chainCodeName")
    private String chainCodeName;

    @JsonProperty("args")
    private List<String> args;

    // ==================== 构造函数 ====================

    public ChainCodeInfo() {
    }

    public ChainCodeInfo(String chainCodeName, List<String> args) {
        this.chainCodeName = chainCodeName;
        this.args = args;
    }

    // ==================== Getter / Setter ====================

    public String getChainCodeName() {
        return chainCodeName;
    }

    public void setChainCodeName(String chainCodeName) {
        this.chainCodeName = chainCodeName;
    }

    public List<String> getArgs() {
        return args;
    }

    public void setArgs(List<String> args) {
        this.args = args;
    }

    // ==================== Builder Pattern ====================

    public static ChainCodeInfoBuilder builder() {
        return new ChainCodeInfoBuilder();
    }

    public static class ChainCodeInfoBuilder {
        private String chainCodeName;
        private List<String> args;

        ChainCodeInfoBuilder() {
        }

        public ChainCodeInfoBuilder chainCodeName(String chainCodeName) {
            this.chainCodeName = chainCodeName;
            return this;
        }

        public ChainCodeInfoBuilder args(List<String> args) {
            this.args = args;
            return this;
        }

        public ChainCodeInfo build() {
            return new ChainCodeInfo(chainCodeName, args);
        }

        public String toString() {
            return "ChainCodeInfo.ChainCodeInfoBuilder(chainCodeName=" + this.chainCodeName + ", args=" + this.args + ")";
        }
    }

    // ==================== equals, hashCode, toString ====================

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        ChainCodeInfo that = (ChainCodeInfo) o;
        return Objects.equals(chainCodeName, that.chainCodeName) &&
                Objects.equals(args, that.args);
    }

    @Override
    public int hashCode() {
        return Objects.hash(chainCodeName, args);
    }

    @Override
    public String toString() {
        return "ChainCodeInfo{" +
                "chainCodeName='" + chainCodeName + '\'' +
                ", args=" + args +
                '}';
    }
}