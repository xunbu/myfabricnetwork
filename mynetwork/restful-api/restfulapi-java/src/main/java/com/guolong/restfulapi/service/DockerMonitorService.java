package com.guolong.restfulapi.service;

import com.github.dockerjava.api.DockerClient;
import com.github.dockerjava.api.model.Container;
import com.github.dockerjava.api.model.Statistics;
import com.github.dockerjava.core.InvocationBuilder;
import com.guolong.restfulapi.dto.CpuMetric;
import com.guolong.restfulapi.dto.MemoryMetric;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import javax.annotation.PostConstruct;
import java.time.LocalDateTime;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

@Service
@Slf4j
@RequiredArgsConstructor
public class DockerMonitorService {

    private final DockerClient dockerClient;

    @Value("#{'${docker.containers}'.split(',')}")
    private List<String> containerNames;

    private static final int MAX_HISTORY_SIZE = 1000;

    private final Map<String, LinkedList<CpuMetric>> cpuHistory = new ConcurrentHashMap<>();
    private final Map<String, LinkedList<MemoryMetric>> memoryHistory = new ConcurrentHashMap<>();
    private final Map<String, String> containerIdMap = new ConcurrentHashMap<>();
    private final Map<String, Statistics> preStatsMap = new ConcurrentHashMap<>();

    @PostConstruct
    public void init() {
        log.info("初始化 Docker 监控: {}", containerNames);
        refreshContainerIds();
        for (String name : containerNames) {
            cpuHistory.put(name, new LinkedList<>());
            memoryHistory.put(name, new LinkedList<>());
        }
    }

    private void refreshContainerIds() {
        try {
            List<Container> containers = dockerClient.listContainersCmd().withShowAll(true).exec();
            for (String targetName : containerNames) {
                for (Container c : containers) {
                    for (String cName : c.getNames()) {
                        if (cName.endsWith(targetName) || cName.equals("/" + targetName)) {
                            containerIdMap.put(targetName, c.getId());
                            break;
                        }
                    }
                }
            }
        } catch (Exception e) {
            log.error("刷新容器列表失败: {}", e.getMessage());
        }
    }

    @Scheduled(fixedRate = 2000)
    public void collectStats() {
        if (containerIdMap.isEmpty()) refreshContainerIds();

        containerIdMap.forEach((name, id) -> {
            try {
                InvocationBuilder.AsyncResultCallback<Statistics> callback = new InvocationBuilder.AsyncResultCallback<>();
                dockerClient.statsCmd(id).withNoStream(true).exec(callback);
                Statistics stats = callback.awaitResult();
                if (stats != null) {
                    processStats(name, stats);
                }
            } catch (Exception e) {
                // 容器可能停止了
                containerIdMap.remove(name);
            }
        });
    }

    private void processStats(String name, Statistics stats) {
        LocalDateTime now = LocalDateTime.now();

        // Memory
        Long used = stats.getMemoryStats().getUsage();
        Long limit = stats.getMemoryStats().getLimit();
        if (used != null && limit != null) {
            addToHistory(memoryHistory.get(name), new MemoryMetric(now, used, limit));
        }

        // CPU
        Statistics pre = preStatsMap.get(name);
        if (pre != null) {
            double percent = calculateCpu(stats, pre);
            addToHistory(cpuHistory.get(name), new CpuMetric(now, percent));
        }
        preStatsMap.put(name, stats);
    }

    private double calculateCpu(Statistics cur, Statistics pre) {
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

    public Map<String, List<CpuMetric>> getAllCpuHistory() {
        return new HashMap<>(cpuHistory);
    }

    public Map<String, List<MemoryMetric>> getAllMemoryHistory() {
        return new HashMap<>(memoryHistory);
    }
}