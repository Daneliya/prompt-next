<script setup lang="ts">
import { t } from '../i18n'
import { computed } from 'vue'
import { NButton, NModal } from 'naive-ui'
import { imageURL } from '../composables/useEditorImages'
import type { PromptService } from '../../bindings/github.com/Daneliya/prompt-next/internal/desktop'

type OrphanImage = NonNullable<Awaited<ReturnType<typeof PromptService.ListUnreferencedImages>>>[number]
const props = defineProps<{
  files?: OrphanImage[]
  busy: boolean
  formatBytes: (size: number) => string
}>()
const emit = defineEmits<{ close: []; confirm: [names: string[]] }>()
const bytes = computed(() => props.files?.reduce((sum, file) => sum + file.bytes, 0) ?? 0)
</script>

<template>
  <NModal :show="files !== undefined" preset="card" :title="t('清理未被引用的图片')" class="app-modal cleanup-modal"
    :mask-closable="!busy" :close-on-esc="!busy" :closable="!busy" @update:show="emit('close')">
    <template v-if="files?.length">
      <p>{{ t('找到') }} {{ files.length }} {{ t('张，共') }} {{ formatBytes(bytes) }}{{ t('。确认后会重新检查引用，只删除下列仍未被引用的文件。') }}</p>
      <ul class="cleanup-files">
        <li v-for="file in files" :key="file.name">
          <img :src="imageURL(file.name, true)" loading="lazy" alt="">
          <code>{{ file.name }}</code><span>{{ formatBytes(file.bytes) }}</span>
        </li>
      </ul>
    </template>
    <p v-else>{{ t('目前没有可清理的图片。撤销期限内的删除记录仍保留其图片。') }}</p>
    <div class="actions">
      <NButton :disabled="busy" @click="emit('close')">{{ t('关闭') }}</NButton>
      <NButton v-if="files?.length" type="error" :loading="busy" @click="emit('confirm', files.map(file => file.name))">{{ t('确认清理') }}</NButton>
    </div>
  </NModal>
</template>
