/** Return only bytes newer than lastOffset from one output chunk. */
export function sliceOutput(lastOffset: number, endOffset: number, data: Uint8Array): Uint8Array {
  if (!Number.isSafeInteger(lastOffset) || lastOffset < 0) {
    throw new RangeError('lastOffset must be a safe non-negative integer')
  }
  if (!Number.isSafeInteger(endOffset) || endOffset < 0) {
    throw new RangeError('endOffset must be a safe non-negative integer')
  }
  if (!(data instanceof Uint8Array)) {
    throw new TypeError('output data must be a Uint8Array')
  }
  if (endOffset < data.length) {
    throw new RangeError(`endOffset ${endOffset} is smaller than chunk length ${data.length}`)
  }

  const startOffset = endOffset - data.length
  if (startOffset > lastOffset) {
    throw new Error(`output gap: expected offset ${lastOffset}, received ${startOffset}`)
  }
  if (endOffset <= lastOffset) return new Uint8Array(0)
  return data.subarray(Math.max(0, lastOffset - startOffset))
}
