import type { QuickSelectConfig } from '../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/models'
import type { QuickSelectScan } from './quickSelectCore'

export const QUICK_SELECT_BUDGET_MS = 200

interface WorkerReply {
  result?: QuickSelectScan
  validationErrors?: string[]
  error?: string
}

function requestWorker(
  request: Record<string, unknown>,
  signal?: AbortSignal,
  budgetMs = QUICK_SELECT_BUDGET_MS,
): Promise<WorkerReply> {
  if (signal?.aborted) return Promise.reject(new DOMException('Quick Select cancelled', 'AbortError'))
  return new Promise((resolve, reject) => {
    const worker = new Worker(new URL('./quickSelect.worker.ts', import.meta.url), { type: 'module' })
    let settled = false
    const finish = (error?: unknown, reply?: WorkerReply) => {
      if (settled) return
      settled = true
      clearTimeout(timeout)
      signal?.removeEventListener('abort', abort)
      worker.terminate()
      if (error) reject(error)
      else resolve(reply!)
    }
    const abort = () => finish(new DOMException('Quick Select cancelled', 'AbortError'))
    const timeout = setTimeout(() => finish(new Error(`Quick Select exceeded ${budgetMs} ms`)), budgetMs)
    signal?.addEventListener('abort', abort, { once: true })
    if (signal?.aborted) { abort(); return }
    worker.onmessage = (event: MessageEvent<WorkerReply>) => {
      if (event.data.error) finish(new Error(event.data.error))
      else if (event.data.result || event.data.validationErrors) finish(undefined, event.data)
      else finish(new Error('Quick Select worker returned an invalid response'))
    }
    worker.onerror = event => finish(new Error(event.message || 'Quick Select worker failed'))
    try { worker.postMessage(request) }
    catch (error) { finish(error) }
  })
}

export async function scanQuickSelectInWorker(
  text: string,
  config: QuickSelectConfig,
  signal?: AbortSignal,
  budgetMs = QUICK_SELECT_BUDGET_MS,
): Promise<QuickSelectScan> {
  const reply = await requestWorker({ mode: 'scan', text, config }, signal, budgetMs)
  if (!reply.result) throw new Error(reply.error || 'Quick Select worker returned an invalid scan')
  return reply.result
}

export async function validateQuickSelectRulesInWorker(
  rules: QuickSelectConfig['rules'],
  signal?: AbortSignal,
  budgetMs = QUICK_SELECT_BUDGET_MS,
): Promise<string[]> {
  const reply = await requestWorker({ mode: 'validate', rules: rules ?? [] }, signal, budgetMs)
  if (!reply.validationErrors) throw new Error(reply.error || 'Quick Select worker returned an invalid validation response')
  return reply.validationErrors
}
