import { describe, expect, it, vi } from 'vitest'
import { readVideoRatios } from './videoAspectRatio'

describe('video cover metadata', () => {
  it('retains the landscape and portrait ratios and leaves unavailable covers for the fallback', async () => {
    class Video extends EventTarget {
      src = ''
      videoWidth = 0
      videoHeight = 0

      removeAttribute() { this.src = '' }

      load() {
        if (!this.src) return
        const size = this.src === 'landscape' ? [1920, 1080] : this.src === 'portrait' ? [1080, 1920] : []
        queueMicrotask(() => {
          this.videoWidth = size[0] ?? 0
          this.videoHeight = size[1] ?? 0
          this.dispatchEvent(new Event(size.length ? 'loadedmetadata' : 'error'))
        })
      }
    }
    vi.stubGlobal('document', { createElement: () => new Video() })
    try {
      const controller = new AbortController()
      const ratios = await readVideoRatios(['landscape', 'portrait', 'unavailable'], controller.signal)
      expect(Object.fromEntries(ratios)).toEqual({ landscape: 16 / 9, portrait: 9 / 16 })

      const cancelled = readVideoRatios(['landscape', 'portrait'], controller.signal)
      controller.abort()
      await expect(cancelled).resolves.toEqual(new Map())
    } finally {
      vi.unstubAllGlobals()
    }
  })
})
