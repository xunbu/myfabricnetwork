package com.guolong.restapi.dto;

import java.time.LocalDateTime;
import java.util.Objects;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;

public class MemoryMetric {

    @JsonProperty("Timestamp")
    @JsonFormat(pattern = "yyyy-MM-dd HH:mm:ss")
    private LocalDateTime timestamp;

    @JsonProperty("UsedMemory")
    private long usedMemory;

    @JsonProperty("TotalMemory")
    private long totalMemory;

    // ==================== 构造函数 ====================

    public MemoryMetric() {
    }

    public MemoryMetric(LocalDateTime timestamp, long usedMemory, long totalMemory) {
        this.timestamp = timestamp;
        this.usedMemory = usedMemory;
        this.totalMemory = totalMemory;
    }

    // ==================== Getter / Setter ====================

    public LocalDateTime getTimestamp() {
        return timestamp;
    }

    public void setTimestamp(LocalDateTime timestamp) {
        this.timestamp = timestamp;
    }

    public long getUsedMemory() {
        return usedMemory;
    }

    public void setUsedMemory(long usedMemory) {
        this.usedMemory = usedMemory;
    }

    public long getTotalMemory() {
        return totalMemory;
    }

    public void setTotalMemory(long totalMemory) {
        this.totalMemory = totalMemory;
    }

    // ==================== equals / hashCode / toString ====================

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        MemoryMetric that = (MemoryMetric) o;
        return usedMemory == that.usedMemory &&
                totalMemory == that.totalMemory &&
                Objects.equals(timestamp, that.timestamp);
    }

    @Override
    public int hashCode() {
        return Objects.hash(timestamp, usedMemory, totalMemory);
    }

    @Override
    public String toString() {
        return "MemoryMetric{" +
                "timestamp=" + timestamp +
                ", usedMemory=" + usedMemory +
                ", totalMemory=" + totalMemory +
                '}';
    }
}