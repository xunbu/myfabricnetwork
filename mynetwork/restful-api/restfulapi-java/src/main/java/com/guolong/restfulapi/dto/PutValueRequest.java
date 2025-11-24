package com.guolong.restfulapi.dto;

import lombok.Data;

@Data
public class PutValueRequest {
    private String key;
    private String value;
}