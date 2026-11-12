<script setup lang="ts">
import { NCheckbox } from 'naive-ui'
import { t } from '../i18n'
import { formatPromptNumber, promptCharacterCount, type Prompt } from '../prompt'
import { promptModels } from '../composables/usePromptFilters'
import { imageURL } from '../composables/useEditorImages'
import AppIcon from './AppIcon.vue'
import VideoCover from './VideoCover.vue'

defineProps<{
  item: Prompt
  layout: string
  visibleFields: Set<string>
  selectedId: string
  batchMode: boolean
  authorView: boolean
  showLibrary: boolean
  coverRatio?: number
  authorDisplayName: (name: string) => string
  libraryLabel: (kind: string) => string
  videoAsset: (number: number, name: string) => string
  formatDate: (value: string) => string
}>()

const emit = defineEmits<{ select: [event: MouseEvent, item: Prompt]; author: [name: string] }>()
const hasMedia = (kind: string) => kind === 'art' || kind === 'video'
</script>

<template>
<div class="prompt-entry">
<NCheckbox v-if="batchMode" :value="item.id" class="batch-check" :aria-label="t('选中 ') + item.title" />
<button class="prompt-row" :class="{ active: item.id === selectedId, 'waterfall-photo': hasMedia(item.kind) && layout === 'waterfall' && !!(item.videos?.length || item.images?.length), 'written-preview': !hasMedia(item.kind) }" :aria-pressed="item.id === selectedId" :aria-label="item.title" @click="emit('select', $event, item)">
  <span v-if="hasMedia(item.kind) && layout === 'waterfall' && (item.videos?.length || item.images?.length)" class="waterfall-cover" :class="{ 'waterfall-video': item.videos?.length }" :data-video-cover="item.videos?.length ? '' : undefined" :style="coverRatio ? { aspectRatio: String(coverRatio) } : undefined">
    <VideoCover v-if="item.videos?.length" :key="item.videos[0]" :src="videoAsset(item.number, item.videos[0]!)" />
    <img v-else-if="item.images?.length" :src="imageURL(item.number + '/' + item.images[0]!, true)" alt="" loading="lazy">
    <span v-if="visibleFields.has('category') && item.category" class="waterfall-category">{{ item.category }}</span>
    <span v-if="item.favorite" class="waterfall-favorite" :aria-label="t('已收藏')"><AppIcon name="star" /></span>
  </span>
  <span v-else-if="layout === 'cards' && hasMedia(item.kind)" class="card-cover" :class="item.kind" :data-video-cover="item.videos?.length ? '' : undefined">
    <VideoCover v-if="item.videos?.length" :key="item.videos[0]" :src="videoAsset(item.number, item.videos[0]!)" />
    <img class="cover-photo" v-else-if="item.images?.length" :src="imageURL(item.number + '/' + item.images[0]!, true)" alt="" loading="lazy">
    <AppIcon v-else-if="item.kind === 'video'" name="video" />
    <img v-else class="cover-avatar" :src="'/icons/' + item.kind + '-avatar.svg'" alt="" loading="lazy">
    <span v-if="item.images?.length && visibleFields.has('models') && promptModels(item).length"
      class="cover-model">{{ promptModels(item).join(' · ') }}</span>
    <span v-if="item.videos?.length" class="video-count">{{ item.videos.length }} {{ t('段') }}</span>
    <span v-else-if="item.images?.length" class="image-count">{{ item.images.length }} {{ t('张') }}</span>
  </span>
  <span class="row-content">
    <span v-if="showLibrary" class="row-category">{{ libraryLabel(item.kind) }}</span>
    <span v-if="!hasMedia(item.kind) && layout !== 'list'" class="written-heading">
      <span class="written-icon"><AppIcon :name="item.kind === 'persona' ? 'fish' : 'text'" /></span>
      <span v-if="visibleFields.has('category') && (item.category || item.kind === 'persona')" class="written-category" :title="item.category">{{ item.category || t('未分类') }}</span>
      <span v-if="visibleFields.has('characters')" class="written-length">{{ t('{0} 字', promptCharacterCount(item.body)) }}</span>
    </span>
    <span v-if="visibleFields.has('title')" class="row-title" :title="item.title">{{ item.title }} <span v-if="item.favorite && (!hasMedia(item.kind) || layout !== 'waterfall' || !(item.videos?.length || item.images?.length))" class="star"><AppIcon name="star" /></span></span>
    <span v-if="visibleFields.has('body') && (item.body || item.english)" class="row-summary">{{ item.body || item.english }}</span>
    <span v-if="visibleFields.has('description') && (!authorView || item.description) && (item.description || item.kind === 'persona')" class="row-summary" :class="{ 'persona-summary': item.kind === 'persona' }">{{ item.description?.trim() || (item.kind === 'persona' ? item.body : '') }}</span>
    <span v-if="visibleFields.has('models') && promptModels(item).length" class="row-category" :title="promptModels(item).join(' · ')">{{ promptModels(item).join(' · ') }}</span>
    <span v-if="visibleFields.has('category') && (item.category || item.kind === 'persona') && (layout === 'list' || hasMedia(item.kind)) && (layout !== 'waterfall' || !(item.videos?.length || item.images?.length))" class="row-category">{{ item.category || t('未分类') }}</span>
    <span v-if="layout === 'list' && visibleFields.has('characters')" class="row-category">{{ t('{0} 字', promptCharacterCount(item.body)) }}</span>
    <span v-if="visibleFields.has('tags') && item.tags?.length" class="row-category">{{ item.tags.join(' · ') }}</span>
    <span v-if="visibleFields.has('number') || visibleFields.has('created') || visibleFields.has('updated')" class="row-meta">
      <span v-if="visibleFields.has('number')" class="row-number">{{ formatPromptNumber(item.number) }}</span>
      <span v-if="visibleFields.has('created') || visibleFields.has('updated')" class="row-times">
        <span v-if="visibleFields.has('created')">{{ t('创建') }} {{ formatDate(item.createdAt) }}</span>
        <span v-if="visibleFields.has('updated')">{{ t('更新') }} {{ formatDate(item.updatedAt) }}</span>
      </span>
    </span>
  </span>
</button>
<button v-if="visibleFields.has('author') && item.author?.trim()" type="button" class="row-author"
  :title="t('查看作者主页')" @click="emit('author', item.author)">
  <AppIcon name="persona" /><span>{{ authorDisplayName(item.author) }}</span>
</button>
</div>
</template>
