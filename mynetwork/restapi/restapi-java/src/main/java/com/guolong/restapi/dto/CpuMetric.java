package com.guolong.restapi.dto;

import java.time.LocalDateTime;
import java.util.Objects;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;

public class CpuMetric {
    
    // 1. Java字段改用标准小写开头
    // 2. 使用 @JsonProperty 指定输出为大写开头 (和Go保持一致)
    @JsonProperty("Timestamp") 
    @JsonFormat(pattern = "yyyy-MM-dd HH:mm:ss")
    private LocalDateTime timestamp;

    @JsonProperty("CPUUsage")
    private double cpuUsage;

    // ==================== 构造函数 ====================

    public CpuMetric() {
    }

    public CpuMetric(LocalDateTime timestamp, double cpuUsage) {
        this.timestamp = timestamp;
        this.cpuUsage = cpuUsage;
    }

    // ==================== Getter / Setter ====================

    public LocalDateTime getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(LocalDateTime timestamp) {
        this.timestamp = timestamp;
    }

    public double getCpuUsage() {
        return cpuUsage;
    }

    public void setCpuUsage(double cpuUsage) {
        this.cpuUsage = cpuUsage;
    }

    // ==================== equals / hashCode / toString ====================

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        CpuMetric cpuMetric = (CpuMetric) o;
        return Double.compare(cpuMetric.cpuUsage, cpuUsage) == 0 &&
                Objects.equals(timestamp, cpuMetric.timestamp);
    }

    @Override
    public int hashCode() {
        return Objects.hash(timestamp, cpuUsage);
    }

    @Override
    public String toString() {
        return "CpuMetric{" +
                "timestamp=" + timestamp +
                ", cpuUsage=" + cpuUsage +
                '}';
    }
}