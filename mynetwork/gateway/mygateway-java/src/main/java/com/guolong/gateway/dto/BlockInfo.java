package com.guolong.gateway.dto;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonInclude;
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
// 对应 Go 的 omitempty：如果字段为 null，不包含在 JSON 中
@JsonInclude(JsonInclude.Include.NON_NULL) 
public class BlockInfo {

    @JsonProperty("blockHash")
    private String blockHash;

    @JsonProperty("previousHash")
    private String previousHash;

    // Go代码中 Struct 字段是 MerkleRoot, 但 json tag 是 "dataHash"
    @JsonProperty("dataHash")
    private String merkleRoot; 

    @JsonProperty("blockNumber")
    private Long blockNumber; // 使用 Long 对应 uint64

    @JsonProperty("txCount")
    private Long txCount;

    @JsonProperty("blockSize")
    private Long blockSize;

    @JsonProperty("timestamp")
    @JsonFormat(shape = JsonFormat.Shape.STRING, pattern = "yyyy-MM-dd'T'HH:mm:ss.SSSX", timezone = "UTC")
    private Date timestamp;

    @JsonProperty("channelId")
    private String channelId;

    @JsonProperty("blockCreator")
    private String blockCreator;

    @JsonProperty("txIds")
    private List<String> txIds;
}