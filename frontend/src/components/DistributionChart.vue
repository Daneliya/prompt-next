<script setup lang="ts">
import { computed, ref } from 'vue'
import { t } from '../i18n'

const props = defineProps<{
  title: string
  subtitle: string
  items: { name: string; count: number }[]
}>()
const palette = [1, 2, 3, 4, 5, 6].map(index => `var(--distribution-${index})`)
const selected = ref<string | null>(null)
const hovered = ref<string | null>(null)
const total = computed(() => props.items.reduce((sum, item) => sum + item.count, 0))
const slices = computed(() => {
  const ordered = props.items.filter(item => item.count > 0)
    .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name, 'zh-CN'))
  const head = ordered.slice(0, 5).map(item => ({ ...item, key: `item:${item.name}` }))
  const tail = ordered.slice(5)
  if (tail.length) head.push({ key: 'other', name: t('其余 {0} 项', tail.length), count: tail.reduce((sum, item) => sum + item.count, 0) })
  let start = 0
  return head.map((item, index) => {
    const share = item.count / total.value * 100
    const gap = head.length > 1 ? Math.min(1.1, share / 3) : 0
    const slice = {
      ...item, color: palette[index],
      start: start + gap / 2, length: share - gap,
      percent: share < 0.1 ? '<0.1%' : `${Number(share.toFixed(1))}%`,
    }
    start += share
    return slice
  })
})
const active = computed(() => slices.value.find(slice => slice.key === hovered.value)
  ?? slices.value.find(slice => slice.key === selected.value))

function toggle(key: string) {
  selected.value = selected.value === key ? null : key
}

function clearSelection() {
  selected.value = null
  hovered.value = null
}
</script>

<template>
  <section class="chart-card distribution-card" :aria-label="title" @keydown.esc="clearSelection">
    <header class="chart-heading"><div><h2>{{ title }}</h2><p>{{ subtitle }}</p></div></header>
    <div class="distribution-content">
      <div class="distribution-ring">
        <svg viewBox="0 0 124 124" aria-hidden="true" @pointerleave="hovered = null">
          <circle cx="62" cy="62" r="49" class="distribution-track" />
          <circle v-for="slice in slices" :key="slice.key" cx="62" cy="62" r="49" pathLength="100"
            class="distribution-segment" :class="{ 'is-active': active?.key === slice.key, 'is-muted': active && active.key !== slice.key }"
            :stroke="slice.color" :stroke-dasharray="`${slice.length} 100`" :stroke-dashoffset="-slice.start"
            transform="rotate(-90 62 62)" @pointerenter="hovered = slice.key" @pointerleave="hovered = null" @click="toggle(slice.key)" />
        </svg>
        <div class="distribution-center">
          <strong>{{ active?.count ?? total }}</strong>
          <span>{{ active?.percent ?? (total ? t('合计') : t('暂无数据')) }}</span>
        </div>
      </div>
      <ul v-if="slices.length" class="distribution-legend">
        <li v-for="slice in slices" :key="slice.key">
          <button type="button" :class="{ 'is-active': active?.key === slice.key }" :aria-pressed="selected === slice.key"
            :title="`${slice.name} · ${slice.count} · ${slice.percent}`"
            @pointerenter="hovered = slice.key" @pointerleave="hovered = null"
            @focus="hovered = slice.key" @blur="hovered = null" @click="toggle(slice.key)">
            <span class="chart-swatch" :style="{ background: slice.color }" aria-hidden="true" />
            <span class="distribution-label">{{ slice.name }}</span>
            <strong>{{ slice.count }}</strong>
          </button>
        </li>
      </ul>
    </div>
  </section>
</template>
