import { describe, expect, it, vi } from 'vitest'
import { createMetadataBatch } from './useMetadataBatch'
import { PATCH_MODES } from './useMetadataEditor'

function makeApi(overrides = {}) {
    return {
        pickDirectory: vi.fn(async () => '/music'),
        scanDirectory: vi.fn(async () => ['/music/a.flac', '/music/b.flac']),
        preview: vi.fn(async (_paths, _patch) => ({
            total: 2, eligible: 1, blocked: 1,
            items: [{ path: '/music/a.flac', status: 'eligible' }, { path: '/music/b.flac', status: 'blocked', code: 'unsupported_field' }],
        })),
        start: vi.fn(async () => 'op-1'),
        result: vi.fn(async () => ({
            operationId: 'op-1', total: 2, updated: 1, skipped: 0, failed: 1, blocked: 0, cancelled: false,
            items: [{ path: '/music/a.flac', status: 'updated' }, { path: '/music/b.flac', status: 'failed', code: 'apply_failed' }],
        })),
        cancel: vi.fn(async () => true),
        ...overrides,
    }
}

describe('createMetadataBatch', () => {
    it('adds directory files and de-duplicates paths', async () => {
        const api = makeApi()
        const batch = createMetadataBatch({ api })
        await batch.chooseDirectory()
        expect(batch.paths.value).toEqual(['/music/a.flac', '/music/b.flac'])
        batch.addPaths(['/music/a.flac', '/music/c.flac'])
        expect(batch.paths.value).toEqual(['/music/a.flac', '/music/b.flac', '/music/c.flac'])
    })

    it('builds explicit tri-state patches with parsed lists and pairs', () => {
        const batch = createMetadataBatch({ api: makeApi() })
        batch.setMode('title', PATCH_MODES.set)
        batch.setValue('title', 'New title')
        batch.setMode('artists', PATCH_MODES.set)
        batch.setValue('artists', 'A; B')
        batch.setMode('track', PATCH_MODES.set)
        batch.setValue('track', '2/12')
        batch.setMode('album', PATCH_MODES.clear)
        const patch = batch.buildPatch()
        expect(patch.title).toEqual({ mode: 'set', value: 'New title' })
        expect(patch.artists).toEqual({ mode: 'set', value: ['A', 'B'] })
        expect(patch.track).toEqual({ mode: 'set', value: { value: 2, total: 12 } })
        expect(patch.album).toEqual({ mode: 'clear', value: '' })
        expect(patch.comment).toEqual({ mode: 'unchanged', value: '' })
        expect(patch.pictures).toEqual({ mode: 'unchanged', value: [] })
    })

    it('previews and starts a batch, then polls to a terminal result', async () => {
        const api = makeApi()
        const batch = createMetadataBatch({ api, pollInterval: 5 })
        batch.addPaths(['/music/a.flac', '/music/b.flac'])
        batch.setMode('title', PATCH_MODES.set)
        batch.setValue('title', 'T')
        const preview = await batch.runPreview()
        expect(preview.eligible).toBe(1)
        expect(api.preview).toHaveBeenCalledOnce()

        const opID = await batch.start()
        expect(opID).toBe('op-1')
        await vi.waitFor(() => expect(batch.polling.value).toBe(false))
        expect(batch.result.value.failed).toBe(1)
        expect(batch.progress.value).toEqual({ processed: 2, total: 2, percent: 100 })
    })

    it('cancels a running operation', async () => {
        const api = makeApi({
            result: vi.fn(async () => ({ operationId: 'op-1', total: 2, updated: 0, skipped: 0, failed: 0, blocked: 0, cancelled: false, items: [] })),
        })
        const batch = createMetadataBatch({ api, pollInterval: 50 })
        batch.addPaths(['/music/a.flac', '/music/b.flac'])
        batch.setMode('title', PATCH_MODES.set)
        batch.setValue('title', 'T')
        await batch.start()
        await batch.cancel()
        expect(api.cancel).toHaveBeenCalledWith('op-1')
        batch.stopPolling()
    })
})
