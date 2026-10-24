import { computed, ref } from 'vue'
// The generated bindings live beside `src`, so composables need to traverse
// two levels (`src/composables` -> `frontend/wailsjs`).
import * as generatedApi from '../../wailsjs/go/main/App.js'

// Keep displayed values separate from patch intent: an empty value can mean
// either "leave unchanged" or "clear", depending on the selected mode.
export const PATCH_MODES = Object.freeze({
    unchanged: 'unchanged',
    set: 'set',
    clear: 'clear',
})

export const METADATA_FIELDS = Object.freeze([
    { name: 'title', label: 'Title', kind: 'text' },
    { name: 'artists', label: 'Artist', kind: 'list' },
    { name: 'albumArtists', label: 'Album artist', kind: 'list' },
    { name: 'album', label: 'Album', kind: 'text' },
    { name: 'track', label: 'Track', kind: 'numberPair' },
    { name: 'disc', label: 'Disc', kind: 'numberPair' },
    { name: 'date', label: 'Date', kind: 'text' },
    { name: 'genres', label: 'Genre', kind: 'list' },
    { name: 'composers', label: 'Composer', kind: 'list' },
    { name: 'comment', label: 'Comment', kind: 'text' },
    { name: 'bpm', label: 'BPM', kind: 'number' },
    { name: 'key', label: 'Key', kind: 'text' },
    { name: 'rating', label: 'Rating', kind: 'number' },
    { name: 'isrc', label: 'ISRC', kind: 'text' },
    { name: 'lyrics', label: 'Lyrics', kind: 'textarea' },
    { name: 'pictures', label: 'Pictures', kind: 'pictures' },
])

const LIST_FIELDS = new Set(['artists', 'albumArtists', 'genres', 'composers'])
const NUMBER_PAIR_FIELDS = new Set(['track', 'disc'])
let pictureIDSequence = 0

const clone = (value) => {
    if (value === undefined || value === null) return value
    return JSON.parse(JSON.stringify(value))
}

function bytesToBase64(value) {
    if (!value) return ''
    if (typeof value === 'string') return value
    const bytes = value instanceof Uint8Array ? value : Uint8Array.from(value)
    let binary = ''
    for (let i = 0; i < bytes.length; i += 1) binary += String.fromCharCode(bytes[i])
    if (typeof btoa === 'function') return btoa(binary)
    if (globalThis.Buffer) return globalThis.Buffer.from(bytes).toString('base64')
    return ''
}

function normalizePicture(raw) {
    const source = raw && typeof raw === 'object' ? raw : {}
    let mime = source.mime || source.MIME || ''
    let data = source.data ?? source.Data ?? ''
    let thumbnailMime = source.thumbnailMime || source.ThumbnailMIME || ''
    let thumbnail = source.thumbnail ?? source.Thumbnail ?? ''
    if (typeof data === 'string' && data.startsWith('data:')) {
        const match = data.match(/^data:([^;,]+)(?:;[^,]*)?,([\s\S]*)$/)
        if (match) {
            mime = mime || match[1]
            data = match[2]
        }
    }
    if (typeof thumbnail === 'string' && thumbnail.startsWith('data:')) {
        const match = thumbnail.match(/^data:([^;,]+)(?:;[^,]*)?,([\s\S]*)$/)
        if (match) {
            thumbnailMime = thumbnailMime || match[1]
            thumbnail = match[2]
        }
    }
    if (typeof data !== 'string') data = bytesToBase64(data)
    if (typeof thumbnail !== 'string') thumbnail = bytesToBase64(thumbnail)
    return {
        id: String(source.id ?? source.ID ?? ''),
        type: Number(source.type ?? source.Type ?? 0) || 0,
        mime: String(mime || ''),
        description: String(source.description ?? source.Description ?? ''),
        bytes: Number(source.bytes ?? source.Bytes ?? 0) || (data ? Math.floor(data.length * 3 / 4) : 0),
        width: Number(source.width ?? source.Width ?? 0) || 0,
        height: Number(source.height ?? source.Height ?? 0) || 0,
        data: String(data || '').replace(/\s/g, ''),
        thumbnailMime: String(thumbnailMime || ''),
        thumbnail: String(thumbnail || '').replace(/\s/g, ''),
    }
}

function normalizePictures(value) {
    return Array.isArray(value) ? value.map(normalizePicture) : []
}

function mergePicturePayloads(primary, fallback) {
    const fallbackList = normalizePictures(fallback)
    const primaryList = normalizePictures(primary)
    // MetadataPictures is a secondary payload call. During startup it may
    // transiently return an empty list even though MetadataRead already
    // supplied descriptors (and possibly bytes). Never let that transient
    // response erase the document's picture set.
    if (primaryList.length === 0 && fallbackList.length > 0) return fallbackList
    const byID = new Map(fallbackList.filter((picture) => picture.id).map((picture) => [picture.id, picture]))
    return primaryList.map((picture, index) => {
        if (picture.data && picture.thumbnail) return picture
        const match = (picture.id && byID.get(picture.id)) || fallbackList[index]
        if (!match) return picture
        return {
            ...picture,
            data: picture.data || match.data,
            mime: picture.mime || match.mime,
            width: picture.width || match.width,
            height: picture.height || match.height,
            thumbnail: picture.thumbnail || match.thumbnail,
            thumbnailMime: picture.thumbnailMime || match.thumbnailMime,
        }
    })
}

function newPictureID(existing) {
    const used = new Set(normalizePictures(existing).map((picture) => picture.id).filter(Boolean))
    const randomUUID = globalThis.crypto?.randomUUID
    for (;;) {
        const candidate = typeof randomUUID === 'function'
            ? `picture-${randomUUID.call(globalThis.crypto)}`
            : `picture-${Date.now().toString(36)}-${(++pictureIDSequence).toString(36)}`
        if (!used.has(candidate)) return candidate
    }
}

export function pictureDataURI(picture) {
    const normalized = normalizePicture(picture)
    if (normalized.data) {
        const mime = normalized.mime || 'application/octet-stream'
        return `data:${mime};base64,${normalized.data}`
    }
    if (!normalized.thumbnail) return ''
    const mime = normalized.thumbnailMime || 'image/jpeg'
    return `data:${mime};base64,${normalized.thumbnail}`
}

function picturePayloadMissing(pictures) {
    return normalizePictures(pictures).find((picture) => !picture.data)
}

function equalValue(a, b) {
    return JSON.stringify(a ?? null) === JSON.stringify(b ?? null)
}

function splitList(value) {
    if (Array.isArray(value)) return value.map((item) => String(item).trim()).filter(Boolean)
    if (value === undefined || value === null || value === '') return []
    return String(value).split(/[;,\n]/).map((item) => item.trim()).filter(Boolean)
}

function normalizeDocument(raw, filePath = '') {
    const source = raw && typeof raw === 'object' ? raw : {}
    const result = { ...source }
    for (const field of LIST_FIELDS) result[field] = splitList(source[field])
    for (const field of NUMBER_PAIR_FIELDS) {
        const pair = source[field]
        if (pair && typeof pair === 'object') {
            result[field] = { value: Number(pair.value || 0), total: Number(pair.total || 0) }
        } else {
            result[field] = { value: Number(pair || 0), total: 0 }
        }
    }
    for (const field of ['title', 'album', 'date', 'comment', 'key', 'isrc', 'lyrics']) {
        result[field] = result[field] == null ? '' : String(result[field])
    }
    for (const field of ['bpm', 'rating']) {
        const n = Number(result[field])
        result[field] = Number.isFinite(n) ? n : 0
    }
    if (!result.format && filePath) {
        const ext = filePath.split(/[\\/.]/).pop()?.toLowerCase()
        result.format = ext === 'aif' ? 'aiff' : ext || ''
    }
    result.warnings = Array.isArray(result.warnings) ? result.warnings : []
    result.pictures = normalizePictures(result.pictures)
    return result
}

function publicError(error, fallbackCode = 'internal') {
    // Wails may reject with JSON, a public error object, or Error. Keep only
    // stable identifiers so native paths/details never reach the view.
    let source = error
    if (typeof source === 'string') {
        try { source = JSON.parse(source) } catch (_) { source = null }
    }
    const code = source?.code || source?.Code || fallbackCode
    const messageKey = source?.messageKey || source?.MessageKey || `errors.${code}`
    return {
        code: String(code),
        messageKey: String(messageKey),
        retryable: Boolean(source?.retryable ?? source?.Retryable),
    }
}

function method(name) {
    const app = typeof window !== 'undefined' ? window?.go?.main?.App : null
    // Prefer a live Wails binding. This also lets older installations fall
    // back to the legacy ID3 methods when the new binding is not exported yet.
    if (app && typeof app[name] === 'function') return app[name].bind(app)
    if (app && typeof generatedApi[name] === 'function') return generatedApi[name]
    return null
}

function legacyDocument(raw, filePath) {
    return normalizeDocument({
        ...raw,
        artists: raw?.artists || raw?.artist,
        genres: raw?.genres || raw?.genre,
        date: raw?.date || raw?.year,
    }, filePath)
}

function createDefaultApi() {
    const readMethod = method('MetadataRead')
    const applyMethod = method('MetadataApply')
    const picturesMethod = method('MetadataPictures')
    const undoMethod = method('MetadataUndo')
    const pickMethod = method('MetadataPickFile') || method('ID3PickFile')
    const legacyRead = method('ID3ReadTag')
    const legacyWrite = method('ID3WriteTag')
    const legacyCover = method('ID3GetCover')

    return {
        pick: pickMethod ? () => pickMethod() : null,
        read: readMethod
            ? (path) => readMethod(path)
            : legacyRead
                ? async (path) => legacyDocument(await legacyRead(path), path)
                : null,
        apply: applyMethod
            ? (path, patch) => applyMethod(path, patch)
            : legacyWrite
                ? async (path, patch) => {
                    const current = legacyDocument(legacyRead ? await legacyRead(path) : {}, path)
                    const value = (name, fallback = '') => {
                        const item = patch[name]
                        if (!item || item.mode === PATCH_MODES.unchanged) return current[name] ?? fallback
                        if (item.mode === PATCH_MODES.clear) return fallback
                        return item.value ?? fallback
                    }
                    const unsupported = ['albumArtists', 'track', 'disc', 'bpm', 'key', 'rating', 'isrc', 'lyrics', 'composers', 'pictures']
                    if (unsupported.some((name) => patch[name]?.mode && patch[name].mode !== PATCH_MODES.unchanged)) {
                        throw { code: 'unsupported_field', messageKey: 'errors.unsupportedField' }
                    }
                    const response = await legacyWrite(
                        path,
                        value('title'),
                        splitList(value('artists')).join('; '),
                        value('album'),
                        value('date'),
                        splitList(value('genres')).join('; '),
                    )
                    if (response && response !== 'ok') throw { code: 'apply_failed', messageKey: 'errors.apply_failed' }
                    return { ok: true, changed: 1 }
                }
                : null,
        pictures: picturesMethod
            ? (path) => picturesMethod(path)
            : legacyCover
                ? async (path) => {
                    const data = await legacyCover(path)
                    return data ? [{ id: 'legacy-front', type: 3, mime: 'image/jpeg', data }] : []
                }
                : async () => [],
        undo: undoMethod ? (...args) => undoMethod(...args) : null,
    }
}

function emptyValue(field) {
    if (LIST_FIELDS.has(field)) return []
    if (NUMBER_PAIR_FIELDS.has(field)) return { value: 0, total: 0 }
    if (field === 'pictures') return []
    if (field === 'bpm' || field === 'rating') return 0
    return ''
}

export function createMetadataEditor(initialPath = '', options = {}) {
    const api = options.api || createDefaultApi()
    const filePath = ref(initialPath || '')
    const doc = ref(null)
    const original = ref(null)
    const modes = ref(Object.fromEntries(METADATA_FIELDS.map(({ name }) => [name, PATCH_MODES.unchanged])))
    const loading = ref(false)
    const saving = ref(false)
    const undoing = ref(false)
    const error = ref(null)
    const result = ref(null)
    const pictures = ref([])

    const dirty = computed(() => Object.values(modes.value).some((mode) => mode !== PATCH_MODES.unchanged))
    const hasDocument = computed(() => Boolean(doc.value))
    const canUndo = computed(() => Boolean(api.undo && filePath.value && result.value?.snapshotId))

    function setPath(path) {
        filePath.value = path || ''
    }

    function modeFor(name) {
        return modes.value[name] || PATCH_MODES.unchanged
    }

    function setMode(name, mode) {
        if (!Object.prototype.hasOwnProperty.call(modes.value, name)) return
        if (!Object.values(PATCH_MODES).includes(mode)) return
        modes.value = { ...modes.value, [name]: mode }
        if (mode === PATCH_MODES.clear && doc.value) {
            doc.value = { ...doc.value, [name]: emptyValue(name) }
            if (name === 'pictures') pictures.value = []
        } else if (mode === PATCH_MODES.unchanged && doc.value && original.value) {
            doc.value = { ...doc.value, [name]: clone(original.value[name]) ?? emptyValue(name) }
            if (name === 'pictures') pictures.value = normalizePictures(original.value[name])
        } else if (mode === PATCH_MODES.set && doc.value && original.value && equalValue(doc.value[name], emptyValue(name))) {
            doc.value = { ...doc.value, [name]: clone(original.value[name]) ?? emptyValue(name) }
            if (name === 'pictures') pictures.value = normalizePictures(original.value[name])
        }
        error.value = null
    }

    function setField(name, value) {
        if (!doc.value || !Object.prototype.hasOwnProperty.call(modes.value, name)) return
        let normalized = value
        if (LIST_FIELDS.has(name)) normalized = splitList(value)
        if (NUMBER_PAIR_FIELDS.has(name)) {
            normalized = {
                value: Math.max(0, Number(value?.value ?? value) || 0),
                total: Math.max(0, Number(value?.total) || 0),
            }
        }
        if (name === 'pictures') normalized = normalizePictures(value)
        if (name === 'bpm' || name === 'rating') normalized = Number(value) || 0
        doc.value = { ...doc.value, [name]: normalized }
        if (name === 'pictures') pictures.value = clone(normalized)
        const nextMode = original.value && equalValue(normalized, original.value[name])
            ? PATCH_MODES.unchanged
            : PATCH_MODES.set
        modes.value = { ...modes.value, [name]: nextMode }
        error.value = null
    }

    function buildPatch() {
        const patch = {}
        for (const field of METADATA_FIELDS) {
            const mode = modeFor(field.name)
            const value = field.name === 'pictures' ? pictures.value : doc.value?.[field.name]
            patch[field.name] = {
                mode,
                value: mode === PATCH_MODES.set
                    ? clone(value) ?? emptyValue(field.name)
                    : emptyValue(field.name),
            }
        }
        return patch
    }

    async function load(path = filePath.value) {
        setPath(path)
        error.value = null
        result.value = null
        if (!filePath.value) {
            doc.value = null
            original.value = null
            pictures.value = []
            return null
        }
        if (!api.read) {
            error.value = publicError({ code: 'metadata_unavailable', messageKey: 'errors.metadataUnavailable' }, 'metadata_unavailable')
            return null
        }
        loading.value = true
        try {
            const loaded = normalizeDocument(await api.read(filePath.value), filePath.value)
            let pictureResponse = loaded.pictures
            if (api.pictures) {
                try {
                    pictureResponse = await api.pictures(filePath.value)
                } catch (_) {
                    pictureResponse = loaded.pictures
                }
            }
            loaded.pictures = mergePicturePayloads(pictureResponse, loaded.pictures)
            doc.value = loaded
            original.value = clone(loaded)
            modes.value = Object.fromEntries(METADATA_FIELDS.map(({ name }) => [name, PATCH_MODES.unchanged]))
            pictures.value = normalizePictures(loaded.pictures)
            return loaded
        } catch (e) {
            doc.value = null
            original.value = null
            pictures.value = []
            error.value = publicError(e, 'read_failed')
            return null
        } finally {
            loading.value = false
        }
    }

    function reset() {
        if (!original.value) return
        doc.value = clone(original.value)
        pictures.value = normalizePictures(original.value.pictures)
        modes.value = Object.fromEntries(METADATA_FIELDS.map(({ name }) => [name, PATCH_MODES.unchanged]))
        error.value = null
        result.value = null
    }

    async function save() {
        if (!doc.value || !filePath.value || !dirty.value) return null
        if (!api.apply) {
            error.value = publicError({ code: 'metadata_unavailable', messageKey: 'errors.metadataUnavailable' }, 'metadata_unavailable')
            return null
        }
        saving.value = true
        error.value = null
        try {
            const response = await api.apply(filePath.value, buildPatch())
            if (response?.publicError) throw response.publicError
            if (response?.error && typeof response.error !== 'object') throw { code: 'apply_failed', messageKey: 'errors.apply_failed' }
            const loaded = await api.read(filePath.value)
            const normalized = normalizeDocument(loaded, filePath.value)
            let pictureResponse = normalized.pictures
            if (api.pictures) {
                try {
                    pictureResponse = await api.pictures(filePath.value)
                } catch (_) {
                    pictureResponse = normalized.pictures
                }
            }
            normalized.pictures = mergePicturePayloads(pictureResponse, normalized.pictures)
            doc.value = normalized
            original.value = clone(normalized)
            modes.value = Object.fromEntries(METADATA_FIELDS.map(({ name }) => [name, PATCH_MODES.unchanged]))
            result.value = response || { ok: true }
            pictures.value = normalizePictures(normalized.pictures)
            return result.value
        } catch (e) {
            error.value = publicError(e, 'apply_failed')
            return null
        } finally {
            saving.value = false
        }
    }

    async function undo() {
        if (!canUndo.value) return null
        undoing.value = true
        error.value = null
        try {
            const response = await api.undo(filePath.value, result.value.snapshotId)
            if (response?.publicError) throw response.publicError
            await load(filePath.value)
            result.value = response || { ok: true }
            return result.value
        } catch (e) {
            error.value = publicError(e, 'undo_failed')
            return null
        } finally {
            undoing.value = false
        }
    }

    function replacePicture(index, picture) {
        const next = normalizePictures(pictures.value)
        const normalized = normalizePicture(picture)
        if (index === null || index === undefined || index < 0 || index >= next.length) {
            if (!normalized.id) normalized.id = newPictureID(next)
            next.push(normalized)
        }
        else {
            const existing = next[index]
            const id = existing.id || normalized.id || newPictureID(next)
            next[index] = { ...existing, ...normalized, id }
        }
        setField('pictures', next)
    }

    function removePicture(index) {
        const next = normalizePictures(pictures.value)
        if (index < 0 || index >= next.length) return
        next.splice(index, 1)
        if (next.length === 0) {
            pictures.value = []
            if (doc.value) doc.value = { ...doc.value, pictures: [] }
            modes.value = { ...modes.value, pictures: PATCH_MODES.clear }
            error.value = null
            return
        }
        setField('pictures', next)
    }

    return {
        api,
        filePath,
        doc,
        original,
        modes,
        pictures,
        loading,
        saving,
        undoing,
        error,
        result,
        dirty,
        hasDocument,
        canUndo,
        setPath,
        modeFor,
        setMode,
        setField,
        replacePicture,
        removePicture,
        buildPatch,
        load,
        save,
        undo,
        reset,
    }
}
