<template>
    <div class="modal-overlay" style="z-index: 110;">
        <div class="modal-backdrop" @click="$emit('close')"></div>
        <div class="modal-content" style="max-width: 48rem; max-height: 90vh;">
            <div class="modal-header">
                <h3 style="font-size: 1.125rem; font-weight: 700;">交易详情</h3>
                <button @click="$emit('close')" class="modal-close">×</button>
            </div>
            <div class="modal-body">
                <div v-if="loading" class="flex-row justify-center" style="padding: 2.5rem 0;">
                    <div
                        style="width: 2rem; height: 2rem; border: 4px solid var(--color-primary); border-top-color: transparent; border-radius: 50%; animation: spin 1s linear infinite;">
                    </div>
                </div>
                <div v-else-if="txDetail" class="flex-col gap-4">
                    <div class="detail-grid" style="font-size: 0.875rem; gap: 1rem;">
                        <div class="info-box" style="grid-column: span 2;">
                            <div class="info-label">交易 ID (Tx ID)</div>
                            <div class="info-value select-all font-mono" style="font-weight: 700;">{{ txDetail.txId
                                }}</div>
                        </div>
                        <div class="info-box">
                            <div class="info-label">时间 (Time)</div>
                            <div class="info-value" style="font-weight: 700;">{{ formatTimestamp(txDetail.timestamp)
                                }}</div>
                        </div>
                        <div class="info-box">
                            <div class="info-label">交易类型 (Type)</div>
                            <div class="info-value" style="font-weight: 700;">{{ formatTxType(txDetail.type) }}
                            </div>
                        </div>
                        <div class="info-box"
                            :style="txDetail.validationCode === 0 ? 'background: #f0fdf4; border-color: #bbf7d0;' : 'background: #fef2f2; border-color: #fecaca;'">
                            <div class="info-label"
                                :style="{color: txDetail.validationCode === 0 ? '#16a34a' : '#dc2626'}">验证状态
                                (Status)</div>
                            <div style="font-weight: 700;"
                                :style="{color: txDetail.validationCode === 0 ? '#15803d' : '#b91c1c'}">{{
                                txDetail.validationCode === 0 ? '有效 (VALID)' : '无效 (Code: ' +
                                txDetail.validationCode + ')' }}</div>
                        </div>
                        <div class="info-box">
                            <div class="info-label">数据大小 (Size)</div>
                            <div class="info-value" style="font-weight: 700;">{{ formatBytes(txDetail.size) }}</div>
                        </div>
                        <div v-if="txDetail.creatorDomain" class="info-box" style="grid-column: span 2;">
                            <div class="info-label">交易发起者 (Creator Domain)</div>
                            <div class="info-value font-mono"
                                style="font-weight: 700; color: var(--color-primary);">{{ txDetail.creatorDomain }}
                            </div>
                        </div>
                    </div>

                    <div v-if="txDetail.chainCodeInfos && txDetail.chainCodeInfos.length > 0">
                        <h4
                            style="font-weight: 700; border-left: 4px solid var(--color-primary); padding-left: 0.75rem; margin-bottom: 0.5rem; color: #1e293b;">
                            链码调用 (Arguments)</h4>
                        <div v-for="(cc, idx) in txDetail.chainCodeInfos" :key="idx"
                            style="background: #f8fafc; padding: 1rem; border-radius: 0.25rem; border: 1px solid var(--border-color); margin-bottom: 0.5rem;">
                            <div
                                style="color: var(--color-primary); font-weight: 700; margin-bottom: 0.5rem; display: flex; align-items: center; gap: 0.5rem;">
                                <span class="badge badge-blue">Name</span> {{ cc.chainCodeName }}
                            </div>
                            <div class="flex-col gap-2">
                                <div v-for="(arg, i) in cc.args" :key="i"
                                    style="font-size: 0.75rem; font-family: monospace; color: #475569; background: white; padding: 0.5rem; border: 1px solid var(--border-color); border-radius: 0.25rem; word-break: break-all; display: flex;">
                                    <span
                                        style="color: #94a3b8; margin-right: 0.5rem; user-select: none; font-weight: 700; white-space: nowrap;">Arg[{{i}}]</span><span>{{
                                        arg }}</span>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { formatTimestamp, formatTxType, formatBytes } from '../utils';

const props = defineProps(['txId']);
const emit = defineEmits(['close']);

const txDetail = ref(null);
const loading = ref(true);

onMounted(async () => {
    try {
        const r = await fetch(`/valuechain/transaction?txID=${props.txId}`);
        txDetail.value = await r.json();
    } catch (e) {
        console.error(e);
    } finally {
        loading.value = false;
    }
});
</script>

<style scoped>
@keyframes spin {
    from {
        transform: rotate(0deg);
    }
    to {
        transform: rotate(360deg);
    }
}
</style>