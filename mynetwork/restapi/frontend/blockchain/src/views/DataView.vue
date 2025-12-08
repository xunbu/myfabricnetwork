<template>
    <div class="flex-col gap-4">
        <div class="card-panel" style="padding: 1.25rem; display: flex; flex-direction: column; gap: 1rem;">
            <div class="flex-row gap-4" style="width: 100%; flex-wrap: wrap; justify-content: space-between;">
                <div class="flex-row gap-2" style="flex-grow: 1;">
                    <input v-model="searchTerm" @keyup.enter="filterChainData" type="text"
                        placeholder="搜索当前页键名 (Key)..." class="form-input" style="flex-grow: 1; max-width: 400px;">
                    <button @click="showAddDataModal" class="btn btn-primary" style="white-space: nowrap;">+
                        新增</button>
                </div>
                <div class="flex-row items-center gap-2"
                    style="background: #f8fafc; padding: 0.25rem; border-radius: 0.5rem; border: 1px solid var(--border-color);">
                    <button @click="prevDataPage" :disabled="dataPageNum === 0" class="btn btn-ghost"
                        style="padding: 0.25rem 0.5rem; font-size: 0.75rem;">上一页</button>
                    <span class="font-mono"
                        style="color: var(--color-primary); font-weight: 700; padding: 0 0.25rem; font-size: 0.875rem;">第
                        {{ dataPageNum + 1 }} 页</span>
                    <button @click="nextDataPage" :disabled="chainData.length < dataPageSize" class="btn btn-ghost"
                        style="padding: 0.25rem 0.5rem; font-size: 0.75rem;">下一页</button>
                </div>
            </div>
        </div>

        <div class="card-panel card-padding" style="padding: 0;">
            <table class="data-table" style="width: 100%;">
                <thead>
                    <tr>
                        <th style="width: 25%;">键名 (Key)</th>
                        <th style="width: 50%;">键值 (Value)</th>
                        <th style="text-align: right;">操作</th>
                    </tr>
                </thead>
                <tbody>
                    <tr v-if="filteredData.length === 0">
                        <td colspan="3"
                            style="padding: 2rem; text-align: center; color: #94a3b8; font-style: italic;">暂无数据</td>
                    </tr>
                    <tr v-for="(item, idx) in filteredData" :key="item.key">
                        <td class="font-mono"
                            style="color: var(--color-primary); font-weight: 700; vertical-align: top;">{{ item.key
                            }}</td>
                        <td style="vertical-align: top; color: #475569;">
                            <div v-if="item.isJson">
                                <button @click="toggleJsonDisplay(idx)" class="btn-text"
                                    style="font-size: 0.75rem; font-weight: 700; margin-bottom: 0.25rem;">{{
                                    item.collapsed ? '▶ 展开 JSON' : '▼ 折叠' }}</button>
                                <pre v-if="!item.collapsed"
                                    style="background: #f8fafc; padding: 0.75rem; border-radius: 0.25rem; border: 1px solid var(--border-color); font-size: 0.75rem; color: #475569; overflow-x: auto;">{{ item.formattedValue }}</pre>
                                <div v-else class="truncate">{{ item.value.substring(0, 50) }}...</div>
                            </div>
                            <div v-else class="break-all">{{ truncateText(item.value, 100) }}</div>
                        </td>
                        <td style="text-align: right; vertical-align: top;">
                            <div class="flex-row justify-between" style="justify-content: flex-end; gap: 0.5rem;">
                                <button @click="showKeyHistory(item.key)" class="btn btn-ghost"
                                    style="padding: 0.25rem 0.5rem; font-size: 0.75rem;">历史</button>
                                <button @click="deleteKey(item.key)" class="btn-text btn-text-danger"
                                    style="font-weight: 700; font-size: 0.75rem;">删除</button>
                            </div>
                        </td>
                    </tr>
                </tbody>
            </table>
        </div>

        <!-- Add Data Modal -->
        <div v-if="showAddModal" class="modal-overlay">
            <div class="modal-backdrop" @click="showAddModal=false"></div>
            <div class="modal-content" style="max-width: 32rem;">
                <div class="modal-header">
                    <h3 style="font-size: 1.125rem; font-weight: 700; color: var(--color-success);">新数据上链</h3>
                    <button @click="showAddModal=false" class="modal-close">×</button>
                </div>
                <div class="modal-body flex-col gap-4">
                    <div>
                        <label class="info-label" style="display: block; margin-bottom: 0.25rem;">键名 (KEY)</label>
                        <input v-model="newKey" class="form-input">
                    </div>
                    <div>
                        <label class="info-label" style="display: block; margin-bottom: 0.25rem;">键值 (VALUE
                            JSON)</label>
                        <textarea v-model="newValue" rows="5" class="form-textarea font-mono"></textarea>
                    </div>
                    <button @click="submitData" :disabled="addSubmitting" class="btn"
                        style="background-color: var(--color-success); color: white; width: 100%;">{{ addSubmitting
                        ? '提交中...' : '提交上链' }}</button>
                </div>
            </div>
        </div>

        <!-- History Modal -->
        <div v-if="showHistoryModal" class="modal-overlay">
            <div class="modal-backdrop" @click="showHistoryModal=false"></div>
            <div class="modal-content" style="max-width: 56rem; max-height: 80vh;">
                <div class="modal-header">
                    <h3 style="font-size: 1.125rem; font-weight: 700;">键值历史: <span class="font-mono"
                            style="color: var(--color-primary);">{{ historyKey }}</span></h3>
                    <button @click="showHistoryModal=false" class="modal-close">×</button>
                </div>
                <div class="modal-body" style="padding: 0;">
                    <table class="data-table" style="width: 100%;">
                        <thead style="position: sticky; top: 0; background: #f8fafc; z-index: 10;">
                            <tr>
                                <th>交易 ID</th>
                                <th>时间</th>
                                <th>值</th>
                                <th>类型</th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="h in historyData" :key="h.txId">
                                <td><button @click="openTxFromHistory(h.txId)" class="btn-text font-mono">{{
                                        h.txId.substring(0,10) }}...</button></td>
                                <td>{{ formatTimestamp(h.timestamp) }}</td>
                                <td class="font-mono" style="font-size: 0.75rem;">{{ truncateText(h.value, 40) }}
                                </td>
                                <td><span class="badge" :class="h.isDelete ? 'badge-danger' : 'badge-success'">{{
                                        h.isDelete ? '删除' : '写入' }}</span></td>
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
import { ref, onMounted } from 'vue';
import TxModal from '../components/TxModal.vue';
import { truncateText, formatTimestamp } from '../utils';

const chainData = ref([]);
const filteredData = ref([]);
const dataPageNum = ref(0);
const dataPageSize = ref(10);
const searchTerm = ref('');
const showAddModal = ref(false);
const showHistoryModal = ref(false);
const showTxModal = ref(false);
const currentTxId = ref('');
const newKey = ref('');
const newValue = ref('');
const addSubmitting = ref(false);
const historyKey = ref('');
const historyData = ref([]);

const loadChainData = async () => {
    try {
        const r = await fetch(`/valuechain/data/all?pageNum=${dataPageNum.value}&pageSize=${dataPageSize.value}`);
        if (r.ok) {
            chainData.value = (await r.json()).results || [];
            filterChainData();
        }
    } catch (e) { console.error(e); }
};

const filterChainData = () => {
    let d = chainData.value;
    if (searchTerm.value) d = d.filter(i => i.key.includes(searchTerm.value));
    filteredData.value = d.map(i => ({
        ...i,
        isJson: i.value && i.value.trim().startsWith('{'),
        formattedValue: i.value && i.value.trim().startsWith('{') ? tryFormatJson(i.value) : i.value,
        collapsed: true
    }));
};

const tryFormatJson = (str) => { try { return JSON.stringify(JSON.parse(str), null, 2); } catch (e) { return str; } };
const toggleJsonDisplay = i => filteredData.value[i].collapsed = !filteredData.value[i].collapsed;
const prevDataPage = () => { if (dataPageNum.value > 0) { dataPageNum.value--; loadChainData(); } };
const nextDataPage = () => { if (chainData.value.length === dataPageSize.value) { dataPageNum.value++; loadChainData(); } };
const showAddDataModal = () => { newKey.value = ''; newValue.value = ''; showAddModal.value = true; };
const submitData = async () => {
    addSubmitting.value = true;
    try {
        await fetch('/valuechain/data', { method: 'POST', body: JSON.stringify({ key: newKey.value, value: newValue.value }) });
        showAddModal.value = false; loadChainData();
    } finally { addSubmitting.value = false; }
};
const deleteKey = async (k) => { if (confirm('确认删除?')) { await fetch('/valuechain/data/delete', { method: 'POST', body: JSON.stringify({ key: k }) }); loadChainData(); } };
const showKeyHistory = async (k) => { historyKey.value = k; showHistoryModal.value = true; const r = await fetch(`/valuechain/data/history?key=${k}`); historyData.value = await r.json(); };
const openTxFromHistory = (txId) => { currentTxId.value = txId; showTxModal.value = true; };
const triggerRefresh = () => loadChainData();

defineExpose({ triggerRefresh });
onMounted(loadChainData);
</script>