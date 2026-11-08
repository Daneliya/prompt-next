<script setup lang="ts">
import type { DraftImage, useEditorImages } from '../composables/useEditorImages'
import { NButton } from 'naive-ui'
import { t } from '../i18n'

const props = defineProps<{
  editor: ReturnType<typeof useEditorImages>
  title: string
  hint: string
  required?: boolean
  cover?: boolean
  busy: boolean
}>()
const emit = defineEmits<{ preview: [images: DraftImage[], index: number] }>()
const { draftImages, dragFrom, dragOver, imageInput, removeDraftImage, moveImage,
  startDrag, overImage, dropImage, endDrag, addImages } = props.editor
</script>

<template>
  <section class="editor-group">
    <div class="block-heading">
      <h3>{{ title }} <small v-if="required" class="editor-required">{{ t('必填') }}</small>
        <small v-else class="section-label">{{ t('选填') }}</small>
      </h3>
      <NButton size="small" secondary :disabled="busy" @click="imageInput?.click()">{{ t('添加图片') }}</NButton>
    </div>
    <input ref="imageInput" class="hidden" type="file" accept="image/png,image/jpeg,image/gif,image/webp" multiple @change="addImages">
    <p class="section-label">{{ hint }} {{ t('每张图片须在 12 MB 以内') }}</p>
    <div v-if="draftImages.length" class="attached-images editor-images">
      <div v-for="(item, index) in draftImages" :key="item.url" class="draft-image"
        :class="{ 'drop-target': dragOver === index && dragFrom !== index }" :draggable="!busy"
        @dragstart="startDrag($event, index)" @dragover="overImage($event, index)" @drop="dropImage($event, index)" @dragend="endDrag">
        <button class="draft-image-preview" type="button" :aria-label="t('预览图片')" @click="emit('preview', draftImages, index)">
          <img :src="item.url" :alt="title + ' ' + (index + 1)" draggable="false">
          <span v-if="cover && index === 0" class="cover-badge">{{ t('封面') }}</span>
        </button>
        <div class="image-actions">
          <button type="button" :title="t('前移')" :aria-label="t('前移图片')" :disabled="busy || index === 0" @click="moveImage(index, index - 1)">←</button>
          <button type="button" :title="t('后移')" :aria-label="t('后移图片')" :disabled="busy || index === draftImages.length - 1" @click="moveImage(index, index + 1)">→</button>
          <button type="button" :title="t('移除')" :aria-label="t('移除图片')" :disabled="busy" @click="removeDraftImage(index)">×</button>
        </div>
      </div>
    </div>
  </section>
</template>
