package com.guolong.gateway.dto;

import java.util.List;
import java.util.Objects;

import com.fasterxml.jackson.annotation.JsonProperty;

public class RichPageResult {

    @JsonProperty("results")
    private List<QueryRichResult> results;

    @JsonProperty("hasMore")
    private boolean hasMore;

    @JsonProperty("page")
    private int page;

    @JsonProperty("total")
    private int total;

    // ==================== 构造函数 ====================

    public RichPageResult() {
    }

    public RichPageResult(List<QueryRichResult> results, boolean hasMore, int page, int total) {
        this.results = results;
        this.hasMore = hasMore;
        this.page = page;
        this.total = total;
    }

    // ==================== Getter / Setter ====================

    public List<QueryRichResult> getResults() {
        return results;
    }

    public void setResults(List<QueryRichResult> results) {
        this.results = results;
    }

    public boolean isHasMore() {
        return hasMore;
    }

    public void setHasMore(boolean hasMore) {
        this.hasMore = hasMore;
    }

    public int getPage() {
        return page;
    }

    public void setPage(int page) {
        this.page = page;
    }

    public int getTotal() {
        return total;
    }

    public void setTotal(int total) {
        this.total = total;
    }

    // ==================== Builder Pattern ====================

    public static RichPageResultBuilder builder() {
        return new RichPageResultBuilder();
    }

    public static class RichPageResultBuilder {
        private List<QueryRichResult> results;
        private boolean hasMore;
        private int page;
        private int total;

        RichPageResultBuilder() {
        }

        public RichPageResultBuilder results(List<QueryRichResult> results) {
            this.results = results;
            return this;
        }

        public RichPageResultBuilder hasMore(boolean hasMore) {
            this.hasMore = hasMore;
            return this;
        }

        public RichPageResultBuilder page(int page) {
            this.page = page;
            return this;
        }

        public RichPageResultBuilder total(int total) {
            this.total = total;
            return this;
        }

        public RichPageResult build() {
            return new RichPageResult(results, hasMore, page, total);
        }

        public String toString() {
            return "RichPageResult.RichPageResultBuilder(results=" + this.results + ", hasMore=" + this.hasMore + ", page=" + this.page + ", total=" + this.total + ")";
        }
    }

    // ==================== equals, hashCode, toString ====================

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;
        RichPageResult that = (RichPageResult) o;
        return hasMore == that.hasMore &&
                page == that.page &&
                total == that.total &&
                Objects.equals(results, that.results);
    }

    @Override
    public int hashCode() {
        return Objects.hash(results, hasMore, page, total);
    }

    @Override
    public String toString() {
        return "RichPageResult{" +
                "results=" + results +
                ", hasMore=" + hasMore +
                ", page=" + page +
                ", total=" + total +
                '}';
    }
}