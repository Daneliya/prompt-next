<script setup lang="ts">
import { computed, ref } from 'vue'
import type { DailyPromptCount } from '../../bindings/github.com/Daneliya/prompt-next/internal/library'
import { t } from '../i18n'

const props = defineProps<{ days: DailyPromptCount[] }>()
const libraries = [
  { key: 'text', label: '文本提示词库' },
  { key: 'art', label: '绘画提示词库' },
  { key: 'video', label: '视频提示词库' },
  { key: 'persona', label: '大肥鱼人设库' },
] as const
const hovered = ref<number | null>(null)
const left = 34
const right = 440
const top = 12
const bottom = 132
const spacing = computed(() => (right - left) / Math.max(1, props.days.length - 1))
const days = computed(() => props.days.map((day, index) => ({ ...day, x: left + index * spacing.value })))
const step = computed(() => {
  const peak = Math.max(0, ...props.days.flatMap(day => libraries.map(({ key }) => day[key])))
  return Math.max(1, Math.ceil(peak / 3))
})
const ticks = computed(() => [0, 1, 2, 3].map(index => ({
  label: index * step.value,
  y: bottom - index * (bottom - top) / 3,
})))
const series = computed(() => libraries.map(library => {
  const points = days.value.map(day => ({
    x: day.x,
    y: bottom - day[library.key] / (step.value * 3) * (bottom - top),
  }))
  const total = props.days.reduce((sum, day) => sum + day[library.key], 0)
  let line = ''
  if (total > 0) {
    line = `M ${points[0]!.x} ${points[0]!.y}`
    for (let index = 1; index < points.length; index++) {
      const previous = points[index - 1]!
      const point = points[index]!
      // Horizontal control handles keep the curve between adjacent daily counts.
      const middle = (previous.x + point.x) / 2
      line += ` C ${middle} ${previous.y}, ${middle} ${point.y}, ${point.x} ${point.y}`
    }
  }
  const last = points.at(-1)
  const area = line && last ? `${line} L ${last.x} ${bottom} H ${left} Z` : ''
  return { ...library, total, points, line, area }
}))
const total = computed(() => props.days.reduce((sum, day) => sum + day.count, 0))
const current = computed(() => hovered.value === null ? undefined : days.value[hovered.value])
const labels = computed(() => days.value.filter((_, index) =>
  index === days.value.length - 1 || (index % 4 === 0 && index < days.value.length - 2)))
</script>

<template>
  <section class="chart-card trend-card" :aria-label="t('近 14 天新增提示词')">
    <header class="chart-heading"><div><h2>{{ t('新增提示词') }}</h2><p>{{ t('最近 14 天 · 按条目创建日期') }}</p></div><strong>{{ total }} {{ t('条') }}</strong></header>
    <svg class="trend-plot" viewBox="0 0 460 157" role="img" :aria-label="t('每日新增提示词的折线面积图')">
      <defs>
        <linearGradient v-for="entry in series" :id="`trend-fill-${entry.key}`" :key="entry.key" :class="`trend-${entry.key}`" gradientUnits="userSpaceOnUse" :x1="0" :x2="0" :y1="top" :y2="bottom">
          <stop offset="0" stop-color="var(--trend-color)" stop-opacity=".32" />
          <stop offset="1" stop-color="var(--trend-color)" stop-opacity=".07" />
        </linearGradient>
      </defs>
      <g v-for="tick in ticks" :key="tick.label">
        <line :x1="left" :x2="right" :y1="tick.y" :y2="tick.y" class="trend-grid" />
        <text :x="left - 8" :y="tick.y + 3" text-anchor="end">{{ tick.label }}</text>
      </g>
      <path v-for="entry in series" :key="entry.key" :d="entry.area" :fill="`url(#trend-fill-${entry.key})`" />
      <path v-for="entry in series" :key="entry.key" :class="`trend-${entry.key}`" :d="entry.line" fill="none" stroke="var(--trend-color)" stroke-width="1.9" stroke-linecap="round" />
      <g v-if="current && hovered !== null" pointer-events="none">
        <line :x1="current.x" :x2="current.x" :y1="top" :y2="bottom" class="trend-guide" />
        <template v-for="entry in series" :key="entry.key">
          <circle v-if="current[entry.key]" :class="`trend-${entry.key}`" :cx="current.x" :cy="entry.points[hovered]?.y" r="3.5" fill="var(--trend-color)" />
        </template>
      </g>
      <rect v-for="(day, index) in days" :key="day.date" :x="day.x - spacing / 2" :y="top" :width="spacing" :height="bottom - top" fill="transparent" @pointerenter="hovered = index" @pointerleave="hovered = null" />
      <text v-for="day in labels" :key="day.date" :x="day.x" y="150" text-anchor="middle">{{ day.date.slice(5).replace('-', '/') }}</text>
    </svg>
    <p class="chart-note">{{ current ? current.date + t(' · 新增 ') + current.count + ' ' + t('条') : t('统计四个库的在库条目') }}</p>
    <ul class="trend-legend">
      <li v-for="entry in series" :key="entry.key" :class="`trend-${entry.key}`">
        <span class="chart-swatch" aria-hidden="true"></span>{{ t(entry.label) }} <strong>{{ current?.[entry.key] ?? entry.total }}</strong>
      </li>
    </ul>
  </section>
</template>
