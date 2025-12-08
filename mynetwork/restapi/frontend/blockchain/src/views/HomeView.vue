<template>
    <div class="flex-col gap-4">
        <!-- 统计卡片 -->
        <div class="stats-grid">
            <div v-for="(item, key) in overviewItems" :key="key" class="stat-card">
                <div class="stat-icon">{{ item.icon }}</div>
                <div class="stat-label">{{ item.label }}</div>
                <div class="stat-value">{{ formatNumber(overview[item.key]) }}</div>
            </div>
        </div>

        <!-- 图表 -->
        <div class="charts-grid">
            <div class="card-panel card-padding chart-container">
                <h3 class="chart-title">
                    <span class="title-mark bg-mark-primary"></span> 交易趋势
                </h3>
                <div class="canvas-wrapper"><canvas ref="txChartEl"></canvas></div>
            </div>
            <div class="card-panel card-padding chart-container">
                <h3 class="chart-title">
                    <span class="title-mark bg-mark-accent"></span> 出块统计
                </h3>
                <div class="canvas-wrapper"><canvas ref="blockChartEl"></canvas></div>
            </div>
        </div>

        <!-- 区块流 -->
        <div class="card-panel chain-view-container">
            <div class="chain-header flex-row items-center gap-2">
                <span>⛓️ 实时区块流 (最新 50 块)</span>
            </div>
            <div class="chain-scroll-area">
                <div class="chain-list">
                    <div v-for="block in trendBlocks" :key="block.blockNumber" class="chain-item"
                        @click="goToBlock(block.blockNumber)">
                        <div class="chain-link">
                            <div class="block-card">
                                <div class="flex-row justify-between" style="align-items: flex-start;">
                                    <span class="font-mono"
                                        style="font-size: 1.5rem; font-weight: 700; color: #1e293b;">#{{
                                        block.blockNumber }}</span>
                                    <span class="badge badge-blue">{{ block.txCount }} 笔</span>
                                </div>
                                <div class="flex-col gap-2">
                                    <div>
                                        <div class="info-label">哈希 (Hash)</div>
                                        <div class="font-mono break-all"
                                            style="font-size: 10px; color: #475569; line-height: 1.2;">{{
                                            block.blockHash.substring(0, 16) }}...</div>
                                    </div>
                                </div>
                                <div class="flex-row justify-between font-mono"
                                    style="border-top: 1px solid #f1f5f9; padding-top: 0.75rem; font-size: 10px; color: #94a3b8;">
                                    <span>{{ formatTimestamp(block.timestamp).split(' ')[1] }}</span>
                                    <span>{{ formatBytes(block.blockSize) }}</span>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- 省略号 -->
                    <div v-if="trendBlocks.length >= 50" class="chain-item"
                        style="opacity: 0.5; padding-left: 1rem;">
                        <div class="flex-col items-center justify-center gap-2">
                            <div
                                style="font-size: 2.25rem; font-weight: 700; color: #cbd5e1; letter-spacing: 0.1em;">
                                •••</div>
                            <div class="info-label">History</div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- 区块列表表格 -->
        <div class="card-panel card-padding">
            <div class="flex-row justify-between items-center" style="margin-bottom: 1.5rem;">
                <h2 style="font-size: 1.125rem; font-weight: 700;">区块账本记录</h2>
                <div class="flex-row items-center gap-2">
                    <button @click="prevPage" :disabled="blockPage === 0" class="btn btn-ghost">上一页</button>
                    <span class="font-mono"
                        style="color: var(--color-primary); font-weight: 700; font-size: 0.875rem;">第 {{ blockPage +
                        1 }} 页</span>
                    <button @click="nextPage" :disabled="(blockPage + 1) * pageSize >= overview.blockHeight"
                        class="btn btn-ghost">下一页</button>
                </div>
            </div>
            <div class="table-container">
                <table class="data-table">
                    <thead>
                        <tr>
                            <th>区块号</th>
                            <th>哈希值</th>
                            <th style="text-align: center;">交易数</th>
                            <th>生成时间</th>
                            <th style="text-align: right;">大小</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr v-for="block in blocks" :key="block.blockNumber" @click="goToBlock(block.blockNumber)"
                            style="cursor: pointer;">
                            <td class="font-mono" style="font-weight: 700; color: var(--color-primary);">#{{
                                block.blockNumber }}</td>
                            <td class="font-mono" style="font-size: 0.75rem; color: var(--text-sub);">{{
                                block.blockHash.substring(0, 32) }}...</td>
                            <td style="text-align: center;"><span class="badge badge-purple">{{ block.txCount
                                    }}</span></td>
                            <td class="font-mono" style="font-size: 0.75rem;">{{ formatTimestamp(block.timestamp) }}
                            </td>
                            <td class="font-mono" style="text-align: right; font-size: 0.75rem;">{{
                                formatBytes(block.blockSize) }}</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';
import Chart from 'chart.js/auto';
import { formatNumber, formatTimestamp, formatBytes } from '../utils';

const router = useRouter();
const overview = reactive({});
const blocks = ref([]);
const trendBlocks = ref([]);
const blockPage = ref(0);
const pageSize = 10;
const txChartEl = ref(null);
const blockChartEl = ref(null);

let txChart = null, blockChart = null, timer = null;

const overviewItems = {
    blockHeight: { label: '区块高度', key: 'blockHeight', icon: '⛓' },
    totalTransactionCount: { label: '总交易量', key: 'totalTransactionCount', icon: '⚡' },
    orgCount: { label: '组织数量', key: 'orgCount', icon: '🏢' },
    chainCodeCount: { label: '智能合约', key: 'chainCodeCount', icon: '📜' },
    nodeCount: { label: '活跃节点', key: 'nodeCount', icon: '🖥️' }
};

const refreshData = async () => {
    try {
        const [infoRes, blockRes, trendRes] = await Promise.all([
            fetch('/valuechain/info'),
            fetch(`/valuechain/blocks?pageNum=${blockPage.value}&pageSize=${pageSize}`),
            fetch('/valuechain/trend')
        ]);

        if (infoRes.ok) Object.assign(overview, await infoRes.json());
        if (blockRes.ok) blocks.value = (await blockRes.json()).blockPage || [];

        if (trendRes.ok) {
            const rawData = (await trendRes.json()).blockPage || [];
            processCharts(rawData);
            trendBlocks.value = rawData.slice(0, 50);
        }
    } catch (e) { console.error("Home refresh failed", e); }
};

const processCharts = (rawBlocks) => {
    if (!txChartEl.value || !blockChartEl.value) return;

    const buckets = {};
    const labels = [];
    const daysToShow = 14;

    for (let i = daysToShow - 1; i >= 0; i--) {
        const d = new Date();
        d.setDate(d.getDate() - i);
        const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
        labels.push(`${d.getMonth() + 1}/${d.getDate()}`);
        buckets[key] = { tx: 0, blk: 0 };
    }

    rawBlocks.forEach(b => {
        const d = new Date(b.timestamp);
        const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
        if (buckets[key]) {
            buckets[key].blk++;
            buckets[key].tx += b.txCount;
        }
    });

    const opts = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: { legend: false },
        scales: {
            x: { grid: { display: false } },
            y: { grid: { color: '#e2e8f0', borderDash: [4, 4] }, beginAtZero: true }
        }
    };

    if (txChart) txChart.destroy();
    txChart = new Chart(txChartEl.value, {
        type: 'line',
        data: {
            labels,
            datasets: [{
                data: Object.values(buckets).map(v => v.tx),
                borderColor: '#2563eb',
                backgroundColor: '#2563eb20',
                fill: true,
                tension: 0.4
            }]
        },
        options: opts
    });

    if (blockChart) blockChart.destroy();
    blockChart = new Chart(blockChartEl.value, {
        type: 'bar',
        data: {
            labels,
            datasets: [{
                data: Object.values(buckets).map(v => v.blk),
                backgroundColor: '#9333ea',
                borderRadius: 3
            }]
        },
        options: opts
    });
};

const prevPage = () => { if (blockPage.value > 0) { blockPage.value--; refreshData(); } };
const nextPage = () => { blockPage.value++; refreshData(); };
const goToBlock = (num) => router.push('/block/' + num);
const triggerRefresh = () => refreshData();

defineExpose({ triggerRefresh });

onMounted(() => { refreshData(); timer = setInterval(refreshData, 10000); });
onUnmounted(() => { clearInterval(timer); if (txChart) txChart.destroy(); if (blockChart) blockChart.destroy(); });
</script>