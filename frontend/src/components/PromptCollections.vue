<script setup lang="ts">
import { t } from '../i18n'
import { computed, ref, watch } from 'vue'
import { NButton, NInput, NModal, NPopconfirm, NSelect } from 'naive-ui'
import type { PromptService } from '../../bindings/github.com/Daneliya/prompt-next/internal/desktop'
import { formatPromptNumber, type Prompt } from '../prompt'
import AppIcon from './AppIcon.vue'
import { imageURL } from '../composables/useEditorImages'

type Collection = Awaited<ReturnType<typeof PromptService.SaveCollection>>
const props = defineProps<{
  selectedId: string
  seedId: string
  collections: Collection[]
  prompts: Prompt[]
  busy: boolean
}>()
const emit = defineEmits<{
  close: []
  select: [id: string]
  save: [collection: Collection]
  remove: [id: string]
  openPrompt: [item: Prompt]
}>()

const name = ref('')
const description = ref('')
const editingId = ref('')
const promptIds = ref<string[]>([])
const selected = computed(() => props.collections.find(item => item.id === props.selectedId))
const promptMap = computed(() => new Map(props.prompts.map(item => [item.id, item])))
const members = computed(() => (selected.value?.promptIds ?? [])
  .map(id => promptMap.value.get(id))
  .filter((item): item is Prompt => item !== undefined))
const promptOptions = computed(() => props.prompts.map(item => ({
  value: item.id,
  label: formatPromptNumber(item.number) + ' · ' + item.title,
})))

const editing = ref(false)
watch(selected, collection => { if (collection) editing.value = false })

function createCollection() {
  editingId.value = ''
  name.value = ''
  description.value = ''
  promptIds.value = props.seedId ? [props.seedId] : []
  editing.value = true
}

function editCollection() {
  editingId.value = selected.value?.id ?? ''
  name.value = selected.value?.name ?? ''
  description.value = selected.value?.description ?? ''
  const ids = (selected.value?.promptIds ?? []).filter(id => promptMap.value.has(id))
  promptIds.value = [...new Set([...ids, ...(props.seedId ? [props.seedId] : [])])]
  editing.value = true
}

function save() {
  if (!name.value.trim() || props.busy) return
  // Keep members in the undo window linked while editing active members.
  const hidden = (props.collections.find(item => item.id === editingId.value)?.promptIds ?? []).filter(id => !promptMap.value.has(id))
  emit('save', { id: editingId.value, name: name.value, description: description.value, promptIds: [...promptIds.value, ...hidden] })
}
function cover(collection: Collection) {
  for (const id of collection.promptIds ?? []) {
    const item = promptMap.value.get(id)
    const image = item?.images?.[0]
    if (item && image) return imageURL(item.number + '/' + image, true)
  }
  return ''
}

</script>

<template>
  <section class="home-pane collections-page" :aria-label="t('提示词合集')">
    <header class="collections-heading">
      <div><p class="section-label">{{ t('把同一套提示词及其变体放在一起') }}</p><h1>{{ t('提示词合集') }}</h1></div>
      <div class="actions">
        <NButton :disabled="busy" @click="emit('close')">{{ t('返回') }}</NButton>
        <NButton type="primary" :disabled="busy" @click="createCollection">{{ t('＋ 新建合集') }}</NButton>
      </div>
    </header>
    <p v-if="seedId" class="section-label">{{ t('选择已有合集加入这条提示词，也可以创建新的合集。') }}</p>
    <div v-if="collections.length" class="collections-index">
      <button v-for="collection in collections" :key="collection.id" class="collection-card"
        :class="{ 'collection-active': selectedId === collection.id }" type="button"
        :aria-pressed="selectedId === collection.id" :disabled="busy" @click="emit('select', collection.id)">
        <span class="collection-card-cover">
          <img v-if="cover(collection)" :src="cover(collection)" alt="" loading="lazy">
          <AppIcon v-else name="folder" />
        </span>
        <span class="collection-card-body">
          <strong>{{ collection.name }}</strong>
          <span v-if="collection.description" class="collection-card-description">{{ collection.description }}</span>
          <small>{{ collection.promptIds?.filter(id => promptMap.has(id)).length ?? 0 }} {{ t('条提示词') }}</small>
        </span>
      </button>
    </div>
    <div v-else-if="!editing" class="empty collections-empty">{{ t('还没有合集') }}</div>

    <NModal :show="editing" preset="card" :title="editingId ? t('编辑合集') : t('新建合集')"
      class="app-modal collection-modal" :mask-closable="!busy" :close-on-esc="!busy" :closable="!busy"
      @update:show="value => editing = value">
      <form class="collections-form" @submit.prevent="save">
        <label>{{ t('合集名称') }} <NInput v-model:value="name" maxlength="80" :placeholder="t('例如：城市夜景系列')" :disabled="busy" /></label>
        <label>{{ t('合集说明') }} <NInput v-model:value="description" type="textarea" maxlength="1000" show-count
          :placeholder="t('介绍这个合集的主题或用途')" :disabled="busy" /></label>
        <label>{{ t('合集成员') }} <NSelect v-model:value="promptIds" :options="promptOptions" multiple filterable
          :placeholder="t('搜索编号或标题，选择相关提示词')" :disabled="busy" /></label>
        <div class="actions">
          <NButton :disabled="busy" @click="editing = false">{{ t('取消') }}</NButton>
          <NButton type="primary" attr-type="submit" :loading="busy" :disabled="!name.trim()">{{ editingId ? t('保存合集') : t('创建合集') }}</NButton>
        </div>
      </form>
    </NModal>

    <section v-if="selected" class="collections-editor">
      <div class="block-heading">
        <h2>{{ selected.name }} · {{ members.length }} {{ t('条') }}</h2>
        <div class="actions">
          <NButton size="small" :disabled="busy" @click="editCollection">{{ seedId ? t('加入此合集') : t('编辑合集') }}</NButton>
          <NPopconfirm :negative-text="t('取消')" :positive-text="t('删除')" @positive-click="emit('remove', selectedId)">
            <template #trigger><NButton size="small" type="error" quaternary :disabled="busy">{{ t('删除合集') }}</NButton></template>
            {{ t('删除合集后，成员提示词仍保留在原来的库中。') }}
          </NPopconfirm>
        </div>
      </div>
      <p v-if="selected.description" class="collection-description">{{ selected.description }}</p>
      <p v-if="!members.length" class="section-label">{{ t('这个合集还没有提示词。') }}</p>
      <button v-for="item in members" :key="item.id" class="collection-member" type="button"
        :disabled="busy" @click="emit('openPrompt', item)">
        <img v-if="item.images?.length" class="collection-thumb"
          :src="imageURL(item.number + '/' + item.images[0]!, true)" alt="" loading="lazy">
        <span class="collection-member-copy">
          <strong>{{ item.title }}</strong>
          <span>{{ item.body || item.english }}</span>
        </span>
        <small>{{ formatPromptNumber(item.number) }}</small>
      </button>
    </section>
  </section>
</template>
