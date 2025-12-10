<template>
    <div class="flex-col gap-4">
        <div class="card-panel"
            style="padding: 1rem; display: flex; flex-wrap: wrap; gap: 1rem; justify-content: space-between; align-items: center;">
            <div
                style="background-color: #f1f5f9; padding: 0.25rem; border-radius: 0.5rem; display: flex; gap: 0.25rem;">
                <button @click="switchNodeTab('cpu')" class="btn" style="padding: 0.375rem 1.5rem;"
                    :style="nodeTab === 'cpu' ? 'background: white; box-shadow: var(--shadow-sm); color: var(--color-primary);' : 'background: transparent; color: var(--text-sub);'">
                    CPU
                </button>
                <button @click="switchNodeTab('memory')" class="btn" style="padding: 0.375rem 1.5rem;"
                    :style="nodeTab === 'memory' ? 'background: white; box-shadow: var(--shadow-sm); color: var(--color-accent);' : 'background: transparent; color: var(--text-sub);'">
                    内存
                </button>
            </div>
            <div class="flex-row gap-4">
                <select v-model="timeRange" @change="forceRefreshNode" class="form-select"
                    style="width: auto; padding-top: 0.25rem; padding-bottom: 0.25rem;">
                    <option value="1">最近 1 分钟</option>
                    <option value="5">最近 5 分钟</option>
                    <option value="30">最近 30 分钟</option>
                </select>
                <select v-model="refreshRate" @change="startNodeRefresh" class="form-select"
                    style="width: auto; padding-top: 0.25rem; padding-bottom: 0.25rem;">
                    <option value="3">3s 刷新</option>
                    <option value="5">5s 刷新</option>
                    <option value="0">停止刷新</option>
                </select>
            </div>
        </div>
        <div class="nodes-grid">
            <div v-for="(node, name) in (nodeTab === 'cpu' ? cpuNodes : memoryNodes)" :key="name"
                class="card-panel card-padding node-card flex-col"
                :class="nodeTab === 'cpu' ? 'border-primary' : 'border-accent'">
                <div class="flex-row justify-between items-start" style="margin-bottom: 1rem;">
                    <h3 class="truncate" :title="name" style="font-weight: 700; color: #334155; max-width: 70%;">{{
                        name }}</h3>
                    <div style="text-align: right;">
                        <div class="font-mono" style="font-size: 1.875rem; font-weight: 700;"
                            :class="getUsageColor(node.latestUsage)">{{ node.latestUsage.toFixed(1) }}%</div>
                    </div>
                </div>
                <div style="height: 12rem; width: 100%; position: relative;">
                    <!-- 在 Vue 3 中可以使用 :id 来绑定唯一的 ID -->
                    <canvas :id="`${nodeTab}-chart-${name}`"></canvas>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, reactive, nextTick, onMounted, onUnmounted } from 'vue';
import Chart from 'chart.js/auto';
import { getUsageColor } from '../utils';

defineOptions({
    name: 'NodesPage'
});

const nodeTab = ref('cpu');
const timeRange = ref('1');
const refreshRate = ref('5');
const cpuNodes = reactive({});
const memoryNodes = reactive({});
const charts = {};
let timer = null;

const calculateYAxisMax = (dataValues) => {
    const maxVal = Math.max(...dataValues, 0);
    let limit = Math.ceil((maxVal * 1.2) / 10) * 10;
    if (limit < 10) limit = 10;
    if (limit > 100) limit = 100;
    return limit;
};

const refreshNodeData = async () => {
    const currentTab = nodeTab.value;
    const url = currentTab === 'cpu' ? '/valuechain/cpuHistory' : '/valuechain/memoryHistory';
    try {
        const res = await fetch(url);
        if (!res.ok) return;
        const data = await res.json();
        const targetNodes = currentTab === 'cpu' ? cpuNodes : memoryNodes;
        const cutoffTime = Date.now() - (parseInt(timeRange.value) * 60 * 1000);

        Object.keys(data).forEach(name => {
            const hist = data[name] || [];
            const last = hist[hist.length - 1] || {};
            targetNodes[name] = {
                latestUsage: currentTab === 'cpu'
                    ? (last.CPUUsage || 0)
                    : ((last.TotalMemory > 0) ? (last.UsedMemory / last.TotalMemory * 100) : 0)
            };
        });

        nextTick(() => {
            Object.keys(data).forEach(name => {
                const hist = data[name] || [];
                // const filtered = hist.filter(d => new Date(d.Timestamp).getTime() > cutoffTime);
                let filtered = [];

                if (hist.length > 0) {
                    // 1. 获取这份数据里【最新】的一个时间点 (也就是数组最后一条)
                    const lastItemTime = new Date(hist[hist.length - 1].Timestamp).getTime();

                    // 2. 以【最新数据时间】为基准，向前推算 N 分钟
                    const rangeInMs = parseInt(timeRange.value) * 60 * 1000;
                    const cutoffTime = lastItemTime - rangeInMs;

                    // 3. 进行过滤
                    filtered = hist.filter(d => new Date(d.Timestamp).getTime() > cutoffTime);
                }

                const labels = filtered.map(d => new Date(d.Timestamp).toLocaleTimeString());
                const values = filtered.map(d => currentTab === 'cpu'
                    ? (d.CPUUsage || 0)
                    : (d.TotalMemory > 0 ? (d.UsedMemory / d.TotalMemory * 100) : 0)
                );

                const ctxId = `${currentTab}-chart-${name}`;
                const ctx = document.getElementById(ctxId);

                if (!ctx) return;

                const dynamicMax = calculateYAxisMax(values);

                if (charts[ctxId]) {
                    charts[ctxId].data.labels = labels;
                    charts[ctxId].data.datasets[0].data = values;
                    charts[ctxId].options.scales.y.max = dynamicMax;
                    charts[ctxId].update('none');
                } else {
                    const color = currentTab === 'cpu' ? '#2563eb' : '#9333ea';
                    charts[ctxId] = new Chart(ctx, {
                        type: 'line',
                        data: {
                            labels: labels,
                            datasets: [{
                                data: values,
                                borderColor: color,
                                borderWidth: 2,
                                pointRadius: 0,
                                fill: false,
                                tension: 0.2
                            }]
                        },
                        options: {
                            responsive: true,
                            maintainAspectRatio: false,
                            animation: false,
                            plugins: { legend: { display: false } },
                            scales: {
                                x: { display: false },
                                y: {
                                    min: 0,
                                    max: dynamicMax,
                                    ticks: { count: 5 }
                                }
                            }
                        }
                    });
                }
            });
        });
    } catch (e) { console.error(e); }
};

const switchNodeTab = (t) => {
    if (nodeTab.value === t) return;
    Object.keys(charts).forEach(key => { if (charts[key]) { charts[key].destroy(); delete charts[key]; } });
    nodeTab.value = t;
    nextTick(() => { refreshNodeData(); });
};

const startNodeRefresh = () => {
    clearInterval(timer);
    if (refreshRate.value > 0) timer = setInterval(refreshNodeData, refreshRate.value * 1000);
};

const forceRefreshNode = () => { refreshNodeData(); };
const triggerRefresh = () => refreshNodeData();

defineExpose({ triggerRefresh });

onMounted(() => { refreshNodeData(); startNodeRefresh(); });
onUnmounted(() => { clearInterval(timer); Object.values(charts).forEach(c => c.destroy()); });
</script>