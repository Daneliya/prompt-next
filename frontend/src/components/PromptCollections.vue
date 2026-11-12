<script setup lang="ts">
import { computed, h, ref, watch } from 'vue'
import { NButton, NInput, NModal, NPopconfirm, NRadioButton, NRadioGroup, NSelect, type SelectOption } from 'naive-ui'
import type { PromptService } from '../../bindings/github.com/Daneliya/prompt-next/internal/desktop'
import { t } from '../i18n'
import { formatPromptNumber, libraries, type Prompt } from '../prompt'
import { promptModels } from '../composables/usePromptFilters'
import AppIcon from './AppIcon.vue'
import CollectionMedia from './CollectionMedia.vue'

type Collection = Awaited<ReturnType<typeof PromptService.SaveCollection>>
const props = defineProps<{
  selectedId: string
  seedId: string
  collections: Collection[]
  prompts: Prompt[]
  busy: boolean
  formatDate: (value: string) => string
  authorDisplayName: (name: string) => string
}>()
const emit = defineEmits<{
  select: [id: string]
  save: [collection: Collection]
  remove: [id: string]
  openPrompt: [item: Prompt]
  copy: [text: string]
  export: [ids: string[]]
}>()

const search = ref('')
const sort = ref('updated')
const memberLayout = ref('list')
const dragged = ref('')
const dropTarget = ref('')
const editing = ref(false)
const adding = ref(false)
const name = ref('')
const description = ref('')
const editingId = ref('')
const promptIds = ref<string[]>([])
const promptMap = computed(() => new Map(props.prompts.map(item => [item.id, item])))

const summaries = computed(() => props.collections.map(collection => {
  const members = (collection.promptIds ?? []).flatMap(id => {
    const item = promptMap.value.get(id)
    return item ? [item] : []
  })
  return {
    collection, members,
    cover: members.find(item => item.videos?.length || item.images?.length) ?? members[0],
    kinds: new Set(members.map(item => item.kind)).size,
    illustrated: members.filter(item => item.images?.length).length,
    videoCount: members.reduce((sum, item) => sum + (item.videos?.length ?? 0), 0),
  }
}))
const ordered = computed(() => [...summaries.value].sort((a, b) => {
  if (sort.value === 'name') return a.collection.name.localeCompare(b.collection.name, 'zh-CN')
  if (sort.value === 'size') return b.members.length - a.members.length
  return (b.collection.updatedAt ?? '').localeCompare(a.collection.updatedAt ?? '')
}))
const visible = computed(() => {
  const keyword = search.value.trim().toLocaleLowerCase()
  return ordered.value.filter(({ collection }) =>
    [collection.name, collection.description].some(value => value.toLocaleLowerCase().includes(keyword)))
})
const selected = computed(() => summaries.value.find(entry => entry.collection.id === props.selectedId) ?? ordered.value[0])
const promptOptions = computed(() => {
  const existing = new Set(adding.value ? selected.value?.collection.promptIds ?? [] : [])
  return props.prompts.filter(item => !existing.has(item.id)).map(item => ({
    value: item.id,
    label: `${formatPromptNumber(item.number)} · ${props.authorDisplayName(item.author) || t('未署名')} · ${item.title}`,
  }))
})

function renderPromptLabel(option: SelectOption) {
  const label = typeof option.label === 'string' ? option.label : String(option.value ?? '')
  return h('span', { title: label }, label)
}
const sortOptions = computed(() => [
  { value: 'updated', label: t('最近更新') }, { value: 'name', label: t('按名称') }, { value: 'size', label: t('按成员数量') },
])

watch(() => selected.value?.collection, () => { editing.value = false })

function openEditor(collection?: Collection, membersOnly = false) {
  editingId.value = collection?.id ?? ''
  name.value = collection?.name ?? ''
  description.value = collection?.description ?? ''
  const active = (collection?.promptIds ?? []).filter(id => promptMap.value.has(id))
  const seed = props.seedId && !active.includes(props.seedId) ? [props.seedId] : []
  promptIds.value = membersOnly ? seed : [...active, ...seed]
  adding.value = membersOnly
  editing.value = true
}

function mergeMembers(collection: Collection | undefined, active: string[]) {
  // Keep members in the undo window linked and preserve their slots in the order.
  let index = 0
  const ids = (collection?.promptIds ?? []).flatMap(id => {
    if (!promptMap.value.has(id)) return [id]
    return index < active.length ? [active[index++]!] : []
  })
  return [...ids, ...active.slice(index)]
}

function save() {
  if (!name.value.trim() || props.busy || (adding.value && !promptIds.value.length)) return
  const collection = props.collections.find(item => item.id === editingId.value)
  const active = adding.value
    ? [...(collection?.promptIds ?? []).filter(id => promptMap.value.has(id)), ...promptIds.value]
    : promptIds.value
  emit('save', {
    ...collection, id: editingId.value, name: name.value, description: description.value,
    promptIds: mergeMembers(collection, active),
  })
}

function updateMembers(active: string[]) {
  if (!selected.value || props.busy) return
  const { collection } = selected.value
  emit('save', { ...collection, promptIds: mergeMembers(collection, active) })
}

function moveMember(id: string, index: number) {
  const ids = selected.value?.members.map(item => item.id) ?? []
  const from = ids.indexOf(id)
  if (from < 0 || index < 0 || index >= ids.length || index === from) return
  ids.splice(from, 1)
  ids.splice(index, 0, id)
  updateMembers(ids)
}

function startDrag(event: DragEvent, id: string) {
  if (!event.dataTransfer || props.busy) return
  dragged.value = id
  event.dataTransfer.effectAllowed = 'move'
  event.dataTransfer.setData('text/plain', id)
}

function overMember(event: DragEvent, id: string) {
  if (!dragged.value || dragged.value === id || props.busy) return
  event.preventDefault()
  dropTarget.value = id
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'
}

function endDrag() {
  dragged.value = ''
  dropTarget.value = ''
}

function dropMember(index: number) {
  if (dragged.value) moveMember(dragged.value, index)
  endDrag()
}

function copyMembers(members: Prompt[]) {
  const text = members.map(item => [item.title, item.body, item.english,
    item.negative && `${t('负向提示词')}\n${item.negative}`,
    item.parameters && `${t('参数备注')}\n${item.parameters}`]
    .filter(value => value?.trim()).join('\n\n')).join('\n\n---\n\n')
  if (text) emit('copy', text)
}

function libraryName(kind: string) {
  return t(libraries.find(library => library.kind === kind)?.shortLabel)
}
</script>

<template>
  <section class="home-pane collections-page" :aria-label="t('提示词合集')">
    <header class="collections-heading"><h1>{{ t('提示词合集') }}</h1></header>
    <p v-if="seedId" class="collections-seed">{{ t('选择已有合集加入这条提示词，也可以创建新的合集。') }}</p>
    <div class="collections-workspace">
      <aside class="collections-directory" :aria-label="t('合集目录')">
        <div class="collections-directory-top">
          <strong>{{ t('合集目录') }} <small>{{ collections.length }}</small></strong>
          <NInput v-model:value="search" size="small" clearable :placeholder="t('搜索合集')" :aria-label="t('搜索合集')">
            <template #prefix><AppIcon name="search" class="search-icon" /></template>
          </NInput>
          <NSelect v-model:value="sort" size="small" :options="sortOptions" :aria-label="t('排序')" />
          <NButton size="small" type="primary" block :disabled="busy" @click="openEditor()">{{ t('＋ 新建合集') }}</NButton>
        </div>
        <div class="collections-index">
          <button v-for="entry in visible" :key="entry.collection.id" class="collection-card" type="button"
            :aria-pressed="selected?.collection.id === entry.collection.id" :disabled="busy" @click="emit('select', entry.collection.id)">
            <CollectionMedia :item="entry.cover" compact />
            <span class="collection-card-body">
              <strong>{{ entry.collection.name }}</strong>
              <small>{{ t('{0} 条 · 跨 {1} 库', entry.members.length, entry.kinds) }}</small>
            </span>
          </button>
          <p v-if="search.trim() && !visible.length" class="empty collections-no-results">{{ t('没有匹配的合集') }}</p>
        </div>
      </aside>

      <div v-if="selected" class="collections-main">
        <section class="collection-hero">
          <CollectionMedia :item="selected.cover" />
          <div class="collection-hero-copy">
            <p class="section-label">{{ selected.collection.updatedAt ? t('更新于 {0}', formatDate(selected.collection.updatedAt)) : t('提示词合集') }}</p>
            <h2>{{ selected.collection.name }}</h2>
            <p v-if="selected.collection.description" class="collection-description">{{ selected.collection.description }}</p>
            <div class="collection-stats">
              <span>{{ t('{0} 条提示词', selected.members.length) }}</span>
              <span>{{ t('跨 {0} 库', selected.kinds) }}</span>
              <span v-if="selected.illustrated">{{ t('{0} 条带效果图', selected.illustrated) }}</span>
              <span v-if="selected.videoCount">{{ t('{0} 段视频', selected.videoCount) }}</span>
            </div>
            <div class="actions collection-actions">
              <NButton size="small" type="primary" :disabled="busy" @click="openEditor(selected.collection, true)">{{ t('添加成员') }}</NButton>
              <NButton size="small" :disabled="busy || !selected.members.length" @click="copyMembers(selected.members)">{{ t('按顺序复制全部') }}</NButton>
              <NButton size="small" :disabled="busy || !selected.members.length" @click="emit('export', selected.members.map(item => item.id))">{{ t('导出成员 JSON') }}</NButton>
              <NButton size="small" :disabled="busy" @click="openEditor(selected.collection)">{{ t('编辑') }}</NButton>
              <NPopconfirm :negative-text="t('取消')" :positive-text="t('删除')" @positive-click="emit('remove', selected.collection.id)">
                <template #trigger><NButton size="small" type="error" quaternary :disabled="busy">{{ t('删除合集') }}</NButton></template>
                {{ t('删除合集后，成员提示词仍保留在原来的库中。') }}
              </NPopconfirm>
            </div>
          </div>
        </section>

        <section class="collection-members">
          <header class="collection-members-heading">
            <div><h2>{{ t('合集成员') }} · {{ selected.members.length }}</h2><p class="section-label">{{ t('可拖动或用上下按钮调整顺序') }}</p></div>
            <NRadioGroup v-model:value="memberLayout" size="small" :aria-label="t('提示词视图')">
              <NRadioButton value="list">{{ t('列表') }}</NRadioButton>
              <NRadioButton value="cards">{{ t('卡片') }}</NRadioButton>
            </NRadioGroup>
          </header>
          <div v-if="!selected.members.length" class="empty collections-empty guided-empty">
            <p>{{ t('这个合集还没有提示词。') }}</p>
            <NButton size="small" :disabled="busy" @click="openEditor(selected.collection, true)">{{ t('添加成员') }}</NButton>
          </div>
          <div v-else class="collection-member-list" :class="{ 'collection-members-grid': memberLayout === 'cards' }">
            <article v-for="(item, index) in selected.members" :key="item.id" class="collection-member" :class="{ 'collection-drop-target': dropTarget === item.id }"
              @dragover="overMember($event, item.id)" @dragleave.self="dropTarget = ''" @drop.prevent="dropMember(index)">
              <span class="collection-member-order" :draggable="!busy" :title="t('拖动排序')" @dragstart="startDrag($event, item.id)" @dragend="endDrag">{{ String(index + 1).padStart(2, '0') }} <span aria-hidden="true">⠿</span></span>
              <button class="collection-member-open" type="button" :disabled="busy" @click="emit('openPrompt', item)">
                <CollectionMedia :item="item" />
                <span class="collection-member-copy">
                  <strong>{{ item.title }}</strong>
                  <small class="collection-member-meta"><span :class="item.kind">{{ libraryName(item.kind) }}</span> · {{ formatPromptNumber(item.number) }}<template v-if="promptModels(item).length"> · {{ promptModels(item).join(' · ') }}</template></small>
                </span>
              </button>
              <div class="actions collection-member-actions">
                <NButton size="tiny" :disabled="busy || index === 0" :aria-label="t('上移')" :title="t('上移')" @click="moveMember(item.id, index - 1)">↑</NButton>
                <NButton size="tiny" :disabled="busy || index === selected.members.length - 1" :aria-label="t('下移')" :title="t('下移')" @click="moveMember(item.id, index + 1)">↓</NButton>
                <NButton size="tiny" :disabled="busy" @click="copyMembers([item])">{{ t('复制') }}</NButton>
                <NPopconfirm :negative-text="t('取消')" :positive-text="t('移出')" @positive-click="updateMembers(selected.members.filter(member => member.id !== item.id).map(member => member.id))">
                  <template #trigger><NButton size="tiny" quaternary :disabled="busy">{{ t('移出') }}</NButton></template>
                  {{ t('从合集移出这条提示词？原条目仍会保留。') }}
                </NPopconfirm>
              </div>
            </article>
          </div>
        </section>
      </div>
      <div v-else class="empty collections-empty guided-empty">
        <strong>{{ t('还没有合集') }}</strong>
        <p>{{ t('创建一个合集，集中浏览相关的提示词和变体。') }}</p>
        <NButton size="small" :disabled="busy" @click="openEditor()">{{ t('＋ 新建合集') }}</NButton>
      </div>
    </div>

    <NModal :show="editing" preset="card" :title="adding ? t('添加成员') : editingId ? t('编辑合集') : t('新建合集')"
      class="app-modal collection-modal" :mask-closable="!busy" :close-on-esc="!busy" :closable="!busy" @update:show="value => editing = value">
      <form class="collections-form" @submit.prevent="save">
        <template v-if="!adding">
          <label>{{ t('合集名称') }} <NInput v-model:value="name" maxlength="80" :placeholder="t('例如：城市夜景系列')" :disabled="busy" /></label>
          <label>{{ t('合集说明') }} <NInput v-model:value="description" type="textarea" maxlength="1000" show-count
            :placeholder="t('介绍这个合集的主题或用途')" :disabled="busy" /></label>
        </template>
        <label>{{ t('合集成员') }} <NSelect v-model:value="promptIds" :options="promptOptions" :render-label="renderPromptLabel" multiple filterable
          :placeholder="t('搜索编号、标题或作者，选择相关提示词')" :disabled="busy" /></label>
        <div class="actions">
          <NButton :disabled="busy" @click="editing = false">{{ t('取消') }}</NButton>
          <NButton type="primary" attr-type="submit" :loading="busy" :disabled="!name.trim() || (adding && !promptIds.length)">{{ editingId ? t('保存合集') : t('创建合集') }}</NButton>
        </div>
      </form>
    </NModal>
  </section>
</template>
