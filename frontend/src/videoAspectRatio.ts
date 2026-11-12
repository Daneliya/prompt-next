function readVideoRatio(src: string, signal: AbortSignal): Promise<number | undefined> {
  return new Promise(resolve => {
    if (signal.aborted) {
      resolve(undefined)
      return
    }

    const video = document.createElement('video')
    function finish() {
      signal.removeEventListener('abort', finish)
      video.removeEventListener('loadedmetadata', finish)
      video.removeEventListener('error', finish)
      const ratio = !signal.aborted && video.videoWidth && video.videoHeight
        ? video.videoWidth / video.videoHeight
        : undefined
      video.removeAttribute('src')
      video.load()
      resolve(ratio)
    }

    video.preload = 'metadata'
    video.muted = true
    signal.addEventListener('abort', finish, { once: true })
    video.addEventListener('loadedmetadata', finish, { once: true })
    video.addEventListener('error', finish, { once: true })
    video.src = src
    video.load()
  })
}

export async function readVideoRatios(sources: string[], signal: AbortSignal) {
  const result = new Map<string, number>()
  if (!sources.length || signal.aborted) return result

  // Bound simultaneous media reads and the total time spent preparing a page.
  const controller = new AbortController()
  const deadline = controller.signal
  const cancel = () => controller.abort()
  const timer = setTimeout(cancel, 4000)
  signal.addEventListener('abort', cancel, { once: true })
  let next = 0
  async function readNext() {
    while (next < sources.length && !deadline.aborted) {
      const src = sources[next++]!
      const ratio = await readVideoRatio(src, deadline)
      if (ratio) result.set(src, ratio)
    }
  }

  try {
    await Promise.all(Array.from({ length: Math.min(8, sources.length) }, readNext))
    return result
  } finally {
    clearTimeout(timer)
    signal.removeEventListener('abort', cancel)
  }
}
