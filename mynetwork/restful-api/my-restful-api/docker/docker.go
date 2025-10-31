package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// CPUMetric 存储单个时间点的CPU使用率
type CPUMetric struct {
	Timestamp time.Time
	CPUUsage  float64
}

// MemoryMetric 存储单个时间点的内存使用情况
type MemoryMetric struct {
	Timestamp   time.Time
	UsedMemory  uint64 // 已使用内存(字节)
	TotalMemory uint64 // 总内存限制(字节)
}

// CPUHistory 存储容器的CPU历史记录
var (
	monitorCpuInterval    = 2 * time.Second
	monitorMemoryInterval = 2 * time.Second
	cpuHistory            = make(map[string][]CPUMetric)
	memoryHistory         = make(map[string][]MemoryMetric)
	cpuHistoryMutex       sync.RWMutex
	memoryHistoryMutex    sync.RWMutex
	monitorCtx            context.Context
	monitorCancel         context.CancelFunc
	monitorWg             sync.WaitGroup
)

// GetCpuHistoryByContainerNames 启动一个或多个goroutine来监控指定容器的CPU使用情况,
// 并将历史数据存储在cpuHistory中。
func GetCpuHistoryByContainerNames(containerNames []string) error {
	if len(containerNames) == 0 {
		return fmt.Errorf("容器名称列表不能为空")
	}

	// 创建Docker客户端
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("创建Docker客户端失败: %w", err)
	}
	defer cli.Close()

	// 创建监控上下文
	monitorCtx, monitorCancel = context.WithCancel(context.Background())

	// 验证容器是否存在并启动监控
	for _, containerName := range containerNames {
		containerID, err := getContainerIDByName(cli, containerName)
		if err != nil {
			return fmt.Errorf("获取容器 %s 失败: %w", containerName, err)
		}

		// 为每个容器启动一个监控goroutine
		monitorWg.Add(1)
		go monitorContainerCPU(containerName, containerID)
	}

	return nil
}

// GetMemoryHistoryByContainerNames 启动一个或多个goroutine来监控指定容器的内存使用情况,
// 并将历史数据存储在memoryHistory中。
func GetMemoryHistoryByContainerNames(containerNames []string) error {
	if len(containerNames) == 0 {
		return fmt.Errorf("容器名称列表不能为空")
	}

	// 创建Docker客户端
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("创建Docker客户端失败: %w", err)
	}
	defer cli.Close()

	// 如果监控上下文未创建，则创建
	if monitorCtx == nil {
		monitorCtx, monitorCancel = context.WithCancel(context.Background())
	}

	// 验证容器是否存在并启动监控
	for _, containerName := range containerNames {
		containerID, err := getContainerIDByName(cli, containerName)
		if err != nil {
			return fmt.Errorf("获取容器 %s 失败: %w", containerName, err)
		}

		// 为每个容器启动一个监控goroutine
		monitorWg.Add(1)
		go monitorContainerMemory(containerName, containerID)
	}

	return nil
}

// getContainerIDByName 根据容器名称获取容器ID
func getContainerIDByName(cli *client.Client, containerName string) (string, error) {
	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", err
	}

	for _, c := range containers {
		for _, name := range c.Names {
			// 容器名称以 / 开头，需要去除
			if strings.TrimPrefix(name, "/") == containerName {
				return c.ID, nil
			}
		}
	}

	return "", fmt.Errorf("未找到容器: %s", containerName)
}

// monitorContainerCPU 监控单个容器的CPU使用情况
func monitorContainerCPU(containerName, containerID string) {
	defer monitorWg.Done()

	// 创建Docker客户端
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("为容器 %s 创建Docker客户端失败: %v\n", containerName, err)
		return
	}
	defer cli.Close()

	ticker := time.NewTicker(monitorCpuInterval)
	defer ticker.Stop()

	for {
		select {
		case <-monitorCtx.Done():
			fmt.Printf("停止监控容器CPU: %s\n", containerName)
			return
		case <-ticker.C:
			cpuUsage, err := getContainerCPUUsage(cli, containerID)
			if err != nil {
				fmt.Printf("获取容器 %s CPU使用率失败: %v\n", containerName, err)
				continue
			}

			// 存储CPU指标
			metric := CPUMetric{
				Timestamp: time.Now(),
				CPUUsage:  cpuUsage,
			}

			cpuHistoryMutex.Lock()
			cpuHistory[containerName] = append(cpuHistory[containerName], metric)

			// 保留最近1000条记录，避免内存无限增长
			if len(cpuHistory[containerName]) > 1000 {
				cpuHistory[containerName] = cpuHistory[containerName][len(cpuHistory[containerName])-1000:]
			}
			cpuHistoryMutex.Unlock()

			// fmt.Printf("[%s] 容器: %s, CPU使用率: %.2f%%\n",
			// 	metric.Timestamp.Format("2006-01-02 15:04:05"),
			// 	containerName,
			// 	cpuUsage)
		}
	}
}

// monitorContainerMemory 监控单个容器的内存使用情况
func monitorContainerMemory(containerName, containerID string) {
	defer monitorWg.Done()

	// 创建Docker客户端
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Printf("为容器 %s 创建Docker客户端失败: %v\n", containerName, err)
		return
	}
	defer cli.Close()

	ticker := time.NewTicker(monitorMemoryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-monitorCtx.Done():
			fmt.Printf("停止监控容器内存: %s\n", containerName)
			return
		case <-ticker.C:
			usedMemory, totalMemory, err := getContainerMemoryUsage(cli, containerID)
			if err != nil {
				fmt.Printf("获取容器 %s 内存使用情况失败: %v\n", containerName, err)
				continue
			}

			// 存储内存指标
			metric := MemoryMetric{
				Timestamp:   time.Now(),
				UsedMemory:  usedMemory,
				TotalMemory: totalMemory,
			}

			memoryHistoryMutex.Lock()
			memoryHistory[containerName] = append(memoryHistory[containerName], metric)

			// 保留最近1000条记录，避免内存无限增长
			if len(memoryHistory[containerName]) > 1000 {
				memoryHistory[containerName] = memoryHistory[containerName][len(memoryHistory[containerName])-1000:]
			}
			memoryHistoryMutex.Unlock()

			// fmt.Printf("[%s] 容器: %s, 内存使用: %d/%d 字节 (%.2f%%)\n",
			// 	metric.Timestamp.Format("2006-01-02 15:04:05"),
			// 	containerName,
			// 	usedMemory,
			// 	totalMemory,
			// 	float64(usedMemory)/float64(totalMemory)*100)
		}
	}
}

// StatsJSON 定义统计数据结构（兼容 Docker SDK v28）
type StatsJSON struct {
	CPUStats    CPUStats    `json:"cpu_stats"`
	PreCPUStats CPUStats    `json:"precpu_stats"`
	MemoryStats MemoryStats `json:"memory_stats"`
}

// CPUStats CPU统计信息
type CPUStats struct {
	CPUUsage       CPUUsage `json:"cpu_usage"`
	SystemUsage    uint64   `json:"system_cpu_usage"`
	OnlineCPUs     uint32   `json:"online_cpus"`
	ThrottlingData struct {
		Periods          uint64 `json:"periods"`
		ThrottledPeriods uint64 `json:"throttled_periods"`
		ThrottledTime    uint64 `json:"throttled_time"`
	} `json:"throttling_data"`
}

// CPUUsage CPU使用情况
type CPUUsage struct {
	TotalUsage        uint64   `json:"total_usage"`
	PercpuUsage       []uint64 `json:"percpu_usage"`
	UsageInKernelmode uint64   `json:"usage_in_kernelmode"`
	UsageInUsermode   uint64   `json:"usage_in_usermode"`
}

// MemoryStats 内存统计信息
type MemoryStats struct {
	Usage    uint64 `json:"usage"`
	MaxUsage uint64 `json:"max_usage"`
	Limit    uint64 `json:"limit"`
}

// getContainerCPUUsage 计算容器的CPU使用率
func getContainerCPUUsage(cli *client.Client, containerID string) (float64, error) {
	ctx := context.Background()

	// 获取容器统计信息
	stats, err := cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return 0, err
	}
	defer stats.Body.Close()

	// 读取统计数据
	var v StatsJSON
	decoder := json.NewDecoder(stats.Body)
	if err := decoder.Decode(&v); err != nil {
		if err == io.EOF {
			return 0, fmt.Errorf("容器可能已停止")
		}
		return 0, err
	}

	// 计算CPU使用率
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)

	cpuUsage := 0.0
	if systemDelta > 0 && cpuDelta > 0 {
		// 使用 OnlineCPUs 或 PercpuUsage 的长度
		numCPUs := float64(v.CPUStats.OnlineCPUs)
		if numCPUs == 0 {
			numCPUs = float64(len(v.CPUStats.CPUUsage.PercpuUsage))
		}
		if numCPUs == 0 {
			numCPUs = 1 // 默认至少1个CPU
		}
		cpuUsage = (cpuDelta / systemDelta) * numCPUs * 100.0
	}

	return cpuUsage, nil
}

// getContainerMemoryUsage 获取容器的内存使用情况
func getContainerMemoryUsage(cli *client.Client, containerID string) (uint64, uint64, error) {
	ctx := context.Background()

	// 获取容器统计信息
	stats, err := cli.ContainerStats(ctx, containerID, false)
	if err != nil {
		return 0, 0, err
	}
	defer stats.Body.Close()

	// 读取统计数据
	var v StatsJSON
	decoder := json.NewDecoder(stats.Body)
	if err := decoder.Decode(&v); err != nil {
		if err == io.EOF {
			return 0, 0, fmt.Errorf("容器可能已停止")
		}
		return 0, 0, err
	}

	// 返回已使用内存和总内存限制
	usedMemory := v.MemoryStats.Usage
	totalMemory := v.MemoryStats.Limit

	return usedMemory, totalMemory, nil
}

// GetCPUHistory 获取指定容器的CPU历史记录
func GetCPUHistory(containerName string) []CPUMetric {
	cpuHistoryMutex.RLock()
	defer cpuHistoryMutex.RUnlock()

	history, exists := cpuHistory[containerName]
	if !exists {
		return []CPUMetric{}
	}

	// 返回副本，避免外部修改
	result := make([]CPUMetric, len(history))
	copy(result, history)
	return result
}

// GetMemoryHistory 获取指定容器的内存历史记录
func GetMemoryHistory(containerName string) []MemoryMetric {
	memoryHistoryMutex.RLock()
	defer memoryHistoryMutex.RUnlock()

	history, exists := memoryHistory[containerName]
	if !exists {
		return []MemoryMetric{}
	}

	// 返回副本，避免外部修改
	result := make([]MemoryMetric, len(history))
	copy(result, history)
	return result
}

// GetAllCPUHistory 获取所有容器的CPU历史记录
func GetAllCPUHistory() map[string][]CPUMetric {
	cpuHistoryMutex.RLock()
	defer cpuHistoryMutex.RUnlock()

	result := make(map[string][]CPUMetric)
	for name, history := range cpuHistory {
		historyClone := make([]CPUMetric, len(history))
		copy(historyClone, history)
		result[name] = historyClone
	}
	return result
}

// GetAllMemoryHistory 获取所有容器的内存历史记录
func GetAllMemoryHistory() map[string][]MemoryMetric {
	memoryHistoryMutex.RLock()
	defer memoryHistoryMutex.RUnlock()

	result := make(map[string][]MemoryMetric)
	for name, history := range memoryHistory {
		historyClone := make([]MemoryMetric, len(history))
		copy(historyClone, history)
		result[name] = historyClone
	}
	return result
}

// StopMonitoring 停止所有容器的监控
func StopMonitoring() {
	if monitorCancel != nil {
		monitorCancel()
		monitorWg.Wait()
		fmt.Println("所有容器监控已停止")
	}
}

// ClearHistory 清除指定容器的历史记录
func ClearHistory(containerName string) {
	cpuHistoryMutex.Lock()
	defer cpuHistoryMutex.Unlock()
	delete(cpuHistory, containerName)

	memoryHistoryMutex.Lock()
	defer memoryHistoryMutex.Unlock()
	delete(memoryHistory, containerName)
}

// ClearAllHistory 清除所有容器的历史记录
func ClearAllHistory() {
	cpuHistoryMutex.Lock()
	defer cpuHistoryMutex.Unlock()
	cpuHistory = make(map[string][]CPUMetric)

	memoryHistoryMutex.Lock()
	defer memoryHistoryMutex.Unlock()
	memoryHistory = make(map[string][]MemoryMetric)
}
