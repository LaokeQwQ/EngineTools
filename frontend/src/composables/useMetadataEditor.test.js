import { describe, expect, it, vi } from 'vitest'
import { createMetadataEditor, PATCH_MODES, pictureDataURI } from './useMetadataEditor'

const documentFixture = {
    format: 'mp3',
    tagVersion: 'id3v24',
    title: 'Original title',
    artists: ['Artist'],
    album: 'Album',
    track: { value: 1, total: 10 },
    disc: { value: 1, total: 1 },
    date: '2024',
    genres: ['House'],
    comment: '',
    bpm: 124,
    key: 'Am',
    rating: 0,
    isrc: '',
    lyrics: '',
    pictures: [],
}

function makeApi(overrides = {}) {
    let current = structuredClone(documentFixture)
    return {
        read: vi.fn(async () => structuredClone(current)),
        pictures: vi.fn(async () => []),
        apply: vi.fn(async (_path, patch) => {
            if (patch.title.mode === PATCH_MODES.set) current.title = patch.title.value
            if (patch.title.mode === PATCH_MODES.clear) current.title = ''
            if (patch.artists.mode === PATCH_MODES.set) current.artists = patch.artists.value
            return { ok: true, changed: 1, snapshotId: 'snap-1' }
        }),
        ...overrides,
    }
}

describe('createMetadataEditor', () => {
    it('loads a document and emits explicit unchanged intent', async () => {
        const api = makeApi()
        const editor = createMetadataEditor('song.mp3', { api })
        await editor.load()
        expect(editor.doc.value.title).toBe('Original title')
        expect(editor.dirty.value).toBe(false)
        expect(editor.buildPatch().title).toEqual({ mode: 'unchanged', value: '' })
    })

    it('distinguishes clear from unchanged and set', async () => {
        const editor = createMetadataEditor('song.mp3', { api: makeApi() })
        await editor.load()
        editor.setMode('title', PATCH_MODES.clear)
        expect(editor.dirty.value).toBe(true)
        expect(editor.buildPatch().title).toEqual({ mode: 'clear', value: '' })
        editor.setMode('title', PATCH_MODES.unchanged)
        expect(editor.dirty.value).toBe(false)
        editor.setField('title', 'New title')
        expect(editor.buildPatch().title).toEqual({ mode: 'set', value: 'New title' })
    })

    it('keeps edits and exposes a stable public error when save fails', async () => {
        const api = makeApi({ apply: vi.fn(async () => { throw new Error('C:\\Users\\Alice\\song.mp3: denied') }) })
        const editor = createMetadataEditor('song.mp3', { api })
        await editor.load()
        editor.setField('title', 'Pending')
        await editor.save()
        expect(editor.doc.value.title).toBe('Pending')
        expect(editor.dirty.value).toBe(true)
        expect(editor.error.value).toEqual({ code: 'apply_failed', messageKey: 'errors.apply_failed', retryable: false })
        expect(JSON.stringify(editor.error.value)).not.toContain('Alice')
    })

    it('reloads the backend document after save', async () => {
        const api = makeApi()
        const editor = createMetadataEditor('song.mp3', { api })
        await editor.load()
        editor.setField('title', 'Saved')
        await editor.save()
        expect(api.apply).toHaveBeenCalledTimes(1)
        expect(editor.doc.value.title).toBe('Saved')
        expect(editor.dirty.value).toBe(false)
    })

    it('keeps read descriptors when the secondary picture payload is temporarily empty', async () => {
        const descriptor = {
            id: 'front-1',
            type: 3,
            mime: 'image/jpeg',
            data: 'aGVsbG8=',
            bytes: 5,
        }
        const api = makeApi({
            read: vi.fn(async () => ({ ...structuredClone(documentFixture), pictures: [descriptor] })),
            pictures: vi.fn(async () => []),
        })
        const editor = createMetadataEditor('song.mp3', { api })
        await editor.load()
        expect(editor.pictures.value).toHaveLength(1)
        expect(editor.pictures.value[0]).toMatchObject({ id: 'front-1', data: 'aGVsbG8=' })
    })

    it('assigns a non-empty unique ID when adding a picture', async () => {
        const editor = createMetadataEditor('song.mp3', { api: makeApi() })
        await editor.load()
        editor.replacePicture(null, { type: 3, mime: 'image/jpeg', data: 'aGVsbG8=' })
        editor.replacePicture(null, { type: 3, mime: 'image/jpeg', data: 'aGVsbG8=' })
        const pictures = editor.buildPatch().pictures.value
        expect(pictures).toHaveLength(2)
        expect(pictures[0].id).toBeTruthy()
        expect(pictures[1].id).toBeTruthy()
        expect(pictures[0].id).not.toBe(pictures[1].id)
    })

    it('uses the bounded thumbnail for display without treating it as write payload', async () => {
        const editor = createMetadataEditor('song.mp3', {
            api: makeApi({
                read: vi.fn(async () => ({
                    ...structuredClone(documentFixture),
                    pictures: [{ id: 'front-1', type: 3, mime: 'image/jpeg', bytes: 1000, thumbnail: 'dGh1bWI=', thumbnailMime: 'image/jpeg' }],
                })),
                pictures: vi.fn(async () => []),
            }),
        })
        await editor.load()
        expect(pictureDataURI(editor.pictures.value[0])).toBe('data:image/jpeg;base64,dGh1bWI=')
        editor.setMode('pictures', PATCH_MODES.set)
        expect(editor.buildPatch().pictures.value[0].data).toBe('')
        expect(editor.buildPatch().pictures.value[0].thumbnail).toBe('dGh1bWI=')
    })
})
