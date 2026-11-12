import { shallowRef, watch, type Ref } from 'vue'
import { PromptService } from '../../bindings/github.com/Daneliya/prompt-next/internal/desktop'

type Stats = Awaited<ReturnType<typeof PromptService.StorageSnapshot>>

export function useStorageStats(visible: Readonly<Ref<boolean>>, onError: (error: unknown) => void) {
  const stats = shallowRef<Stats>()
  let requested = 1
  let completed = 0
  let mediaRevision = 1
  let scannedRevision = 0
  let pending: Promise<void> | undefined

  function refresh(): Promise<void> {
    if (pending) return pending
    if (completed === requested) return Promise.resolve()
    pending = (async () => {
      while (visible.value && completed !== requested) {
        const revision = requested
        const media = mediaRevision
        const scanMedia = media !== scannedRevision
        const snapshot = await PromptService.StorageSnapshot(scanMedia)
        const previous = stats.value
        stats.value = !scanMedia && previous ? {
          ...snapshot, imageBytes: previous.imageBytes, imageFiles: previous.imageFiles,
          videoBytes: previous.videoBytes, videoFiles: previous.videoFiles,
        } : snapshot
        if (scanMedia) scannedRevision = media
        completed = revision
      }
    })().finally(() => { pending = undefined })
    return pending
  }

  function invalidate(media = false) {
    requested++
    if (media) mediaRevision++
    if (visible.value) void refresh().catch(onError)
  }

  watch(visible, active => { if (active) void refresh().catch(onError) })
  return { stats, refresh, invalidate }
}
