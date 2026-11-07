<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{ days: { date: string; count: number }[] }>()
const hovered = ref<number | null>(null)
const left = 34
const right = 443
const top = 14
const bottom = 120
const step = computed(() => Math.max(1, Math.ceil(Math.max(0, ...props.days.map(day => day.count)) / 3)))
const ticks = computed(() => [0, 1, 2, 3].map(index => ({
  label: index * step.value,
  y: bottom - index * (bottom - top) / 3,
})))
const points = computed(() => props.days.map((day, index) => ({
  ...day,
  x: left + index * (right - left) / Math.max(1, props.days.length - 1),
  y: bottom - day.count / (step.value * 3) * (bottom - top),
})))
const line = computed(() => points.value.map(point => `${point.x},${point.y}`).join(' '))
const area = computed(() => points.value.length ? `${left},${bottom} ${line.value} ${right},${bottom}` : '')
const total = computed(() => props.days.reduce((sum, day) => sum + day.count, 0))
const current = computed(() => hovered.value === null ? undefined : points.value[hovered.value])
const labels = computed(() => points.value.filter((_, index) => index % 4 === 0 || index === points.value.length - 1))
</script>

<template>
  <section class="chart-card trend-card" aria-label="近 14 天新增图片">
    <header class="chart-heading"><div><h2>新增图片</h2><p>最近 14 天 · 按文件首次保存日期</p></div><strong>{{ total }} 张</strong></header>
    <svg class="trend-plot" viewBox="0 0 460 151" role="img" aria-label="每日新增图片的折线面积图">
      <defs><linearGradient id="image-trend-fill" x1="0" x2="0" y1="0" y2="1"><stop offset="0" stop-color="currentColor" stop-opacity=".29"/><stop offset="1" stop-color="currentColor" stop-opacity=".05"/></linearGradient></defs>
      <g v-for="tick in ticks" :key="tick.label">
        <line :x1="left" :x2="right" :y1="tick.y" :y2="tick.y" class="trend-grid" />
        <text :x="left - 8" :y="tick.y + 3" text-anchor="end">{{ tick.label }}</text>
      </g>
      <polygon :points="area" fill="url(#image-trend-fill)" />
      <polyline :points="line" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" />
      <g v-for="(point, index) in points" :key="point.date">
        <circle :cx="point.x" :cy="point.y" :r="hovered === index ? 4.4 : 2.3" fill="currentColor" />
        <rect :x="point.x - 13" :y="top" width="26" :height="bottom - top + 6" fill="transparent" @pointerenter="hovered = index" @pointerleave="hovered = null">
          <title>{{ point.date }} · {{ point.count }} 张</title>
        </rect>
      </g>
      <text v-for="point in labels" :key="point.date" :x="point.x" y="140" text-anchor="middle">{{ point.date.slice(5).replace('-', '/') }}</text>
    </svg>
    <p class="chart-note">{{ current ? current.date + ' · 新增 ' + current.count + ' 张' : '同一图片重复引用时只计一次' }}</p>
  </section>
</template>
