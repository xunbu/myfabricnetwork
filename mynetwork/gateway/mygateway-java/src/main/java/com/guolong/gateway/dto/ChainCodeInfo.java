package com.guolong.gateway.dto;

import java.util.List;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ChainCodeInfo {

    // [修复] Go struct 默认为大写开头
    @JsonProperty("ChainCodeName")
    private String chainCodeName;

    @JsonProperty("Args")
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
}