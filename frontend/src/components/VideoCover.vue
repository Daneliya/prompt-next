<script lang="ts">
const posters = new Map<string, string>()
</script>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'

const props = defineProps<{ src: string }>()
const root = ref<HTMLElement>()
const poster = ref(posters.get(props.src))
const visible = ref(false)
const nativeFailed = ref(false)
const fallbackFrame = ref(false)
const nativePoster = props.src.replace('/prompt-videos/', '/prompt-video-posters/')
let observer: IntersectionObserver | undefined
let sampleTime = 0

onMounted(() => {
  if (poster.value || !root.value) return
  if (!('IntersectionObserver' in window)) {
    visible.value = true
    return
  }
  observer = new IntersectionObserver(([entry]) => {
    if (!entry?.isIntersecting) return
    poster.value = posters.get(props.src)
    visible.value = true
    observer?.disconnect()
  }, { rootMargin: '100px' })
  observer.observe(root.value)
})

onUnmounted(() => observer?.disconnect())

function rememberPoster(url: string) {
  if (posters.size >= 64) {
    const oldest = posters.keys().next().value
    if (oldest !== undefined) posters.delete(oldest)
  }
  posters.set(props.src, url)
  poster.value = url
}

function seekFrame(event: Event) {
  const video = event.currentTarget as HTMLVideoElement
  if (Number.isFinite(video.duration) && video.duration > 0.1) {
    sampleTime = Math.min(1.5, video.duration / 4)
    video.currentTime = sampleTime
  }
}

function captureFrame(event: Event) {
  const video = event.currentTarget as HTMLVideoElement
  if (video.seeking || video.readyState < 2 || !video.videoWidth || !video.videoHeight) return
  if (event.type === 'loadeddata' && sampleTime > 0 && video.currentTime < sampleTime) return
  const scale = Math.min(1, 480 / Math.max(video.videoWidth, video.videoHeight))
  const canvas = document.createElement('canvas')
  canvas.width = Math.max(1, Math.round(video.videoWidth * scale))
  canvas.height = Math.max(1, Math.round(video.videoHeight * scale))
  try {
    const context = canvas.getContext('2d')
    if (!context) return
    context.drawImage(video, 0, 0, canvas.width, canvas.height)
    rememberPoster(canvas.toDataURL('image/jpeg', 0.78))
  } catch {
    fallbackFrame.value = true
  }
}
</script>

<template>
  <span ref="root" class="video-cover">
    <img v-if="poster" :src="poster" alt="" draggable="false">
    <img v-else-if="visible && !nativeFailed" :src="nativePoster" alt="" draggable="false"
      @load="rememberPoster(nativePoster)" @error="nativeFailed = true">
    <video v-else-if="visible" :src="src" :class="{ 'video-fallback-ready': fallbackFrame }" preload="auto" muted playsinline
      @loadedmetadata="seekFrame" @loadeddata="captureFrame" @seeked="captureFrame" />
    <span class="video-play-icon" aria-hidden="true">
      <svg viewBox="0 0 24 24"><path d="M8 5.5a1 1 0 0 1 1.55-.83l9.2 6.5a1 1 0 0 1 0 1.66l-9.2 6.5A1 1 0 0 1 8 18.5z" fill="currentColor" /></svg>
    </span>
  </span>
</template>
