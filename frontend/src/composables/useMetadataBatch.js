import { computed, ref } from 'vue'
import * as generatedApi from '../../wailsjs/go/main/App.js'
import { METADATA_FIELDS, PATCH_MODES } from './useMetadataEditor.js'

const LIST_FIELDS = new Set(['artists', 'albumArtists', 'genres', 'composers'])
const NUMBER_PAIR_FIELDS = new Set(['track', 'disc'])
const NUMBER_FIELDS = new Set(['bpm', 'rating'])

function emptyValue(name) {
    if (name === 'pictures') return []
    if (LIST_FIELDS.has(name)) return []
    if (NUMBER_PAIR_FIELDS.has(name)) return { value: 0, total: 0 }
    if (NUMBER_FIELDS.has(name)) return 0
    return ''
}

function splitList(value) {
    if (Array.isArray(value)) return value.map((item) => String(item).trim()).filter(Boolean)
    if (value === undefined || value === null || value === '') return []
    return String(value).split(/[;,\n]/).map((item) => item.trim()).filter(Boolean)
}

function normalizeValue(name, value) {
    if (name === 'pictures') return Array.isArray(value) ? value : []
    if (!LIST_FIELDS.has(name) && !NUMBER_PAIR_FIELDS.has(name) && !NUMBER_FIELDS.has(name)) {
        return String(value ?? '')
    }
    if (LIST_FIELDS.has(name)) return splitList(value)
    if (NUMBER_FIELDS.has(name)) {
        const number = Number(value)
        return Number.isFinite(number) ? number : 0
    }
    const text = String(value ?? '')
    const [first, second] = text.split('/')
    return { value: Number(first) || 0, total: Number(second) || 0 }
}

function method(name) {
    const app = typeof window !== 'undefined' ? window?.go?.main?.App : null
    if (app && typeof app[name] === 'function') return app[name].bind(app)
    if (typeof generatedApi[name] === 'function') return generatedApi[name]
    return null
}

function defaultApi() {
    return {
        pickDirectory: method('MetadataPickDirectory'),
        scanDirectory: method('MetadataScanDirectory'),
        preview: method('MetadataBatchPreview'),
        start: method('MetadataBatchStart'),
        result: method('MetadataBatchResult'),
        cancel: method('MetadataCancel'),
    }
}

function publicError(error, fallbackCode = 'batch_failed') {
    let source = error
    if (typeof source === 'string') {
        try { source = JSON.parse(source) } catch (_) { source = null }
    }
    const code = source?.code || source?.Code || fallbackCode
    return {
        code: String(code),
        messageKey: String(source?.messageKey || source?.MessageKey || `errors.${code}`),
        retryable: Boolean(source?.retryable ?? source?.Retryable),
    }
}

function initialState() {
    return Object.fromEntries(METADATA_FIELDS.map(({ name }) => [name, PATCH_MODES.unchanged]))
}

function isTerminal(result) {
    if (!result) return false
    const total = Number(result.total || 0)
    if (total === 0) return true
    return Number(result.updated || 0) + Number(result.skipped || 0) +
        Number(result.failed || 0) + Number(result.blocked || 0) >= total
}

export function createMetadataBatch(options = {}) {
    const api = options.api || defaultApi()
    const pollInterval = options.pollInterval || 250
    const paths = ref([])
    const modes = ref(initialState())
    const values = ref(Object.fromEntries(METADATA_FIELDS.map(({ name }) => [name, emptyValue(name)])))
    const preview = ref(null)
    const result = ref(null)
    const operationId = ref('')
    const busy = ref(false)
    const polling = ref(false)
    const error = ref(null)
    let timer = null

    const dirty = computed(() => Object.values(modes.value).some((mode) => mode !== PATCH_MODES.unchanged))
    const progress = computed(() => {
        const current = result.value
        if (!current) return { processed: 0, total: 0, percent: 0 }
        const processed = Number(current.updated || 0) + Number(current.skipped || 0) +
            Number(current.failed || 0) + Number(current.blocked || 0)
        const total = Number(current.total || 0)
        return { processed, total, percent: total > 0 ? Math.min(100, Math.round((processed / total) * 100)) : 0 }
    })

    function addPaths(list) {
        const next = [...paths.value]
        const seen = new Set(next.map((path) => path.toLowerCase()))
        for (const raw of list || []) {
            const path = String(raw || '').trim()
            if (!path || seen.has(path.toLowerCase())) continue
            seen.add(path.toLowerCase())
            next.push(path)
        }
        paths.value = next
        preview.value = null
        result.value = null
        error.value = null
    }

    function removePath(path) {
        paths.value = paths.value.filter((item) => item !== path)
        preview.value = null
        result.value = null
    }

    function clearPaths() {
        paths.value = []
        preview.value = null
        result.value = null
    }

    function resetModes() {
        modes.value = initialState()
        values.value = Object.fromEntries(METADATA_FIELDS.map(({ name }) => [name, emptyValue(name)]))
        preview.value = null
        result.value = null
        error.value = null
    }

    function setMode(name, mode) {
        if (!Object.prototype.hasOwnProperty.call(modes.value, name)) return
        if (!Object.values(PATCH_MODES).includes(mode)) return
        modes.value = { ...modes.value, [name]: mode }
        preview.value = null
        error.value = null
    }

    function setValue(name, value) {
        if (!Object.prototype.hasOwnProperty.call(values.value, name)) return
        values.value = { ...values.value, [name]: value }
        preview.value = null
        error.value = null
    }

    function buildPatch() {
        const patch = {}
        for (const field of METADATA_FIELDS) {
            const mode = modes.value[field.name] || PATCH_MODES.unchanged
            patch[field.name] = {
                mode,
                value: mode === PATCH_MODES.set
                    ? normalizeValue(field.name, values.value[field.name])
                    : emptyValue(field.name),
            }
        }
        return patch
    }

    async function chooseDirectory() {
        if (!api.pickDirectory || !api.scanDirectory) {
            error.value = publicError({ code: 'metadata_unavailable', messageKey: 'errors.metadataUnavailable' }, 'metadata_unavailable')
            return []
        }
        try {
            const directory = await api.pickDirectory()
            if (!directory) return []
            const files = await api.scanDirectory(directory)
            addPaths(files || [])
            return files || []
        } catch (e) {
            error.value = publicError(e, 'scan_failed')
            return []
        }
    }

    async function runPreview() {
        if (!api.preview || !dirty.value || paths.value.length === 0) return null
        busy.value = true
        error.value = null
        try {
            preview.value = await api.preview(paths.value, buildPatch())
            return preview.value
        } catch (e) {
            error.value = publicError(e, 'preview_failed')
            return null
        } finally {
            busy.value = false
        }
    }

    function stopPolling() {
        if (timer !== null) {
            clearTimeout(timer)
            timer = null
        }
        polling.value = false
    }

    async function poll() {
        if (!operationId.value || !api.result) return
        try {
            const current = await api.result(operationId.value)
            result.value = current
            if (isTerminal(current)) {
                stopPolling()
                busy.value = false
                return
            }
        } catch (e) {
            error.value = publicError(e, 'batch_result_failed')
            stopPolling()
            busy.value = false
            return
        }
        timer = setTimeout(poll, pollInterval)
    }

    async function start() {
        if (!api.start || !dirty.value || paths.value.length === 0) return ''
        busy.value = true
        error.value = null
        preview.value = null
        result.value = null
        operationId.value = ''
        try {
            operationId.value = await api.start(paths.value, buildPatch())
            if (!operationId.value) throw { code: 'batch_start_failed', messageKey: 'errors.batchStartFailed' }
            polling.value = true
            timer = setTimeout(poll, pollInterval)
            return operationId.value
        } catch (e) {
            error.value = publicError(e, 'batch_start_failed')
            busy.value = false
            return ''
        }
    }

    async function cancel() {
        if (!operationId.value || !api.cancel) return false
        try {
            return await api.cancel(operationId.value)
        } catch (e) {
            error.value = publicError(e, 'batch_cancel_failed')
            return false
        }
    }

    return {
        api,
        paths,
        modes,
        values,
        preview,
        result,
        operationId,
        busy,
        polling,
        error,
        dirty,
        progress,
        addPaths,
        removePath,
        clearPaths,
        resetModes,
        setMode,
        setValue,
        buildPatch,
        chooseDirectory,
        runPreview,
        start,
        cancel,
        stopPolling,
    }
}
