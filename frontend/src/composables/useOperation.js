import { ref, computed } from 'vue'

// createOperationState tracks one long-running operation by ID. Events
// belonging to any other operation are ignored, so a stale listener can
// never render another action's progress. Message keys arrive from the
// backend; raw backend strings are never rendered.
export function createOperationState(operationId) {
    const phase = ref('idle')
    const progress = ref(0)
    const messageKey = ref('')
    const cancellable = ref(false)
    const error = ref(null)

    const isTerminal = computed(() =>
        ['done', 'failed', 'cancelled'].includes(phase.value))

    function consume(event) {
        if (!event || event.operationId !== operationId) return
        phase.value = event.phase ?? phase.value
        progress.value = Math.max(0, Math.min(100, Number(event.progress) || 0))
        messageKey.value = event.messageKey ?? messageKey.value
        cancellable.value = Boolean(event.cancellable)
        if (event.publicError) {
            error.value = {
                code: event.publicError.code ?? 'internal',
                messageKey: event.publicError.messageKey ?? 'errors.internal',
                retryable: Boolean(event.publicError.retryable),
            }
        } else if (phase.value === 'failed' && !error.value) {
            error.value = { code: 'internal', messageKey: 'errors.internal', retryable: false }
        }
    }

    function reset() {
        phase.value = 'idle'
        progress.value = 0
        messageKey.value = ''
        cancellable.value = false
        error.value = null
    }

    return { operationId, phase, progress, messageKey, cancellable, error, isTerminal, consume, reset }
}

// useOperation mirrors the plan-facing composable signature.
export function useOperation(operationId) {
    const state = createOperationState(operationId)
    return state
}
