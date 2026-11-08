<script setup lang="ts">
import { t } from '../i18n'
import { computed } from 'vue'

const props = defineProps<{
  title: string
  subtitle: string
  items: { name: string; count: number }[]
}>()
const palette = ['#b66050', '#597f9d', '#be944e', '#678b7a', '#8f7199', '#9da3ad']
const total = computed(() => props.items.reduce((sum, item) => sum + item.count, 0))
const slices = computed(() => {
  const ordered = [...props.items].sort((a, b) => b.count - a.count || a.name.localeCompare(b.name, 'zh-CN'))
  const head = ordered.slice(0, 5)
  const tail = ordered.slice(5)
  if (tail.length) head.push({ name: t('其余 {0} 项', tail.length), count: tail.reduce((sum, item) => sum + item.count, 0) })
  let position = 0
  return head.map((item, index) => {
    const start = position
    position += item.count / total.value * 100
    return { ...item, color: palette[index], start, end: position }
  })
})
const gradient = computed(() => slices.value.length
  ? `conic-gradient(${slices.value.map(slice => `${slice.color} ${slice.start}% ${slice.end}%`).join(', ')})`
  : undefined)
</script>

<template>
  <section class="chart-card distribution-card" :aria-label="title">
    <header class="chart-heading"><div><h2>{{ title }}</h2><p>{{ subtitle }}</p></div></header>
    <div class="distribution-content">
      <div class="distribution-pie" :style="{ background: gradient }" aria-hidden="true" />
      <ul v-if="slices.length" class="chart-legend">
        <li v-for="slice in slices" :key="slice.start">
          <span class="chart-swatch" :style="{ background: slice.color }" />
          <span :title="slice.name">{{ slice.name }}</span>
          <strong>{{ slice.count }}</strong>
        </li>
      </ul>
      <p v-else class="chart-note">{{ t('暂无数据') }}</p>
    </div>
  </section>
</template>
