<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NButton } from 'naive-ui'
import { t } from '../i18n'

const props = defineProps<{ src: string }>()
const emit = defineEmits<{ load: [event: Event] }>()
const viewport = ref<HTMLElement>()
const image = ref<HTMLImageElement>()
const zoom = ref(1)
const offset = ref({ x: 0, y: 0 })
const drag = ref<{ id: number; x: number; y: number }>()
const imageStyle = computed(() => ({
  transform: `translate(${offset.value.x}px, ${offset.value.y}px) scale(${zoom.value})`,
}))

function reset() {
  zoom.value = 1
  offset.value = { x: 0, y: 0 }
  drag.value = undefined
}

watch(() => props.src, reset)

function panTo(x: number, y: number) {
  if (!viewport.value || !image.value) return
  const limitX = Math.max(0, (image.value.offsetWidth * zoom.value - viewport.value.clientWidth) / 2)
  const limitY = Math.max(0, (image.value.offsetHeight * zoom.value - viewport.value.clientHeight) / 2)
  offset.value = {
    x: Math.max(-limitX, Math.min(limitX, x)),
    y: Math.max(-limitY, Math.min(limitY, y)),
  }
}

function zoomBy(factor: number, x = 0, y = 0) {
  const next = Math.max(1, Math.min(6, zoom.value * factor))
  const ratio = next / zoom.value
  zoom.value = next
  panTo(x + (offset.value.x - x) * ratio, y + (offset.value.y - y) * ratio)
}

function onWheel(event: WheelEvent) {
  if (!viewport.value || !event.deltaY) return
  const bounds = viewport.value.getBoundingClientRect()
  const delta = event.deltaY * (event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? bounds.height : 1)
  zoomBy(Math.exp(-delta * 0.002), event.clientX - bounds.left - bounds.width / 2, event.clientY - bounds.top - bounds.height / 2)
}

function startPan(event: PointerEvent) {
  if (zoom.value <= 1 || event.button !== 0 || drag.value) return
  viewport.value?.setPointerCapture(event.pointerId)
  drag.value = { id: event.pointerId, x: event.clientX, y: event.clientY }
}

function movePan(event: PointerEvent) {
  if (drag.value?.id !== event.pointerId) return
  panTo(offset.value.x + event.clientX - drag.value.x, offset.value.y + event.clientY - drag.value.y)
  drag.value = { id: event.pointerId, x: event.clientX, y: event.clientY }
}

function endPan(event: PointerEvent) {
  if (drag.value?.id === event.pointerId) drag.value = undefined
}
</script>

<template>
  <div class="image-preview-view">
    <div ref="viewport" class="image-preview-viewport" :class="{ zoomed: zoom > 1, panning: drag }" tabindex="0"
      :aria-label="t('滚轮缩放，放大后可拖动')"
      @wheel.prevent.stop="onWheel" @pointerdown="startPan" @pointermove="movePan" @pointerup="endPan"
      @pointercancel="endPan" @lostpointercapture="endPan">
      <img ref="image" :key="src" :src="src" :style="imageStyle" :alt="t('图片预览')" draggable="false" @load="emit('load', $event)">
    </div>
    <div class="image-zoom-controls">
      <span>{{ t('滚轮缩放，放大后可拖动') }}</span>
      <NButton size="tiny" :disabled="zoom <= 1" :aria-label="t('缩小')" @click="zoomBy(1 / 1.25)">−</NButton>
      <NButton size="tiny" :title="t('重置缩放')" :aria-label="t('重置缩放')" @click="reset">{{ Math.round(zoom * 100) }}%</NButton>
      <NButton size="tiny" :disabled="zoom >= 6" :aria-label="t('放大')" @click="zoomBy(1.25)">＋</NButton>
    </div>
  </div>
</template>
