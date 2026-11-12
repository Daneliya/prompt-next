<script setup lang="ts">
import { computed } from 'vue'
import { NButton } from 'naive-ui'
import type { StorageStats } from '../../bindings/github.com/Daneliya/prompt-next/internal/library'
import { libraries, type Kind, type Prompt } from '../prompt'
import { countLabels, promptModels } from '../composables/usePromptFilters'
import { t } from '../i18n'
import AppIcon from './AppIcon.vue'
import DistributionChart from './DistributionChart.vue'
import PromptTrend from './PromptTrend.vue'

const props = defineProps<{
  prompts: Prompt[]
  stats?: StorageStats
  formatDate: (value: string) => string
  formatBytes: (bytes?: number) => string
}>()
const emit = defineEmits<{ open: [kind: Kind, id?: string]; newArtwork: [] }>()
const libraryCounts = computed(() => Object.fromEntries(countLabels(props.prompts.map(item => item.kind)).map(item => [item.name, item.count])))
const recent = computed(() => props.prompts.slice(0, 6))
const artwork = computed(() => props.prompts.filter(item => item.kind === 'art'))
const categoryDistribution = computed(() => countLabels(artwork.value.map(item => item.category || t('未分类'))))
const modelDistribution = computed(() => countLabels(artwork.value.flatMap(item => {
  const names = promptModels(item)
  return names.length ? names : [t('未选模型')]
})))

</script>

<template>
<section class="home-pane" :aria-label="t('工作台首页')">
  <div class="hero">
    <div><p>{{ t('本地创作工作台') }}</p><h1>Prompt Next</h1><p>{{ t('整理文本提示词、绘画参考图和角色设定。') }}</p>
      <div class="actions"><NButton type="primary" @click="emit('open', 'text')">{{ t('进入文本库') }}</NButton><NButton @click="emit('open', 'art')">{{ t('进入绘画库') }}</NButton><NButton @click="emit('newArtwork')">{{ t('新建绘画提示词') }}</NButton></div>
    </div>
    <img class="hero-logo" src="/icons/prompt-next.svg" alt="Prompt Next Logo">
  </div>
  <div class="overview">
    <button v-for="item in libraries" :key="item.kind" class="stat-card" @click="emit('open', item.kind)">
      <span class="library-icon" :class="item.kind"><AppIcon :name="item.icon" /></span>
      <span>{{ t(item.label) }}</span><strong>{{ libraryCounts[item.kind] ?? 0 }} {{ t('条') }}</strong>
    </button>
    <div class="stat-card"><span class="library-icon"><AppIcon name="disk" /></span>
      <span>{{ t('数据库占用') }}</span><strong>{{ formatBytes(stats?.dbBytes) }}</strong>
    </div>
    <div class="stat-card"><span class="library-icon art"><AppIcon name="art" /></span>
      <span>{{ t('图片占用 ·') }} {{ stats?.imageFiles ?? 0 }} {{ t('张') }}</span><strong>{{ formatBytes(stats?.imageBytes) }}</strong>
    </div>
    <div v-if="stats?.videoFiles" class="stat-card"><span class="library-icon"><AppIcon name="video" /></span>
      <span>{{ t('视频占用 ·') }} {{ stats.videoFiles }} {{ t('个') }}</span><strong>{{ formatBytes(stats.videoBytes) }}</strong>
    </div>
  </div>
  <div class="dashboard-charts">
    <PromptTrend :days="stats?.recentPrompts ?? []" />
    <DistributionChart :title="t('绘画分类')" :subtitle="t('按绘画条目统计')" :items="categoryDistribution" />
    <DistributionChart :title="t('AI 模型')" :subtitle="t('多选模型分别计数')" :items="modelDistribution" />
  </div>
  <div class="home-section">
    <h2>{{ t('最近更新') }}</h2>
    <p v-if="!recent.length" class="section-label">{{ t('还没有提示词') }}</p>
    <button v-for="item in recent" :key="item.id" class="recent-row" @click="emit('open', item.kind as Kind, item.id)">
      <span>{{ item.title }}</span><small>{{ t(libraries.find(library => library.kind === item.kind)?.label) }} · {{ formatDate(item.updatedAt) }}</small>
    </button>
  </div>
  <div v-if="stats" class="data-path"><strong>{{ t('本地共占用') }} {{ formatBytes(stats.dbBytes + stats.imageBytes + stats.videoBytes) }}</strong><code>{{ stats.directory }}</code></div>
</section>
</template>
