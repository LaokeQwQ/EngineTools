<script setup>
import { computed, onBeforeUnmount, watch } from 'vue'
import { METADATA_FIELDS, PATCH_MODES } from '../../composables/useMetadataEditor.js'
import { createMetadataBatch } from '../../composables/useMetadataBatch.js'
import BatchResults from './BatchResults.vue'

const emit = defineEmits(['dirty-change'])

const props = defineProps({
    msgs: { type: Object, default: () => ({}) },
})

const batch = createMetadataBatch()
watch(() => batch.dirty.value, (dirty) => emit('dirty-change', Boolean(dirty)), { immediate: true })
onBeforeUnmount(() => { batch.stopPolling(); emit('dirty-change', false) })
const batchFields = METADATA_FIELDS.filter((field) => field.name !== 'pictures')

const fieldLabel = (field) => props.msgs[`metadata${field.name[0].toUpperCase()}${field.name.slice(1)}Label`] || field.label
const modeFor = (field) => batch.modes.value[field.name] || PATCH_MODES.unchanged
const isIncluded = (field) => modeFor(field) !== PATCH_MODES.unchanged
const isSet = (field) => modeFor(field) === PATCH_MODES.set

const pathsSummary = computed(() => {
    const count = batch.paths.value.length
    return count
        ? `${count} ${props.msgs.metadataBatchFiles || 'files'}`
        : (props.msgs.metadataBatchNoFiles || 'Choose a folder to add files')
})

const previewSummary = computed(() => {
    const current = batch.preview.value
    if (!current) return ''
    return `${current.total} ${props.msgs.metadataBatchFiles || 'files'} · ${current.eligible} ${props.msgs.metadataBatchEligible || 'eligible'} · ${current.blocked} ${props.msgs.metadataBatchBlocked || 'blocked'}`
})

function toggle(field, event) {
    batch.setMode(field.name, event.target.checked ? PATCH_MODES.set : PATCH_MODES.unchanged)
}

function setMode(field, mode) {
    batch.setMode(field.name, mode)
}

function displayValue(field) {
    const value = batch.values.value[field.name]
    if (field.kind === 'list') return Array.isArray(value) ? value.join('; ') : ''
    if (field.kind === 'numberPair') return `${value?.value || 0}/${value?.total || 0}`
    return value ?? ''
}

function updateValue(field, event) {
    batch.setValue(field.name, event.target.value)
}
</script>

<template>
    <div class="batch-editor">
        <div class="library-section batch-toolbar">
            <div>
                <div class="library-section-title">{{ msgs.metadataBatchTitle || 'Batch metadata editor' }}</div>
                <div class="library-stats">{{ pathsSummary }}</div>
            </div>
            <div class="batch-toolbar-actions">
                <button class="btn btn-secondary no-drag" type="button" :disabled="batch.busy.value" @click="batch.chooseDirectory()">
                    {{ msgs.metadataBatchChooseFolder || 'Choose folder' }}
                </button>
                <button class="btn btn-secondary no-drag" type="button" :disabled="batch.busy.value || batch.paths.value.length === 0" @click="batch.clearPaths()">
                    {{ msgs.metadataBatchClearFiles || 'Clear files' }}
                </button>
            </div>
        </div>

        <div v-if="batch.paths.value.length" class="library-section batch-file-list">
            <div class="batch-file-row" v-for="path in batch.paths.value.slice(0, 80)" :key="path">
                <span :title="path">{{ path }}</span>
                <button class="batch-remove no-drag" type="button" :disabled="batch.busy.value" @click="batch.removePath(path)">×</button>
            </div>
            <div v-if="batch.paths.value.length > 80" class="library-stats">… {{ batch.paths.value.length - 80 }} more</div>
        </div>

        <div class="library-section batch-fields">
            <div class="batch-field" v-for="field in batchFields" :key="field.name">
                <label class="batch-field-head">
                    <input type="checkbox" :checked="isIncluded(field)" :disabled="batch.busy.value" @change="toggle(field, $event)" />
                    <span>{{ fieldLabel(field) }}</span>
                </label>
                <div v-if="isIncluded(field)" class="batch-field-controls">
                    <button class="metadata-mode-btn" :class="{ active: isSet(field) }" type="button" :disabled="batch.busy.value" @click="setMode(field, PATCH_MODES.set)">{{ msgs.metadataSet || 'Set' }}</button>
                    <button class="metadata-mode-btn" :class="{ active: modeFor(field) === PATCH_MODES.clear }" type="button" :disabled="batch.busy.value" @click="setMode(field, PATCH_MODES.clear)">{{ msgs.metadataClear || 'Clear' }}</button>
                    <textarea v-if="isSet(field) && field.kind === 'textarea'" class="text-input no-drag batch-input" :value="displayValue(field)" :disabled="batch.busy.value" @input="updateValue(field, $event)" />
                    <input v-else-if="isSet(field)" class="text-input no-drag batch-input" :type="field.kind === 'number' ? 'number' : 'text'" :value="displayValue(field)" :disabled="batch.busy.value" @input="updateValue(field, $event)" />
                </div>
            </div>
        </div>

        <div class="library-section batch-actions">
            <button class="btn btn-secondary no-drag" type="button" :disabled="!batch.dirty.value || batch.paths.value.length === 0 || batch.busy.value" @click="batch.runPreview()">{{ msgs.metadataBatchPreview || 'Preview' }}</button>
            <button class="btn btn-primary no-drag" type="button" :disabled="!batch.dirty.value || batch.paths.value.length === 0 || batch.busy.value" @click="batch.start()">{{ msgs.metadataBatchStart || 'Start batch' }}</button>
            <button v-if="batch.polling.value" class="btn btn-secondary no-drag" type="button" @click="batch.cancel()">{{ msgs.metadataCancel || 'Cancel' }}</button>
            <span v-if="batch.error.value" class="batch-error">{{ msgs[batch.error.value.messageKey] || batch.error.value.messageKey }}</span>
            <span v-else-if="batch.busy.value" class="library-stats">{{ msgs.metadataBatchRunning || 'Running...' }}</span>
        </div>

        <div v-if="batch.preview.value" class="library-section batch-preview">
            <div class="library-section-title">{{ msgs.metadataBatchPreviewTitle || 'Preview' }}</div>
            <div class="library-stats">{{ previewSummary }}</div>
            <div class="batch-result-list">
                <div v-for="item in batch.preview.value.items" :key="item.path" class="batch-result-row" :class="'status-' + item.status">
                    <span class="batch-result-status">{{ item.status }}</span>
                    <span class="batch-result-path" :title="item.path">{{ item.path }}</span>
                    <span v-if="item.code" class="batch-result-code">{{ item.messageKey || item.code }}</span>
                </div>
            </div>
        </div>

        <BatchResults :result="batch.result.value" :msgs="msgs" />
    </div>
</template>

<style scoped>
.batch-editor { display: grid; gap: 10px; }
.batch-toolbar { display: flex; justify-content: space-between; gap: 12px; align-items: center; }
.batch-toolbar-actions { display: flex; gap: 8px; }
.batch-file-list { max-height: 180px; overflow: auto; display: grid; gap: 3px; }
.batch-file-row { display: flex; gap: 8px; font-size: 11px; color: var(--text-dim); }
.batch-file-row span { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.batch-remove { border: 0; background: transparent; color: var(--text-muted); cursor: pointer; font-size: 14px; line-height: 1; }
.batch-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px 12px; }
.batch-field { min-width: 0; }
.batch-field-head { display: flex; gap: 6px; align-items: center; color: var(--text-dim); font-size: 12px; }
.batch-field-controls { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 4px; }
.batch-field-controls .metadata-mode-btn { border: 1px solid var(--border); background: transparent; color: var(--text-dim); font-size: 10px; padding: 2px 6px; cursor: pointer; }
.batch-field-controls .metadata-mode-btn.active { color: var(--text); background: var(--bg-card-hover); border-color: var(--accent); }
.batch-input { min-height: 30px; flex: 1 1 100%; }
.batch-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.batch-error { color: var(--error); font-size: 12px; }
.batch-preview { display: grid; gap: 6px; }
.batch-result-list { display: grid; gap: 3px; max-height: 220px; overflow: auto; }
.batch-result-row { display: flex; gap: 8px; align-items: baseline; font-size: 11px; color: var(--text-dim); }
.batch-result-status { min-width: 62px; color: var(--text); }
.batch-result-row.status-failed .batch-result-status { color: var(--error); }
.batch-result-row.status-blocked .batch-result-status { color: var(--warning); }
.batch-result-path { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.batch-result-code { color: var(--text-muted); }
@media (max-width: 640px) {
    .batch-fields { grid-template-columns: 1fr; }
    .batch-toolbar { flex-direction: column; align-items: flex-start; }
}
</style>
