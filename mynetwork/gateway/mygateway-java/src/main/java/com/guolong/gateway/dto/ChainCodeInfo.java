package com.guolong.gateway.dto;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ChainCodeInfo {

    @JsonProperty("chainCodeName")
    private String chainCodeName;

    @JsonProperty("args")
    private List<String> args;
}