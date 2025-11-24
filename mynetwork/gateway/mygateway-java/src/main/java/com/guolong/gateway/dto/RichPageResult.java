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
public class RichPageResult {

    @JsonProperty("results")
    private List<QueryRichResult> results;

    @JsonProperty("hasMore")
    private boolean hasMore;

    @JsonProperty("page")
    private int page;

    @JsonProperty("total")
    private int total;
}