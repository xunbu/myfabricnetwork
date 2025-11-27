package com.guolong.restapi.service;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.LinkedList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

import javax.annotation.PostConstruct;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import com.github.dockerjava.api.DockerClient;
import com.github.dockerjava.api.model.Container;
import com.github.dockerjava.api.model.Statistics;
import com.github.dockerjava.core.InvocationBuilder;
import com.guolong.restapi.dto.CpuMetric;
import com.guolong.restapi.dto.MemoryMetric;

@Service
@ConfigurationProperties(prefix = "docker") // 1. 绑定配置前缀
public class DockerMonitorService {

    private static final Logger log = LoggerFactory.getLogger(DockerMonitorService.class);

    private final DockerClient dockerClient;

    public DockerMonitorService(DockerClient dockerClient) {
        this.dockerClient = dockerClient;
    }

    // 2. 定义容器列表，对应 YAML 中的 docker.containers
    private List<String> containers = new ArrayList<>();

    // 3. 必须提供 Setter 方法，Spring Boot 才能注入 YAML 列表
    public void setContainers(List<String> containers) {
        this.containers = containers;
    }

    public List<String> getContainers() {
        return containers;
    }

    private static final int MAX_HISTORY_SIZE = 1000;

    // 内部存储依然使用 ConcurrentHashMap 保证并发读写安全
    private final Map<String, LinkedList<CpuMetric>> cpuHistory = new ConcurrentHashMap<>();
    private final Map<String, LinkedList<MemoryMetric>> memoryHistory = new ConcurrentHashMap<>();
    private final Map<String, String> containerIdMap = new ConcurrentHashMap<>();
    private final Map<String, Statistics> preStatsMap = new ConcurrentHashMap<>();

    @PostConstruct
    public void init() {
        log.info("初始化 Docker 监控列表 (顺序已保留): {}", containers);
        refreshContainerIds();
        
        // 根据配置列表初始化 Map
        for (String name : containers) {
            cpuHistory.put(name, new LinkedList<>());
            memoryHistory.put(name, new LinkedList<>());
        }
    }

    private void refreshContainerIds() {
        try {
            List<Container> containerList = dockerClient.listContainersCmd().withShowAll(true).exec();
            
            for (String targetName : containers) {
                boolean found = false;
                for (Container c : containerList) {
                    if (c.getNames() == null) continue;
                    for (String cName : c.getNames()) {
                        // 匹配名称
                        if (cName.endsWith(targetName) || cName.equals("/" + targetName)) {
                            containerIdMap.put(targetName, c.getId());
                            found = true;
                            break;
                        }
                    }
                    if (found) break;
                }
            }
        } catch (Exception e) {
            log.error("刷新容器列表失败: {}", e.getMessage());
        }
    }

    @Scheduled(fixedRate = 2000)
    public void collectStats() {
        if (containerIdMap.isEmpty()) {
            refreshContainerIds();
        }

        containerIdMap.forEach((name, id) -> {
            try {
                InvocationBuilder.AsyncResultCallback<Statistics> callback = new InvocationBuilder.AsyncResultCallback<>();
                dockerClient.statsCmd(id).withNoStream(true).exec(callback);
                Statistics stats = callback.awaitResult();
                if (stats != null) {
                    processStats(name, stats);
                }
            } catch (Exception e) {
                // 容器可能停止或被删除
                // log.warn("无法获取容器 [{}] 统计信息", name);
                containerIdMap.remove(name);
            }
        });
    }

    private void processStats(String name, Statistics stats) {
        LocalDateTime now = LocalDateTime.now();

        // 处理内存
        if (stats.getMemoryStats() != null) {
            Long used = stats.getMemoryStats().getUsage();
            Long limit = stats.getMemoryStats().getLimit();
            if (used != null && limit != null) {
                addToHistory(memoryHistory.get(name), new MemoryMetric(now, used, limit));
            }
        }

        // 处理 CPU
        Statistics pre = preStatsMap.get(name);
        if (pre != null) {
            double percent = calculateCpu(stats, pre);
            addToHistory(cpuHistory.get(name), new CpuMetric(now, percent));
        }
        preStatsMap.put(name, stats);
    }

    private double calculateCpu(Statistics cur, Statistics pre) {
        // 空指针保护
        if (cur.getCpuStats() == null || pre.getCpuStats() == null) return 0.0;
        if (cur.getCpuStats().getCpuUsage() == null || pre.getCpuStats().getCpuUsage() == null) return 0.0;

        Long cpuTotal = cur.getCpuStats().getCpuUsage().getTotalUsage();
        Long preCpuTotal = pre.getCpuStats().getCpuUsage().getTotalUsage();
        Long sysTotal = cur.getCpuStats().getSystemCpuUsage();
        Long preSysTotal = pre.getCpuStats().getSystemCpuUsage();

        if (cpuTotal != null && preCpuTotal != null && sysTotal != null && preSysTotal != null) {
            long cpuDelta = cpuTotal - preCpuTotal;
            long sysDelta = sysTotal - preSysTotal;

            if (sysDelta > 0 && cpuDelta > 0) {
                long onlineCpus = 1;
                if (cur.getCpuStats().getOnlineCpus() != null) {
                    onlineCpus = cur.getCpuStats().getOnlineCpus();
                }
                return ((double) cpuDelta / sysDelta) * onlineCpus * 100.0;
            }
        }
        return 0.0;
    }

    private <T> void addToHistory(LinkedList<T> list, T item) {
        if (list == null) return;
        synchronized (list) {
            list.add(item);
            if (list.size() > MAX_HISTORY_SIZE) list.removeFirst();
        }
    }

    /**
     * 获取所有容器的 CPU 历史数据
     * 关键修改：使用 LinkedHashMap 并按照 containers 配置的顺序填充，确保前端显示顺序一致
     */
    public Map<String, List<CpuMetric>> getAllCpuHistory() {
        Map<String, List<CpuMetric>> sortedMap = new LinkedHashMap<>();
        for (String name : containers) {
            List<CpuMetric> history = cpuHistory.get(name);
            if (history != null) {
                sortedMap.put(name, history);
            }
        }
        return sortedMap;
    }

    /**
     * 获取所有容器的 内存 历史数据
     * 关键修改：使用 LinkedHashMap 并按照 containers 配置的顺序填充
     */
    public Map<String, List<MemoryMetric>> getAllMemoryHistory() {
        Map<String, List<MemoryMetric>> sortedMap = new LinkedHashMap<>();
        for (String name : containers) {
            List<MemoryMetric> history = memoryHistory.get(name);
            if (history != null) {
                sortedMap.put(name, history);
            }
        }
        return sortedMap;
    }
}