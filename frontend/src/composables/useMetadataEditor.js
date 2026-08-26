import { ref, computed } from 'vue'
import { ReadTag } from '../wailsjs/go/main/App.js'

// createMetadataEditor manages one file's metadata editing session.
export function createMetadataEditor(filePath) {
    const doc = ref(null)
    const dirty = ref(false)
    const saving = ref(false)
    const error = ref(null)
    const original = ref(null)

    async function load() {
        try {
            error.value = null
            const result = await ReadTag(filePath)
            doc.value = result
            original.value = JSON.parse(JSON.stringify(result))
            dirty.value = false
        } catch (e) {
            error.value = { code: 'read_failed', message: String(e) }
        }
    }

    function setField(name, value) {
        if (doc.value) {
            doc.value[name] = value
            dirty.value = true
        }
    }

    function reset() {
        if (original.value) {
            doc.value = JSON.parse(JSON.stringify(original.value))
            dirty.value = false
        }
    }

    return { doc, dirty, saving, error, load, setField, reset,
             operationId: filePath }
}
