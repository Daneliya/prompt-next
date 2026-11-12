<script setup lang="ts">
import { NButton, NCheckbox, NCheckboxGroup, NDatePicker, NInput, NModal, NPopover, NSelect, NTag } from 'naive-ui'
import type { PromptService } from '../../bindings/github.com/Daneliya/prompt-next/internal/desktop'
import { t } from '../i18n'
import type { Prompt } from '../prompt'
import type { DraftImage, useEditorImages } from '../composables/useEditorImages'
import EditorImageGroup from './EditorImageGroup.vue'
import PromptFold from './PromptFold.vue'

type ChosenVideo = NonNullable<Awaited<ReturnType<typeof PromptService.ChooseVideoFiles>>>[number]
type Option = { label: string; value: string }
defineProps<{
  show: boolean
  isDark: boolean
  busy: boolean
  mediaPreviewOpen: boolean
  canSaveDraft: boolean
  categoryOptions: Option[]
  authorOptions: Option[]
  editorModels: string[]
  draftVideoSource: number
  videoDetails: Map<string, number>
  effectImages: ReturnType<typeof useEditorImages>
  referenceImages: ReturnType<typeof useEditorImages>
  videoAsset: (number: number, name: string) => string
  formatBytes: (bytes?: number) => string
  formatDate: (value: string) => string
}>()
const draft = defineModel<Prompt | null>('draft', { required: true })
const tagsText = defineModel<string>('tagsText', { required: true })
const draftCreated = defineModel<number | null>('created', { required: true })
const draftUpdated = defineModel<number | null>('updated', { required: true })
const draftVideoFiles = defineModel<ChosenVideo[]>('videoFiles', { required: true })
const modelPickerOpen = defineModel<boolean>('modelPickerOpen', { required: true })
const modelQuery = defineModel<string>('modelQuery', { required: true })
const emit = defineEmits<{
  close: []
  save: []
  chooseVideos: []
  removeVideo: [index: number]
  preview: [images: DraftImage[], index: number]
}>()
const hasMedia = (kind: string) => kind === 'art' || kind === 'video'
</script>

<template>
<NModal :show="!!draft && show" preset="card" :title="draft?.id ? t('编辑提示词') : t('新建提示词')" class="editor-modal" :class="{ 'is-dark': isDark }" :mask-closable="false" :close-on-esc="!busy && !mediaPreviewOpen && !modelPickerOpen" :closable="!busy" @update:show="value => { if (!value) emit('close') }">
  <form v-if="draft" id="prompt-editor-form" class="editor" @submit.prevent="emit('save')">
    <section class="editor-group">
      <h3>{{ t('基本信息') }}</h3>
      <label><span>{{ t('提示词标题') }} <small class="editor-required">{{ t('必填') }}</small></span> <NInput v-model:value="draft.title" aria-required="true" :placeholder="t('输入标题')" /></label>
      <div class="editor-meta-grid">
        <label>{{ t('分类') }} <NSelect :value="draft.category || null" :options="categoryOptions" clearable filterable :placeholder="t('选择分类，可在设置中添加')" @update:value="value => draft && (draft.category = value ?? '')" /></label>
        <label>{{ t('标签') }} <NInput v-model:value="tagsText" :placeholder="t('使用逗号分隔')" /></label>
      </div>
      <template v-if="hasMedia(draft.kind)">
        <div class="editor-field">
          <span id="model-field-label">{{ t('AI 模型') }}</span>
          <NPopover v-model:show="modelPickerOpen" trigger="click" placement="bottom-start" :show-arrow="false" class="model-picker">
            <template #trigger>
              <button class="model-trigger" type="button" :aria-expanded="modelPickerOpen" aria-labelledby="model-field-label" :disabled="busy">
                <span v-if="!draft.models?.length" class="model-placeholder">{{ t('选择一个或多个模型') }}</span>
                <NTag v-for="model in draft.models" :key="model" size="small"><span class="model-name" :title="model">{{ model }}</span></NTag>
                <span class="model-chevron" aria-hidden="true">⌄</span>
              </button>
            </template>
            <NInput v-model:value="modelQuery" clearable size="small" :placeholder="t('查找模型')" :aria-label="t('查找模型')" />
            <NCheckboxGroup :value="draft.models ?? []" @update:value="names => draft && (draft.models = names.map(String))">
              <div class="model-option-grid">
                <NCheckbox v-for="name in editorModels" :key="name" :value="name"><span class="model-name" :title="name">{{ name }}</span></NCheckbox>
              </div>
            </NCheckboxGroup>
            <p v-if="!editorModels.length" class="model-placeholder">{{ t('没有匹配的模型') }}</p>
            <div class="actions model-picker-footer"><NButton size="small" type="primary" @click="modelPickerOpen = false">{{ t('完成') }}</NButton></div>
          </NPopover>
        </div>
      </template>
    </section>
    <section v-if="draft.kind === 'video'" class="editor-group">
      <div class="block-heading">
        <h3>{{ t('效果视频') }} <small class="editor-required">{{ t('必填') }}</small></h3>
        <NButton size="small" secondary :disabled="busy || (draft.videos?.length ?? 0) + draftVideoFiles.length >= 12" @click="emit('chooseVideos')">{{ t('添加视频') }}</NButton>
      </div>
      <p class="section-label">{{ t('可多选 MP4、WebM，最多 12 段。单段最大 64 MB，每次新增合计最大 192 MB，首段作为封面。') }}</p>
      <div v-for="(name, index) in draft.videos" :key="name" class="draft-video-row">
        <span>{{ t('视频') }} {{ index + 1 }} · {{ formatBytes(videoDetails.get(videoAsset(draftVideoSource, name))) }}</span>
        <NButton size="tiny" quaternary type="error" :disabled="busy" @click="emit('removeVideo', index)">{{ t('移除') }}</NButton>
      </div>
      <div v-for="(file, index) in draftVideoFiles" :key="file.id" class="draft-video-row">
        <span :title="file.name">{{ file.name }} · {{ formatBytes(file.bytes) }}</span>
        <NButton size="tiny" quaternary type="error" :disabled="busy" @click="draftVideoFiles.splice(index, 1)">{{ t('移除') }}</NButton>
      </div>
    </section>
    <EditorImageGroup v-if="draft.kind === 'art' || (draft.kind === 'video' && effectImages.draftImages.value.length)"
      :editor="effectImages" :title="t('效果图')" :required="draft.kind === 'art'" :cover="draft.kind === 'art'" :busy="busy"
      :hint="t(draft.kind === 'art' ? '至少添加一张，首张效果图作为封面。可拖动排序。' : '可保留或移除已有的效果图。')"
      @preview="(items, index) => emit('preview', items, index)" />
    <EditorImageGroup v-if="hasMedia(draft.kind)"
      :editor="referenceImages" :title="t('参考图片')" :hint="t('用于记录生成前的原始图片。可添加多张并拖动排序。')"
      :busy="busy" @preview="(items, index) => emit('preview', items, index)" />
    <p v-if="hasMedia(draft.kind)" class="editor-required">{{ t('中文与英文提示词至少填写一项') }}</p>
    <PromptFold :title="hasMedia(draft.kind) ? t('中文提示词') : t(draft.kind === 'persona' ? '人设提示词（必填）' : '提示词正文（必填）')">
      <NInput v-model:value="draft.body" type="textarea" :autosize="{ minRows: 4, maxRows: 10 }"
        :aria-label="t('提示词正文')" :placeholder="t('可用双花括号标记变量，例如 {{角色名}}')" />
    </PromptFold>
    <PromptFold v-if="hasMedia(draft.kind)" :title="t('英文提示词')">
      <NInput v-model:value="draft.english" type="textarea" :autosize="{ minRows: 4, maxRows: 10 }" :aria-label="t('英文提示词')" />
    </PromptFold>
    <PromptFold v-if="hasMedia(draft.kind)" :title="t('负向提示词')" :initially-open="!!draft.negative">
      <NInput v-model:value="draft.negative" type="textarea" :aria-label="t('负向提示词')" />
    </PromptFold>
    <PromptFold :title="t('补充信息')" :initially-open="false">
      <div class="editor-extra-fields">
        <label>{{ t('说明') }} <NInput v-model:value="draft.description" type="textarea" :placeholder="t('输入补充说明（选填）')" /></label>
        <label v-if="hasMedia(draft.kind)">{{ t('参数备注') }} <NInput v-model:value="draft.parameters" type="textarea" /></label>
      </div>
    </PromptFold>
    <PromptFold :title="t('条目时间')" :initially-open="false">
      <div class="editor-meta-grid">
        <label>{{ t('创建时间') }}
          <NDatePicker v-model:value="draftCreated" type="datetime" format="yyyy-MM-dd HH:mm:ss" clearable
            :placeholder="t(draft.id ? '沿用原创建时间' : '保存时自动记录')" />
        </label>
        <label>{{ t('更新时间') }}
          <NDatePicker v-model:value="draftUpdated" type="datetime" format="yyyy-MM-dd HH:mm:ss" clearable
            :placeholder="t('保存时自动记录')" />
        </label>
      </div>
      <p v-if="draft.updatedAt" class="entry-dates">{{ t('当前更新时间 · {0}', formatDate(draft.updatedAt)) }}</p>
    </PromptFold>
    <section class="editor-group">
      <h3>{{ t('原作者信息') }}</h3>
      <div class="editor-meta-grid">
        <label>{{ t('原作者名称') }} <NSelect :value="draft.author || null" :options="authorOptions" filterable tag clearable
          :placeholder="t('选择或输入作者名称')" @update:value="value => { if (draft) draft.author = value ?? '' }" /></label>
        <label>{{ t('提示词原链接') }} <NInput v-model:value="draft.authorUrl" :placeholder="t('输入网页链接')" /></label>
      </div>
    </section>

  </form>
  <template #footer>
    <div v-if="draft" class="actions editor-footer">
      <NButton :disabled="busy" @click="emit('close')">{{ t('取消') }}</NButton>
      <NButton form="prompt-editor-form" attr-type="submit" type="primary" :loading="busy"
        :disabled="!canSaveDraft">{{ t('保存') }}</NButton>
    </div>
  </template>
</NModal>
</template>
