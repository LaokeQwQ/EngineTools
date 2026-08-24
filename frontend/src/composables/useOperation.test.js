import { describe, it, expect } from 'vitest'
import { createOperationState } from './useOperation'

describe('createOperationState', () => {
    it('ignores events for another operation', () => {
        const state = createOperationState('op-1')
        state.consume({ operationId: 'op-2', phase: 'applying', completedBytes: 9, totalBytes: 10 })
        expect(state.phase.value).toBe('idle')
    })

    it('consumes matching events and clamps progress', () => {
        const state = createOperationState('op-1')
        state.consume({ operationId: 'op-1', phase: 'run', progress: 150, messageKey: 'ops.working', cancellable: true })
        expect(state.phase.value).toBe('run')
        expect(state.progress.value).toBe(100)
        expect(state.messageKey.value).toBe('ops.working')
        expect(state.cancellable.value).toBe(true)
        expect(state.isTerminal.value).toBe(false)
    })

    it('captures the public error on failure without raw strings', () => {
        const state = createOperationState('op-1')
        state.consume({
            operationId: 'op-1',
            phase: 'failed',
            publicError: { code: 'backup_invalid', messageKey: 'errors.backupInvalid', retryable: true },
        })
        expect(state.isTerminal.value).toBe(true)
        expect(state.error.value).toEqual({
            code: 'backup_invalid',
            messageKey: 'errors.backupInvalid',
            retryable: true,
        })
        expect(Object.values(state.error.value).some((v) => typeof v === 'string' && v.includes('C:\\'))).toBe(false)
    })

    it('reset returns to idle', () => {
        const state = createOperationState('op-1')
        state.consume({ operationId: 'op-1', phase: 'done', progress: 100 })
        state.reset()
        expect(state.phase.value).toBe('idle')
        expect(state.progress.value).toBe(0)
        expect(state.isTerminal.value).toBe(false)
    })
})
