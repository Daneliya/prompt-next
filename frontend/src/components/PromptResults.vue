<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NCheckboxGroup, NPagination, NVirtualList } from 'naive-ui'
import { PromptService } from '../../bindings/github.com/Daneliya/prompt-next/internal/desktop'
import { videoAsset, type Prompt } from '../prompt'
import { readVideoRatios } from '../videoAspectRatio'
import { t } from '../i18n'

const props = defineProps<{ items: Prompt[]; layout: string; writtenLibrary: boolean }>()
const checked = defineModel<string[]>('checked', { required: true })
const page = defineModel<number>('page', { required: true })
defineSlots<{ default(props: { item: Prompt; ratio?: number }): unknown }>()

const pageSize = 72
const virtualList = ref<InstanceType<typeof NVirtualList>>()
defineExpose({ scrollContainer: () => virtualList.value?.getScrollContainer() })
const pageCount = computed(() => Math.max(1, Math.ceil(props.items.length / pageSize)))
watch(pageCount, count => { if (page.value > count) page.value = count }, { flush: 'sync' })
const pageItems = computed(() => props.items.slice((page.value - 1) * pageSize, page.value * pageSize))
function coverKey(item: Prompt) {
  if (item.videos?.length) return videoAsset(item.number, item.videos[0]!)
  if (item.images?.length) return item.number + '/' + item.images[0]
  return ''
}

const ratios = new Map<string, number>()
const layoutReady = ref(true)
const coverSignature = computed(() => props.layout === 'waterfall' ? pageItems.value.map(coverKey).join('\n') : '')
watch(coverSignature, async (_, __, onCleanup) => {
  const missing = props.layout === 'waterfall'
    ? pageItems.value.filter(item => coverKey(item) && !ratios.has(coverKey(item)))
    : []
  layoutReady.value = !missing.length
  if (!missing.length) return

  const controller = new AbortController()
  onCleanup(() => controller.abort())
  const images = missing.filter(item => !item.videos?.length)
  const videos = missing.filter(item => item.videos?.length).map(coverKey)
  const readImages = async () => {
    if (!images.length) return
    const dimensions = await PromptService.CoverDimensions(Object.fromEntries(images.map(item => [item.number, item.images![0]!])))
    for (const item of images) ratios.set(coverKey(item), dimensions?.[item.number] ?? 4 / 3)
  }
  const readVideos = async () => {
    const dimensions = await readVideoRatios(videos, controller.signal)
    for (const [key, ratio] of dimensions) ratios.set(key, ratio)
  }
  await Promise.allSettled([readImages(), readVideos()])
  if (!controller.signal.aborted) layoutReady.value = true
}, { immediate: true, flush: 'pre' })
</script>

<template>
  <NCheckboxGroup v-model:value="checked" :class="layout === 'list' ? 'virtual-results' : 'paged-results'">
    <NVirtualList v-if="layout === 'list'" ref="virtualList" class="prompt-virtual-list" :items="items"
      key-field="id" :item-size="76" item-resizable>
      <template #default="{ item }"><slot :item="item" /></template>
    </NVirtualList>
    <template v-else>
      <div v-if="layoutReady" :class="{ gallery: layout === 'cards', waterfall: layout === 'waterfall', 'written-gallery': writtenLibrary }">
        <template v-for="item in pageItems" :key="item.id"><slot :item="item" :ratio="ratios.get(coverKey(item))" /></template>
      </div>
      <div v-else class="gallery browse-skeleton" role="status" :aria-label="t('正在加载提示词…')">
        <span v-for="index in Math.min(pageItems.length, 8)" :key="index" aria-hidden="true" />
      </div>
      <div v-if="pageCount > 1" class="browse-pagination">
        <NPagination v-model:page="page" :page-count="pageCount" :page-slot="5" size="small" />
      </div>
    </template>
  </NCheckboxGroup>
</template>
