<script setup lang="ts">
import { t } from '../i18n'
import { computed, ref, watch } from 'vue'
import { NButton, NCheckbox, NCheckboxGroup, NModal, NPopconfirm } from 'naive-ui'
import type { PromptService } from '../../bindings/github.com/Daneliya/prompt-next/internal/desktop'
import { formatPromptNumber } from '../prompt'

type Entry = NonNullable<Awaited<ReturnType<typeof PromptService.ListTrash>>>[number]
const props = defineProps<{
  show: boolean
  entries: Entry[]
  busy: boolean
  formatDate: (value: string) => string
  libraryLabel: (kind: string) => string
}>()
const emit = defineEmits<{ close: []; restore: [refs: Entry['ref'][]]; purge: [refs: Entry['ref'][]] }>()
const checked = ref<string[]>([])
const key = (entry: Entry) => entry.ref.batch + ':' + entry.ref.id
const selected = computed(() => props.entries.filter(entry => checked.value.includes(key(entry))))
watch(() => props.entries, entries => {
  const valid = new Set(entries.map(key))
  checked.value = checked.value.filter(id => valid.has(id))
})
watch(() => props.show, show => { if (!show) checked.value = [] })
</script>

<template>
  <NModal :show="show" preset="card" :title="t('回收站')" class="app-modal trash-modal"
    :mask-closable="!busy" :close-on-esc="!busy" :closable="!busy" @update:show="emit('close')">
    <p>{{ t('删除后的提示词保留 24 小时。恢复后会回到原来的库。') }}</p>
    <div v-if="!entries.length" class="trash-empty guided-empty">
      <strong>{{ t('回收站为空') }}</strong>
      <p>{{ t('删除的提示词会在这里保留 24 小时。') }}</p>
    </div>
    <NCheckboxGroup v-else v-model:value="checked" :disabled="busy" class="trash-list">
      <div v-for="entry in entries" :key="key(entry)" class="trash-entry">
        <NCheckbox :value="key(entry)" :aria-label="t('选择 ') + entry.title" />
        <div>
          <strong>{{ entry.title }}</strong>
          <small>{{ formatPromptNumber(entry.number) }} · {{ libraryLabel(entry.kind) }}</small>
          <small>{{ t('可恢复至') }} {{ formatDate(new Date(entry.expiresAt * 1000).toISOString()) }}</small>
        </div>
        <div class="actions trash-entry-actions">
          <NButton size="small" :disabled="busy" @click="emit('restore', [entry.ref])">{{ t('恢复') }}</NButton>
          <NPopconfirm :negative-text="t('取消')" :positive-text="t('彻底删除')" :positive-button-props="{ type: 'error' }"
            @positive-click="emit('purge', [entry.ref])">
            <template #trigger><NButton size="small" type="error" quaternary :disabled="busy">{{ t('彻底删除') }}</NButton></template>
            {{ t('彻底删除「') }}{{ entry.title }}{{ t('」？删除后无法从回收站恢复。') }}
          </NPopconfirm>
        </div>
      </div>
    </NCheckboxGroup>
    <template v-if="entries.length" #footer>
      <div class="actions trash-actions">
        <NButton size="small" :disabled="busy" @click="checked = selected.length === entries.length ? [] : entries.map(key)">
          {{ selected.length === entries.length ? t('取消全选') : t('全选') }}
        </NButton>
        <NButton size="small" :disabled="busy" @click="emit('restore', entries.map(entry => entry.ref))">{{ t('恢复全部') }}</NButton>
        <NPopconfirm :negative-text="t('取消')" :positive-text="t('彻底删除')" :positive-button-props="{ type: 'error' }"
          @positive-click="emit('purge', selected.map(entry => entry.ref))">
          <template #trigger><NButton size="small" type="error" :disabled="!selected.length || busy">{{ t('彻底删除选中') }}</NButton></template>
          {{ t('彻底删除选中的') }} {{ selected.length }} {{ t('条提示词？删除后无法从回收站恢复。') }}
        </NPopconfirm>
        <NButton size="small" type="primary" :disabled="!selected.length || busy"
          @click="emit('restore', selected.map(entry => entry.ref))">{{ t('恢复选中（{0}）', selected.length) }}</NButton>
      </div>
    </template>
  </NModal>
</template>
