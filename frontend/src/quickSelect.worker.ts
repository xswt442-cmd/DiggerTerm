import type { QuickSelectConfig } from '../bindings/github.com/xswt442-cmd/DiggerTerm/internal/desktop/models'
import { scanQuickSelectCandidates, validateQuickSelectRules } from './quickSelectCore'

self.addEventListener('message', (event: MessageEvent<{
	mode?: 'scan' | 'validate'
	text?: string
	config?: QuickSelectConfig
	rules?: QuickSelectConfig['rules']
}>) => {
  try {
    if (event.data.mode === 'validate') self.postMessage({ validationErrors: validateQuickSelectRules(event.data.rules ?? []) })
    else if (event.data.text !== undefined && event.data.config) self.postMessage({ result: scanQuickSelectCandidates(event.data.text, event.data.config) })
    else throw new Error('Quick Select worker received an invalid request')
  } catch (error) {
    self.postMessage({ error: String(error) })
  }
})
