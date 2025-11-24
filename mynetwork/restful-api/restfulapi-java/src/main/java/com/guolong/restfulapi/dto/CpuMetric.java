package com.guolong.restfulapi.dto;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Data;
import java.time.LocalDateTime;

@Data
@AllArgsConstructor
public class CpuMetric {
    
    // 1. Java字段改用标准小写开头
    // 2. 使用 @JsonProperty 指定输出为大写开头 (和Go保持一致)
    @JsonProperty("Timestamp") 
    @JsonFormat(pattern = "yyyy-MM-dd HH:mm:ss")
    private LocalDateTime timestamp;

    @JsonProperty("CPUUsage")
    private double cpuUsage;
}