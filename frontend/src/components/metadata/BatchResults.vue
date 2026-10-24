<script setup>
import { computed } from 'vue'

const props = defineProps({
    result: { type: Object, default: null },
    msgs: { type: Object, default: () => ({}) },
})

const progress = computed(() => {
    const current = props.result
    if (!current) return 0
    const processed = Number(current.updated || 0) + Number(current.skipped || 0) +
        Number(current.failed || 0) + Number(current.blocked || 0)
    const total = Number(current.total || 0)
    return total > 0 ? Math.min(100, Math.round((processed / total) * 100)) : 0
})

function statusLabel(status) {
    const key = `metadataBatchStatus${String(status || '').charAt(0).toUpperCase()}${String(status || '').slice(1)}`
    return props.msgs[key] || status || ''
}
</script>

<template>
    <div v-if="result" class="batch-results library-section">
        <div class="batch-results-head">
            <div class="library-section-title">{{ msgs.metadataBatchResults || 'Batch results' }}</div>
            <div class="batch-progress-label">{{ progress }}%</div>
        </div>
        <div class="batch-progress" role="progressbar" :aria-valuenow="progress" aria-valuemin="0" aria-valuemax="100">
            <span :style="{ width: progress + '%' }"></span>
        </div>
        <div class="batch-counts">
            <span>{{ msgs.metadataBatchUpdated || 'Updated' }}: {{ result.updated || 0 }}</span>
            <span>{{ msgs.metadataBatchSkipped || 'Skipped' }}: {{ result.skipped || 0 }}</span>
            <span>{{ msgs.metadataBatchFailed || 'Failed' }}: {{ result.failed || 0 }}</span>
            <span>{{ msgs.metadataBatchBlocked || 'Blocked' }}: {{ result.blocked || 0 }}</span>
            <span v-if="result.cancelled" class="batch-cancelled">{{ msgs.metadataBatchCancelled || 'Cancelled' }}</span>
        </div>
        <div class="batch-result-list">
            <div v-for="item in result.items" :key="item.path + ':' + item.status" class="batch-result-row" :class="'status-' + item.status">
                <span class="batch-result-status">{{ statusLabel(item.status) }}</span>
                <span class="batch-result-path" :title="item.path">{{ item.path }}</span>
                <span v-if="item.code" class="batch-result-code">{{ item.messageKey || item.code }}</span>
            </div>
        </div>
    </div>
</template>

<style scoped>
.batch-results { display: grid; gap: 8px; }
.batch-results-head { display: flex; justify-content: space-between; align-items: baseline; }
.batch-progress-label { color: var(--text-dim); font-size: 12px; }
.batch-progress { height: 6px; border-radius: 3px; background: var(--bg-card-hover); overflow: hidden; }
.batch-progress span { display: block; height: 100%; background: var(--accent); transition: width .2s ease; }
.batch-counts { display: flex; flex-wrap: wrap; gap: 12px; color: var(--text-dim); font-size: 12px; }
.batch-cancelled { color: var(--warning); }
.batch-result-list { display: grid; gap: 3px; max-height: 220px; overflow: auto; }
.batch-result-row { display: flex; gap: 8px; align-items: baseline; font-size: 11px; color: var(--text-dim); }
.batch-result-status { min-width: 62px; color: var(--text); }
.batch-result-row.status-failed .batch-result-status { color: var(--error); }
.batch-result-row.status-blocked .batch-result-status { color: var(--warning); }
.batch-result-path { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.batch-result-code { color: var(--text-muted); }
</style>
