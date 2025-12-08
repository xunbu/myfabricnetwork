<template>
    <div class="flex-col gap-4">
        <div class="flex-row justify-between items-center">
            <button @click="$router.push('/')" class="btn btn-ghost">
                <span>←</span> 返回首页
            </button>
            <h2 style="font-size: 1.25rem; font-weight: 700;">区块详情 <span class="font-mono"
                    style="color: var(--color-primary);">#{{ blockDetail.blockNumber }}</span></h2>
        </div>

        <div v-if="loading" class="flex-row justify-center" style="padding: 5rem 0;">
            <div
                style="width: 2.5rem; height: 2.5rem; border: 4px solid var(--color-primary); border-top-color: transparent; border-radius: 50%; animation: spin 1s linear infinite;">
            </div>
        </div>

        <div v-else class="card-panel card-padding">
            <div class="detail-grid" style="margin-bottom: 2rem;">
                <!-- 左侧信息 -->
                <div class="flex-col gap-4">
                    <div class="info-box">
                        <div class="info-label">区块哈希</div>
                        <div class="info-value break-all select-all">{{ blockDetail.blockHash }}</div>
                    </div>
                    <div v-if="blockDetail.dataHash" class="info-box">
                        <div class="info-label">数据哈希</div>
                        <div class="info-value break-all select-all">{{ blockDetail.dataHash }}</div>
                    </div>
                    <div class="info-box" style="cursor: pointer;" @click="goToBlock(blockDetail.blockNumber - 1)"
                        v-if="blockDetail.blockNumber > 0"
                        onmouseover="this.style.borderColor='var(--color-primary)'"
                        onmouseout="this.style.borderColor='var(--border-color)'">
                        <div class="info-label">前块哈希</div>
                        <div class="info-value break-all" style="color: var(--color-primary);">{{
                            blockDetail.previousHash }}</div>
                    </div>
                </div>
                <!-- 右侧统计 -->
                <div class="flex-col gap-4">
                    <div class="detail-grid" style="grid-template-columns: 1fr 1fr; gap: 1rem;">
                        <div class="info-box"
                            style="background-color: var(--color-primary-light); border-color: #dbeafe;">
                            <div class="info-label">交易数量</div>
                            <div class="info-value info-value-lg" style="color: var(--color-primary);">{{
                                blockDetail.txCount }}</div>
                        </div>
                        <div class="info-box"
                            style="background-color: var(--color-accent-light); border-color: #f3e8ff;">
                            <div class="info-label">区块大小</div>
                            <div class="info-value info-value-lg" style="color: var(--color-accent);">{{
                                formatBytes(blockDetail.blockSize) }}</div>
                        </div>
                    </div>
                    <div class="info-box flex-grow">
                        <div class="info-label">生成时间</div>
                        <div class="info-value info-value-lg" style="font-size: 1.125rem;">{{
                            formatTimestamp(blockDetail.timestamp) }}</div>
                    </div>
                </div>
            </div>

            <div style="border-top: 1px solid var(--border-color); padding-top: 1.5rem;">
                <div class="flex-row justify-between items-center" style="margin-bottom: 1rem;">
                    <h3 style="font-size: 1.125rem; font-weight: 700;">包含的交易</h3>
                    <div class="flex-row items-center gap-2" v-if="blockTxTotal > 0">
                        <button @click="prevBlockTxPage" :disabled="blockTxPage === 0" class="btn btn-ghost"
                            style="padding: 0.25rem 0.75rem; font-size: 0.75rem;">上一页</button>
                        <span class="font-mono"
                            style="color: var(--color-primary); font-weight: 700; font-size: 0.75rem;">{{
                            blockTxPage + 1 }} / {{ Math.ceil(blockTxTotal / blockTxPageSize) }}</span>
                        <button @click="nextBlockTxPage"
                            :disabled="(blockTxPage + 1) * blockTxPageSize >= blockTxTotal" class="btn btn-ghost"
                            style="padding: 0.25rem 0.75rem; font-size: 0.75rem;">下一页</button>
                    </div>
                </div>
                <div class="table-container">
                    <table class="data-table">
                        <thead>
                            <tr>
                                <th>交易 ID</th>
                                <th>类型</th>
                                <th>发起者</th>
                                <th>时间戳</th>
                                <th style="text-align: center;">状态</th>
                                <th style="text-align: right;">操作</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="tx in blockTxList" :key="tx.txId">
                                <td><button @click="showTxDetail(tx.txId)" class="btn-text font-mono"
                                        style="font-size: 0.75rem;">{{ tx.txId.substring(0, 20) }}...</button></td>
                                <td style="font-size: 0.75rem;">{{ formatTxType(tx.type) }}</td>
                                <td class="font-mono" style="font-size: 0.75rem;">{{ tx.creatorDomain }}</td>
                                <td class="font-mono" style="font-size: 0.75rem;">{{ formatTimestamp(tx.timestamp)
                                    }}</td>
                                <td style="text-align: center;">
                                    <span class="badge"
                                        :class="tx.validationCode === 0 ? 'badge-success' : 'badge-danger'">{{
                                        tx.validationCode === 0 ? 'VALID' : 'INVALID' }}</span>
                                </td>
                                <td style="text-align: right;">
                                    <button @click="showTxDetail(tx.txId)" class="btn-text"
                                        style="font-weight: 700; font-size: 0.75rem;">详情</button>
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
        <TxModal v-if="showTxModal" :tx-id="currentTxId" @close="showTxModal = false" />
    </div>
</template>

<script setup>
import { ref, reactive, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import TxModal from '../components/TxModal.vue';
import { formatBytes, formatTimestamp, formatTxType } from '../utils';

const route = useRoute();
const router = useRouter();

const loading = ref(true);
const blockDetail = reactive({});
const blockTxList = ref([]);
const blockTxPage = ref(0);
const blockTxPageSize = ref(10);
const blockTxTotal = ref(0);
const showTxModal = ref(false);
const currentTxId = ref('');

const loadBlock = async (bn) => {
    loading.value = true;
    try {
        const r = await fetch(`/valuechain/block?blockNum=${bn}`);
        Object.assign(blockDetail, await r.json());
        blockTxPage.value = 0;
        await loadTxs(bn);
    } catch (e) { console.error(e); } finally { loading.value = false; }
};

const loadTxs = async (bn) => {
    const r = await fetch(`/valuechain/block/transactions?blockNum=${bn}&pageNum=${blockTxPage.value}&pageSize=${blockTxPageSize.value}`);
    if (r.ok) {
        const d = await r.json();
        blockTxList.value = d.txPage || [];
        blockTxTotal.value = d.total || 0;
    }
};

watch(() => route.params.blockId, (newId) => { if (newId) loadBlock(newId); }, { immediate: true });

const prevBlockTxPage = () => { blockTxPage.value--; loadTxs(blockDetail.blockNumber); };
const nextBlockTxPage = () => { blockTxPage.value++; loadTxs(blockDetail.blockNumber); };
const goToBlock = (num) => router.push('/block/' + num);
const showTxDetail = (id) => { currentTxId.value = id; showTxModal.value = true; };
const triggerRefresh = () => loadBlock(route.params.blockId);

defineExpose({ triggerRefresh });
</script>