<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { createMetadataEditor, METADATA_FIELDS, PATCH_MODES, pictureDataURI } from '../composables/useMetadataEditor.js'
import BatchEditor from '../components/metadata/BatchEditor.vue'

const emit = defineEmits(['dirty-change'])

const props = defineProps({
    activeTab: { type: String, default: 'metadata' },
    msgs: { type: Object, default: () => ({}) },
    initialPath: { type: String, default: '' },
})

const editor = createMetadataEditor(props.initialPath)
const mode = ref('single')

watch(() => editor.dirty.value, (dirty) => emit('dirty-change', Boolean(dirty)), { immediate: true })
const fileInput = ref(null)
const pictureFileInput = ref(null)
const pictureTargetIndex = ref(null)
const dragOver = ref(false)
const listDraft = ref({})
const pairDraft = ref({})

function loadFile(path) {
    // Draft inputs are keyed by field name, not by file, so clear them before
    // a new document arrives or stale values leak into the next file's form.
    listDraft.value = {}
    pairDraft.value = {}
    return editor.load(path)
}

const fieldLabel = (field) => props.msgs[`metadata${field.name[0].toUpperCase()}${field.name.slice(1)}Label`] || field.label
const modeLabel = (mode) => ({
    unchanged: props.msgs.metadataUnchanged || 'Unchanged',
    set: props.msgs.metadataSet || 'Set',
    clear: props.msgs.metadataClear || 'Clear',
}[mode])
const displayValue = (field) => {
    const value = editor.doc.value?.[field.name]
    if (field.kind === 'list') return Array.isArray(value) ? value.join('; ') : ''
    if (field.kind === 'numberPair') return value?.value || ''
    return value ?? ''
}
const isWritable = (field) => {
    const writable = editor.doc.value?.capabilities?.writable
    // Older bridges omit capabilities altogether; retain their editable
    // behavior. Once a capability map is present, absence is unsupported and
    // only an explicit true (including legacy singular aliases) is writable.
    if (!writable || typeof writable !== 'object') return true
    const aliases = {
        artists: 'artist',
        albumArtists: 'albumArtist',
        genres: 'genre',
        composers: 'composer',
    }
    return [field.name, aliases[field.name]].filter(Boolean).some((name) => writable[name] === true)
}
const statusText = computed(() => {
    if (editor.loading.value) return props.msgs.metadataLoading || 'Reading metadata...'
    if (editor.saving.value) return props.msgs.metadataSaving || 'Saving...'
    if (editor.undoing.value) return props.msgs.metadataUndoing || 'Undoing...'
    if (editor.error.value) return props.msgs[editor.error.value.messageKey] || editor.error.value.messageKey
    if (editor.dirty.value) return props.msgs.metadataUnsaved || 'Unsaved changes'
    return editor.doc.value ? (props.msgs.metadataReady || 'Ready') : ''
})

function chooseFile() {
    if (editor.api.pick) editor.api.pick().then((path) => path && loadFile(path)).catch(() => {})
    else fileInput.value?.click()
}

function onFileInput(event) {
    const path = event.target?.files?.[0]?.path || ''
    if (path) loadFile(path)
    event.target.value = ''
}

function onDrop(event) {
    dragOver.value = false
    const file = event.dataTransfer?.files?.[0]
    const path = file?.path || ''
    if (path) loadFile(path)
    else chooseFile()
}

function openPicturePicker(index = null) {
    if (!isWritable({ name: 'pictures' }) || editor.saving.value) return
    pictureTargetIndex.value = index
    pictureFileInput.value?.click()
}

function readPictureFile(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader()
        reader.onerror = () => reject(reader.error || new Error('picture read failed'))
        reader.onload = () => resolve(String(reader.result || ''))
        reader.readAsDataURL(file)
    })
}

function pictureFromDataURL(dataURL, file) {
    const match = dataURL.match(/^data:([^;,]+)(?:;[^,]*)?,([\s\S]*)$/)
    if (!match) throw { code: 'picture_data_missing', messageKey: 'errors.pictureDataMissing' }
    const picture = {
        type: 3,
        mime: file?.type || match[1],
        data: match[2],
        bytes: file?.size || 0,
        width: 0,
        height: 0,
    }
    if (typeof Image === 'undefined') return Promise.resolve(picture)
    return new Promise((resolve) => {
        const image = new Image()
        image.onload = () => resolve({ ...picture, width: image.naturalWidth || image.width || 0, height: image.naturalHeight || image.height || 0 })
        image.onerror = () => resolve(picture)
        image.src = dataURL
    })
}

async function onPictureInput(event) {
    const file = event.target?.files?.[0]
    const index = pictureTargetIndex.value
    pictureTargetIndex.value = null
    event.target.value = ''
    if (!file) return
    try {
        const picture = await pictureFromDataURL(await readPictureFile(file), file)
        editor.replacePicture(index, picture)
    } catch (error) {
        editor.error.value = { code: error?.code || 'picture_read_failed', messageKey: error?.messageKey || 'errors.pictureReadFailed', retryable: false }
    }
}

function removePicture(index) {
    editor.removePicture(index)
}

function valueFor(field) {
    if (!editor.doc.value) return ''
    if (field.kind === 'list') {
        if (!(field.name in listDraft.value)) listDraft.value[field.name] = displayValue(field)
        return listDraft.value[field.name]
    }
    if (field.kind === 'numberPair') {
        if (!(field.name in pairDraft.value)) pairDraft.value[field.name] = {
            value: editor.doc.value[field.name]?.value || 0,
            total: editor.doc.value[field.name]?.total || 0,
        }
        return pairDraft.value[field.name]
    }
    return editor.doc.value[field.name]
}

function updateField(field, value) {
    if (field.kind === 'list') listDraft.value[field.name] = value
    if (field.kind === 'numberPair') {
        pairDraft.value[field.name] = { ...pairDraft.value[field.name], ...value }
        value = pairDraft.value[field.name]
    }
    editor.setField(field.name, value)
}

function setMode(field, mode) {
    editor.setMode(field.name, mode)
    if (mode === PATCH_MODES.unchanged) {
        delete listDraft.value[field.name]
        delete pairDraft.value[field.name]
    }
}

async function save() {
    await editor.save()
    listDraft.value = {}
    pairDraft.value = {}
}

function reset() {
    editor.reset()
    listDraft.value = {}
    pairDraft.value = {}
}

async function undo() {
    await editor.undo()
    listDraft.value = {}
    pairDraft.value = {}
}

watch(() => props.initialPath, (path) => {
    if (path && path !== editor.filePath.value) loadFile(path)
})

onMounted(() => {
    if (props.initialPath) loadFile(props.initialPath)
})

function beforeUnload(event) {
    if (!editor.dirty.value) return
    event.preventDefault()
    event.returnValue = ''
}

onMounted(() => {
    window.addEventListener('beforeunload', beforeUnload)
})

onBeforeUnmount(() => {
    window.removeEventListener('beforeunload', beforeUnload)
    emit('dirty-change', false)
})
</script>

<template>
    <div class="content metadata-editor" v-show="activeTab === 'metadata'">
        <div class="metadata-view-tabs no-drag" role="tablist">
            <button class="metadata-view-tab" :class="{ active: mode === 'single' }" type="button" role="tab" :aria-selected="mode === 'single'" @click="mode = 'single'">{{ msgs.metadataSingleMode || 'Single file' }}</button>
            <button class="metadata-view-tab" :class="{ active: mode === 'batch' }" type="button" role="tab" :aria-selected="mode === 'batch'" @click="mode = 'batch'">{{ msgs.metadataBatchMode || 'Batch' }}</button>
        </div>
        <template v-if="mode === 'single'">
        <input ref="fileInput" type="file" class="metadata-file-input" accept=".mp3,.flac,.wav,.aiff,.aif" @change="onFileInput" />
        <input ref="pictureFileInput" type="file" class="metadata-file-input" accept="image/jpeg,image/png,.jpg,.jpeg,.png" @change="onPictureInput" />
        <div class="library-section metadata-toolbar">
            <div>
                <div class="library-section-title">{{ msgs.metadataEditorTitle || 'Metadata editor' }}</div>
                <div class="library-stats metadata-path" :title="editor.filePath">{{ editor.filePath || (msgs.metadataChooseFile || 'Choose an audio file') }}</div>
            </div>
            <button class="btn btn-secondary no-drag" type="button" @click="chooseFile" :disabled="editor.loading || editor.saving">
                {{ msgs.metadataChooseFile || 'Choose file' }}
            </button>
        </div>

        <div v-if="!editor.doc" class="library-section metadata-drop-zone no-drag"
            :class="{ 'drop-active': dragOver }"
            @click="chooseFile"
            @dragover.prevent="dragOver = true"
            @dragleave.prevent="dragOver = false"
            @drop.prevent="onDrop">
            <strong>{{ msgs.metadataDropHint || 'Drop an audio file here' }}</strong>
            <span>{{ msgs.metadataDropSubhint || 'MP3, FLAC, WAV and AIFF are supported' }}</span>
        </div>

        <template v-if="editor.doc">
            <div class="metadata-summary info-card">
                <div class="info-row"><span class="info-label">{{ msgs.metadataFormatLabel || 'Format' }}</span><span class="info-value">{{ editor.doc.format?.toUpperCase() }}</span></div>
                <div class="info-row"><span class="info-label">{{ msgs.metadataTagVersionLabel || 'Tag version' }}</span><span class="info-value">{{ editor.doc.tagVersion || '—' }}</span></div>
                <div v-if="editor.doc.warnings?.length" class="metadata-warning">{{ editor.doc.warnings.length }} {{ msgs.metadataWarnings || 'warnings preserved' }}</div>
            </div>

            <div class="metadata-fields library-section">
                <div v-for="field in METADATA_FIELDS" :key="field.name" class="metadata-field" :class="{ unsupported: !isWritable(field) }">
                    <div class="metadata-field-head">
                        <label :for="`metadata-${field.name}`">{{ fieldLabel(field) }}</label>
                        <div class="metadata-mode no-drag" role="group" :aria-label="`${fieldLabel(field)} mode`">
                            <button v-for="mode in [PATCH_MODES.unchanged, PATCH_MODES.set, PATCH_MODES.clear]" :key="mode" type="button" class="metadata-mode-btn" :class="{ active: editor.modeFor(field.name) === mode }" :disabled="!isWritable(field) || editor.saving" @click="setMode(field, mode)">{{ modeLabel(mode) }}</button>
                        </div>
                    </div>
                    <template v-if="field.kind === 'textarea'">
                        <textarea :id="`metadata-${field.name}`" class="text-input no-drag metadata-input" :value="valueFor(field)" :disabled="!isWritable(field) || editor.modeFor(field.name) === PATCH_MODES.clear || editor.saving" @input="updateField(field, $event.target.value)" />
                    </template>
                    <template v-else-if="field.kind === 'pictures'">
                        <div class="metadata-picture-controls">
                            <button class="btn btn-secondary no-drag" type="button" :disabled="!isWritable(field) || editor.saving" @click="openPicturePicker(null)">
                                {{ msgs.metadataAddPicture || 'Add picture' }}
                            </button>
                            <span class="metadata-picture-count">{{ editor.pictures.length }} {{ msgs.metadataPictureCount || 'embedded' }}</span>
                        </div>
                        <div v-if="editor.pictures.length" class="metadata-picture-list">
                            <div v-for="(picture, index) in editor.pictures" :key="picture.id || `picture-${index}`" class="metadata-picture-row">
                                <img v-if="pictureDataURI(picture)" :src="pictureDataURI(picture)" :alt="picture.mime || 'cover'" />
                                <span v-else class="metadata-picture-missing">{{ msgs.metadataPicturePayloadMissing || 'Image payload unavailable' }}</span>
                                <span class="metadata-picture-meta">{{ picture.mime || 'image' }} · {{ picture.width || '—' }}×{{ picture.height || '—' }}</span>
                                <button class="btn btn-secondary no-drag" type="button" :disabled="!isWritable(field) || editor.saving" @click="openPicturePicker(index)">{{ msgs.metadataReplacePicture || 'Replace' }}</button>
                                <button class="btn btn-secondary no-drag" type="button" :disabled="!isWritable(field) || editor.saving" @click="removePicture(index)">{{ msgs.metadataRemovePicture || 'Remove' }}</button>
                            </div>
                        </div>
                    </template>
                    <template v-else-if="field.kind === 'numberPair'">
                        <div class="metadata-pair">
                            <input class="text-input no-drag metadata-input" type="number" min="0" :value="valueFor(field).value" :disabled="!isWritable(field) || editor.modeFor(field.name) === PATCH_MODES.clear || editor.saving" @input="updateField(field, { value: $event.target.value })" />
                            <span>/</span>
                            <input class="text-input no-drag metadata-input" type="number" min="0" :value="valueFor(field).total || ''" :disabled="!isWritable(field) || editor.modeFor(field.name) === PATCH_MODES.clear || editor.saving" @input="updateField(field, { total: $event.target.value })" />
                        </div>
                    </template>
                    <input v-else class="text-input no-drag metadata-input" :type="field.kind === 'number' ? 'number' : 'text'" :id="`metadata-${field.name}`" :value="valueFor(field)" :disabled="!isWritable(field) || editor.modeFor(field.name) === PATCH_MODES.clear || editor.saving" @input="updateField(field, $event.target.value)" />
                    <small v-if="!isWritable(field)" class="metadata-unsupported">{{ msgs.metadataUnsupported || 'Not supported by this format' }}</small>
                </div>
            </div>

            <div class="metadata-actions library-section">
                <button class="btn btn-primary no-drag" type="button" :disabled="!editor.dirty || editor.saving || editor.loading" @click="save">{{ msgs.metadataSave || 'Save changes' }}</button>
                <button class="btn btn-secondary no-drag" type="button" :disabled="!editor.dirty || editor.saving" @click="reset">{{ msgs.metadataReset || 'Reset' }}</button>
                <button class="btn btn-secondary no-drag" type="button" :disabled="!editor.canUndo || editor.undoing || editor.saving" @click="undo">{{ msgs.metadataUndo || 'Undo last save' }}</button>
                <span class="metadata-status" :class="{ error: editor.error }" role="status">{{ statusText }}</span>
            </div>

        </template>
        </template>
        <BatchEditor v-else :msgs="msgs" @dirty-change="emit('dirty-change', $event)" />
    </div>
</template>

<style scoped>
.metadata-editor { gap: 10px; }
.metadata-view-tabs { display: inline-flex; align-self: flex-start; }
.metadata-view-tab { border: 1px solid var(--border); background: transparent; color: var(--text-dim); padding: 5px 12px; cursor: pointer; font-size: 12px; }
.metadata-view-tab:first-child { border-radius: 4px 0 0 4px; }
.metadata-view-tab:last-child { border-radius: 0 4px 4px 0; border-left: 0; }
.metadata-view-tab.active { color: var(--text); background: var(--bg-card-hover); border-color: var(--accent); }
.metadata-file-input { display: none; }
.metadata-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.metadata-path { max-width: 62vw; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.metadata-drop-zone { min-height: 130px; align-items: center; justify-content: center; text-align: center; border: 1px dashed var(--border); cursor: pointer; }
.metadata-drop-zone span { color: var(--text-dim); font-size: 12px; }
.metadata-drop-zone.drop-active { border-color: var(--accent); background: var(--bg-card-hover); }
.metadata-summary { gap: 7px; }
.metadata-warning { color: var(--warning); font-size: 12px; }
.metadata-fields { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.metadata-field { min-width: 0; }
.metadata-field-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; margin-bottom: 5px; }
.metadata-field-head label { color: var(--text-dim); font-size: 12px; }
.metadata-mode { display: inline-flex; }
.metadata-mode-btn { border: 1px solid var(--border); background: transparent; color: var(--text-dim); font-size: 10px; padding: 2px 5px; cursor: pointer; }
.metadata-mode-btn:first-child { border-radius: 4px 0 0 4px; }
.metadata-mode-btn:last-child { border-radius: 0 4px 4px 0; }
.metadata-mode-btn + .metadata-mode-btn { border-left: 0; }
.metadata-mode-btn.active { color: var(--text); background: var(--bg-card-hover); border-color: var(--accent); }
.metadata-input { width: 100%; min-height: 32px; }
textarea.metadata-input { min-height: 72px; resize: vertical; }
.metadata-picture-controls { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.metadata-picture-count { color: var(--text-dim); font-size: 12px; }
.metadata-pair { display: flex; align-items: center; gap: 6px; }
.metadata-pair .metadata-input { min-width: 0; }
.metadata-unsupported { color: var(--text-muted); }
.metadata-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.metadata-status { color: var(--text-dim); font-size: 12px; margin-left: auto; }
.metadata-status.error { color: var(--error); }
.metadata-picture-list { display: grid; gap: 6px; }
.metadata-picture-row { display: flex; align-items: center; gap: 8px; color: var(--text-dim); font-size: 11px; flex-wrap: wrap; }
.metadata-picture-row img { width: 42px; height: 42px; object-fit: cover; border-radius: 4px; background: var(--bg-card-hover); }
.metadata-picture-meta { min-width: 110px; }
.metadata-picture-missing { color: var(--warning); }
@media (max-width: 640px) {
    .metadata-fields { grid-template-columns: 1fr; }
    .metadata-toolbar { align-items: flex-start; flex-direction: column; }
    .metadata-path { max-width: 90vw; }
    .metadata-status { width: 100%; margin-left: 0; }
}
</style>
