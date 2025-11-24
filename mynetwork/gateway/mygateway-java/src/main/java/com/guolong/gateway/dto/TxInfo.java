package com.guolong.gateway.dto;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.Date;
import java.util.List;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
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
}