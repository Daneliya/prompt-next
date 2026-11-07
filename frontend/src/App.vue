<script setup lang="ts">
import { computed, onMounted, onUnmounted, nextTick, ref, watch } from 'vue'
import {
  darkTheme, NAlert, NButton, NCheckbox, NCheckboxGroup, NConfigProvider, NInput, NModal, NPopconfirm, NPopover,
  NRadioButton, NRadioGroup, NSelect, NTag,
} from 'naive-ui'
import { PromptService } from '../bindings/github.com/Daneliya/prompt-next/internal/desktop'
import { Events } from '@wailsio/runtime'
import { formatPromptNumber, type Kind, type Prompt } from './prompt'
import { countLabels, promptModels, usePromptFilters } from './composables/usePromptFilters'
import { imageURL, readImage, useEditorImages } from './composables/useEditorImages'
import { useBackupImport } from './composables/useBackupImport'
import ImageCleanupDialog from './components/ImageCleanupDialog.vue'
import AppIcon from './components/AppIcon.vue'
import DistributionChart from './components/DistributionChart.vue'
import ImageTrend from './components/ImageTrend.vue'
import { desktopTheme } from './desktopTheme'
import './workspace.css'

type Theme = 'system' | 'light' | 'dark'
const libraries = [
  { kind: 'art', label: 'AI 绘画提示词库', icon: 'art', description: '画面描述、风格与生成参数' },
  { kind: 'persona', label: '大肥鱼人设库', icon: 'fish', description: '角色设定与助手指令' },
] as const

const prompts = ref<Prompt[]>([])
const kind = ref<Kind>('art')
const homeOpen = ref(true)
type Layout = 'cards' | 'list'
const layout = ref<Layout>('cards')
const viewFieldOptions = [
  { value: 'title', label: '标题' },
  { value: 'body', label: '提示词' },
  { value: 'description', label: '说明' },
  { value: 'models', label: 'AI 模型' },
  { value: 'category', label: '分类' },
  { value: 'tags', label: '标签' },
  { value: 'number', label: '编号' },
  { value: 'created', label: '创建时间' },
  { value: 'updated', label: '更新时间' },
]
const defaultViewFields = ['title', 'body', 'models', 'category', 'number']
const viewFields = ref<Record<Layout, string[]>>({
  cards: [...defaultViewFields],
  list: [...defaultViewFields],
})
const visibleFields = computed(() => new Set(viewFields.value[layout.value]))

const models = ref<string[]>([])
const customModels = ref<string[]>([])
const newModel = ref('')
const editingModel = ref('')
const modelName = ref('')
const configuredCategories = ref<Record<Kind, string[]>>({ art: [], persona: [] })
const settingsCategoryKind = ref<Kind>('art')
const newCategory = ref('')
const customCategories = ref<Record<Kind, string[]>>({ art: [], persona: [] })
const editingCategory = ref('')
const categoryName = ref('')
const tagName = ref('')
const editingTag = ref('')
const batchMode = ref(false)
const batchIds = ref<string[]>([])
const batchCategory = ref<string | null>(null)
const undoBatch = ref('')
const searchInput = ref<InstanceType<typeof NInput>>()
const appVersion = ref('')
const cleanupFiles = ref<Awaited<ReturnType<typeof PromptService.ListUnreferencedImages>>>()
const presetName = ref('')
const activePreset = ref<string | null>(null)
const presets = ref<Awaited<ReturnType<typeof PromptService.VariablePresets>>>({})
let menuUnsubscribe: (() => void)[] = []

const modelQuery = ref('')
const modelPickerOpen = ref(false)
const stats = ref<Awaited<ReturnType<typeof PromptService.StorageStats>>>()
const sortOptions = [
  { label: '最近更新', value: 'updated' },
  { label: '最新创建', value: 'created' },
  { label: '最早创建', value: 'oldest' },
  { label: '按标题', value: 'title' },
]
const selectedId = ref('')
const draft = ref<Prompt | null>(null)
const tagsText = ref('')
const initialDraft = ref('')
const values = ref<Record<string, string>>(Object.create(null))
const fillVariables = ref(false)
const settingsOpen = ref(false)
const settingsSections = [
  { key: 'appearance', label: '外观', icon: 'settings' },
  { key: 'models', label: 'AI 模型', icon: 'art' },
  { key: 'categories', label: '分类', icon: 'list' },
  { key: 'tags', label: '标签', icon: 'list' },
  { key: 'data', label: '数据管理', icon: 'disk' },
  { key: 'about', label: '关于', icon: 'persona' },
] as const
const settingsSection = ref<(typeof settingsSections)[number]['key']>('appearance')
const discardOpen = ref(false)
const theme = ref<Theme>('system')
const media = window.matchMedia('(prefers-color-scheme: dark)')
const systemDark = ref(media.matches)
const loading = ref(true)
const busy = ref(false)
const { draftImages, dragFrom, dragOver, imageInput, imagePreview, clearDraftImages,
  removeDraftImage, moveImage, startDrag, overImage, dropImage, endDrag, addImages } =
  useEditorImages(busy, message => notify(message, 'error'))
const notice = ref<{ text: string; type: 'success' | 'error' }>()
let noticeTimer: ReturnType<typeof setTimeout> | undefined
let nextAction: (() => void) | undefined

const isDark = computed(() => theme.value === 'dark' || (theme.value === 'system' && systemDark.value))
const themeOverrides = computed(() => desktopTheme(isDark.value))
const library = computed(() => libraries.find((item) => item.kind === kind.value)!)
const libraryItems = computed(() => prompts.value.filter((item) => item.kind === kind.value))
const { sort, modelFilter, tagFilter, category, favoritesOnly, query, results } = usePromptFilters(libraryItems)
const groupedTags = computed(() => countLabels(libraryItems.value.flatMap(item => item.tags ?? []))
  .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name, 'zh-CN')))
const managedTags = computed(() => countLabels(prompts.value.filter(item => item.kind === settingsCategoryKind.value)
  .flatMap(item => item.tags ?? [])).sort((a, b) => b.count - a.count || a.name.localeCompare(b.name, 'zh-CN')))
const presetOptions = computed(() => Object.keys(presets.value ?? {}).sort().map(name => ({ label: name, value: name })))
const visibleBatchIds = computed(() => {
  const selectedIds = new Set(batchIds.value)
  return results.value.filter(item => selectedIds.has(item.id)).map(item => item.id)
})
watch([kind, homeOpen], () => {
  batchMode.value = false
  batchIds.value = []
  batchCategory.value = null
})
watch(settingsCategoryKind, () => {
  editingCategory.value = ''
  editingTag.value = ''
})
const categories = computed(() => [...new Set(libraryItems.value.map((item) => item.category).filter(Boolean))]
  .sort((a, b) => a.localeCompare(b, 'zh-CN')))
const tagOptions = computed(() => options(libraryItems.value.flatMap((item) => item.tags ?? [])))
const modelOptions = computed(() => options([...models.value, ...prompts.value.flatMap(promptModels)]))
const categoryOptions = computed(() => options([
  ...configuredCategories.value[kind.value], ...categories.value,
]))
const editorModels = computed(() => [...new Set([...models.value, ...(draft.value?.models ?? [])])]
  .filter((name) => name.toLowerCase().includes(modelQuery.value.trim().toLowerCase())))
const recent = computed(() => prompts.value.slice(0, 6))
const artwork = computed(() => prompts.value.filter(item => item.kind === 'art'))
const categoryDistribution = computed(() => countLabels(artwork.value.map(item => item.category || '未分类')))
const modelDistribution = computed(() => countLabels(artwork.value.flatMap(item => {
  const names = promptModels(item)
  return names.length ? names : ['未选模型']
})))

const selected = computed(() => results.value.find((item) => item.id === selectedId.value))
const variables = computed(() => {
  const text = [selected.value?.body, selected.value?.english, selected.value?.negative].join('\n')
  return [...new Set([...text.matchAll(/\{\{\s*([^{}]+?)\s*\}\}/g)].map((match) => match[1]!.trim()).filter(Boolean))]
})
watch(() => selected.value?.id, (id, _, onCleanup) => {
  presets.value = {}
  resetVariables()
  if (!id) return
  let current = true
  onCleanup(() => { current = false })
  void PromptService.VariablePresets(id).then(result => {
    if (current) presets.value = result ?? {}
  }).catch(error => { if (current) showError(error) })
})
const dirty = computed(() => draft.value !== null && snapshotDraft() !== initialDraft.value)

function snapshotDraft() {
  return JSON.stringify([draft.value, tagsText.value, draftImages.value.map((item) => item.url)])
}

function notify(text: string, type: 'success' | 'error' = 'success') {
  notice.value = { text, type }
  clearTimeout(noticeTimer)
  if (type === 'success') noticeTimer = setTimeout(() => { notice.value = undefined }, 3000)
}

function showError(error: unknown) {
  notify(error instanceof Error ? error.message : String(error), 'error')
}

async function perform(action: () => Promise<void>) {
  if (busy.value) return
  busy.value = true
  try { await action() } catch (error) { showError(error) } finally { busy.value = false }
}

async function reload() {
  prompts.value = await PromptService.ListPrompts() ?? []
  stats.value = await PromptService.StorageStats()
}

function onSystemTheme(event: MediaQueryListEvent) {
  systemDark.value = event.matches
}

onMounted(async () => {
  media.addEventListener('change', onSystemTheme)
  menuUnsubscribe = [
    Events.On('prompt-next:new', () => {
      if (!loading.value && !settingsOpen.value && !pendingImport.value && !imagePreview.value) startEdit()
    }),
    Events.On('prompt-next:search', () => {
      if (draft.value || settingsOpen.value || pendingImport.value) return
      if (homeOpen.value) switchLibrary(kind.value)
      void nextTick(() => searchInput.value?.focus())
    }),
  ]
  try {
    await reload()
    const legacyLayout = localStorage.getItem('prompt-layout')
    if (legacyLayout === 'cards' || legacyLayout === 'list') {
      if (!await PromptService.GetLayout()) await PromptService.SetLayout(legacyLayout)
      localStorage.removeItem('prompt-layout')
    }
    await reloadPreferences()
    undoBatch.value = await PromptService.RecentDeleteBatch()
    appVersion.value = (await PromptService.AppInfo()).version
  } catch (error) { showError(error) } finally { loading.value = false }
})

onUnmounted(() => {
  menuUnsubscribe.forEach(unsubscribe => unsubscribe())
  media.removeEventListener('change', onSystemTheme)
  clearTimeout(noticeTimer)
  clearDraftImages()
})

function resetVariables() {
  values.value = Object.create(null)
  fillVariables.value = false
  presetName.value = ''
  activePreset.value = null
}

function navigate(action: () => void) {
  if (busy.value) return
  if (dirty.value) {
    nextAction = action
    discardOpen.value = true
    return
  }
  draft.value = null
  clearDraftImages()
  action()
}

function discard() {
  draft.value = null
  clearDraftImages()
  discardOpen.value = false
  nextAction?.()
  nextAction = undefined
}

function switchLibrary(next: Kind, id = '') {
  if (!homeOpen.value && kind.value === next && !id) return
  navigate(() => {
    homeOpen.value = false
    kind.value = next
    category.value = ''
    favoritesOnly.value = false
    query.value = ''
    modelFilter.value = null
    tagFilter.value = null
    selectedId.value = id
    resetVariables()
  })
}

function showHome() {
  navigate(() => { homeOpen.value = true })
}

function setLayout(next: string | number | boolean) {
  if (next !== 'cards' && next !== 'list') return
  void perform(async () => {
    await PromptService.SetLayout(next)
    layout.value = next
  })
}

function options(values: string[]) {
  return [...new Set(values.filter(Boolean))].map((value) => ({ label: value, value }))
}

function formatDate(value?: string) {
  if (!value) return '保存时自动记录'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  const pad = (part: number) => String(part).padStart(2, '0')
  const day = [String(date.getFullYear()).padStart(4, '0'), pad(date.getMonth() + 1), pad(date.getDate())].join('-')
  const time = [pad(date.getHours()), pad(date.getMinutes()), pad(date.getSeconds())].join(':')
  return day + ' ' + time
}

function formatBytes(bytes?: number) {
  if (bytes === undefined) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

async function reloadModels() {
  const [all, custom] = await Promise.all([PromptService.Models(), PromptService.CustomModels()])
  models.value = all ?? []
  customModels.value = custom ?? []
}

function addModel() {
  void perform(async () => {
    await PromptService.AddModel(newModel.value)
    await reloadModels()
    newModel.value = ''
    notify('模型已添加')
  })
}

function editModel(name: string) {
  editingModel.value = name
  modelName.value = name
}

function renameModel() {
  void perform(async () => {
    await PromptService.RenameModel(editingModel.value, modelName.value)
    await reloadModels()
    editingModel.value = ''
    notify('模型选项已更新')
  })
}

function deleteModel(name: string) {
  void perform(async () => {
    await PromptService.DeleteModel(name)
    await reloadModels()
    if (editingModel.value === name) editingModel.value = ''
    notify('已从模型选项移除')
  })
}

async function reloadCategories(library: Kind) {
  const [all, custom] = await Promise.all([PromptService.CategoryOptions(library), PromptService.CustomCategories(library)])
  configuredCategories.value[library] = all ?? []
  customCategories.value[library] = custom ?? []
}

function addCategory() {
  const library = settingsCategoryKind.value
  void perform(async () => {
    await PromptService.AddCategory(library, newCategory.value)
    await reloadCategories(library)
    newCategory.value = ''
    notify('分类已添加')
  })
}

function filter(nextCategory = '', nextFavorites = false) {
  navigate(() => {
    category.value = nextCategory
    favoritesOnly.value = nextFavorites
    tagFilter.value = null
    selectedId.value = results.value[0]?.id ?? ''
    resetVariables()
  })
}

function select(item: Prompt) {
  if (item.id === selectedId.value) return
  navigate(() => {
    selectedId.value = item.id
    resetVariables()
  })
}

function blankPrompt(): Prompt {
  return {
    id: '', number: 0, kind: kind.value, title: '', body: '', english: '', images: [], description: '', category: category.value,
    tags: [], favorite: false, author: '', negative: '', model: '', models: [], parameters: '',
    createdAt: '', updatedAt: '',
  }
}

function startEdit(item?: Prompt, copy = false) {
  navigate(() => {
    draft.value = item ? { ...item, english: item.english ?? '', models: [...promptModels(item)] } : blankPrompt()
    draftImages.value = item
      ? (item.images ?? []).map(name => ({ name, url: imageURL(item.number + '/' + name), sourceNumber: item.number }))
      : []
    modelQuery.value = ''
    modelPickerOpen.value = false
    if (copy) {
      draft.value.id = ''
      draft.value.number = 0
      draft.value.createdAt = ''
      draft.value.updatedAt = ''
      draft.value.favorite = false
      draft.value.title += ' · 副本'
    }
    tagsText.value = (draft.value.tags ?? []).join('，')
    initialDraft.value = snapshotDraft()
  })
}

function save() {
  if (!draft.value) return
  const data = { ...draft.value, tags: tagsText.value.split(/[,，、]/) }
  void perform(async () => {
    const attachments = await Promise.all(draftImages.value.map(async (item) => ({
      name: item.name,
      sourceNumber: item.sourceNumber ?? 0,
      data: item.file ? await readImage(item.file) : '',
    })))
    const saved = await PromptService.SaveOrderedImages(data, attachments)
    await reload()
    if (!results.value.some((item) => item.id === saved.id)) {
      category.value = ''
      favoritesOnly.value = false
      query.value = ''
      modelFilter.value = null
      tagFilter.value = null
    }
    draft.value = null
    clearDraftImages()
    selectedId.value = saved.id
    resetVariables()
    notify('已保存提示词')
  })
}

function toggleFavorite(item: Prompt) {
  void perform(async () => {
    await PromptService.SetFavorite(item.id, !item.favorite)
    await reload()
    if (!selected.value) selectedId.value = results.value[0]?.id ?? ''
    notify(item.favorite ? '已取消收藏' : '已收藏')
  })
}

function remove(ids: string[]) {
  void perform(async () => {
    undoBatch.value = await PromptService.DeletePrompts(ids)
    batchIds.value = []
    await reload()
    selectedId.value = results.value[0]?.id ?? ''
    resetVariables()
    notify(`已删除 ${ids.length} 条，24 小时内可撤销`)
  })
}

function undoDelete() {
  void perform(async () => {
    await PromptService.UndoDelete(undoBatch.value)
    undoBatch.value = await PromptService.RecentDeleteBatch()
    await reload()
    notify('已撤销删除')
  })
}

function batchFavorite(favorite: boolean) {
  void perform(async () => {
    await PromptService.BatchFavorite(visibleBatchIds.value, favorite)
    batchIds.value = []
    await reload()
    notify(favorite ? '选中条目已收藏' : '选中条目已取消收藏')
  })
}

function applyBatchCategory() {
  if (batchCategory.value === null) return
  void perform(async () => {
    await PromptService.BatchCategory(visibleBatchIds.value, kind.value, batchCategory.value ?? '')
    batchIds.value = []
    batchCategory.value = null
    await reload()
    notify('选中条目的分类已更新')
  })
}

function exportSelected() {
  void perform(async () => {
    if (await PromptService.ExportSelected(visibleBatchIds.value)) notify('选中条目已导出')
  })
}

function previewCleanup() {
  void perform(async () => {
    cleanupFiles.value = await PromptService.ListUnreferencedImages() ?? []
  })
}

function confirmCleanup(names: string[]) {
  void perform(async () => {
    const result = await PromptService.CleanUnreferencedImages(names)
    cleanupFiles.value = undefined
    await reload()
    const skipped = result.skipped ?? []
    notify(`已清理 ${result.removed} 张，释放 ${formatBytes(result.bytes)}${skipped.length ? `，跳过 ${skipped.length} 张` : ''}`)
  })
}

async function reloadPreferences() {
  const [saved, savedLayout, cards, list] = await Promise.all([
    PromptService.GetTheme(), PromptService.GetLayout(),
    PromptService.ViewFields('cards'), PromptService.ViewFields('list'),
  ])
  viewFields.value = { cards: cards ?? [...defaultViewFields], list: list ?? [...defaultViewFields] }
  layout.value = savedLayout === 'list' ? 'list' : 'cards'
  if (saved === 'dark' || saved === 'light' || saved === 'system') theme.value = saved
  await Promise.all([reloadModels(), reloadCategories('art'), reloadCategories('persona')])
}

function changeViewFields(fields: (string | number)[]) {
  const current = layout.value
  const names = fields.map(String)
  if (!names.length) {
    notify('至少保留一个展示字段', 'error')
    return
  }
  void perform(async () => {
    await PromptService.SetViewFields(current, names)
    viewFields.value[current] = names
  })
}

function updateCategory(current: string, deleting = false) {
  const targetKind = settingsCategoryKind.value
  const next = categoryName.value.trim()
  void perform(async () => {
    if (deleting) await PromptService.DeleteCategory(targetKind, current)
    else await PromptService.RenameCategory(targetKind, current, next)
    if (kind.value === targetKind && category.value === current) category.value = deleting ? '' : next
    editingCategory.value = ''
    await Promise.all([reloadCategories(targetKind), reload()])
    notify(deleting ? '已删除分类，原条目保留为未分类' : '分类已更新')
  })
}

function applyTagFilter(name: string) {
  navigate(() => {
    tagFilter.value = tagFilter.value === name ? null : name
    category.value = ''
    favoritesOnly.value = false
    query.value = ''
    modelFilter.value = null
    selectedId.value = ''
  })
}

function renameTag() {
  const targetKind = settingsCategoryKind.value
  const current = editingTag.value
  const name = tagName.value.trim()
  void perform(async () => {
    await PromptService.RenameTag(targetKind, current, name)
    if (kind.value === targetKind && tagFilter.value === current) tagFilter.value = name
    editingTag.value = ''
    await reload()
    notify('标签已更新，同名标签已合并')
  })
}

function loadPreset(name: string | null) {
  activePreset.value = name
  const saved = name ? presets.value?.[name] : undefined
  if (!name || !saved) return
  values.value = Object.assign(Object.create(null), saved)
  presetName.value = name
}

function savePreset(id: string, deleting = false) {
  const name = deleting ? activePreset.value : presetName.value.trim()
  if (!name) return
  void perform(async () => {
    if (deleting) await PromptService.DeleteVariablePreset(id, name)
    else {
      const current = Object.fromEntries(variables.value.map(key => [key, values.value[key] ?? '']))
      await PromptService.SetVariablePreset(id, name, current)
    }
    presets.value = await PromptService.VariablePresets(id) ?? {}
    activePreset.value = deleting ? null : name
    if (deleting) presetName.value = ''
    notify(deleting ? '预设已删除' : '预设已保存')
  })
}

function copyRendered(item: Prompt) {
  const fields = [[item.kind === 'art' ? '中文提示词' : '人设提示词', item.body], ['英文提示词', item.english], ['负向提示词', item.negative]]
  const output = fields.filter(([, value]) => value?.trim())
    .map(([label, value]) => `【${label}】\n${rendered(value ?? '')}`).join('\n\n')
  void perform(async () => {
    await PromptService.CopyText(output)
    notify('已复制全部提示词')
  })
}

function rendered(text: string) {
  return text.replace(/\{\{\s*([^{}]+?)\s*\}\}/g, (token: string, name: string) => {
    const value = values.value[name.trim()]
    return value?.trim() ? value : token
  })
}

function copyText(text: string) {
  void perform(async () => {
    await PromptService.CopyText(fillVariables.value ? rendered(text) : text)
    notify('已复制到剪贴板')
  })
}

const { pendingImport, restoreOpen, chooseImport, closeImport, confirmImport } = useBackupImport(busy, perform, async (result, replace) => {
  await reload()
  if (!selected.value) selectedId.value = results.value[0]?.id ?? ''
  if (replace) {
    draft.value = null
    clearDraftImages()
    resetVariables()
    batchIds.value = []
    await reloadPreferences()
    if (selected.value) presets.value = await PromptService.VariablePresets(selected.value.id) ?? {}
  }
  notify(replace
    ? `已恢复 ${result.added} 条，安全快照已保存在数据目录的 backups 文件夹`
    : `已导入 ${result.added} 条，跳过 ${result.duplicate} 条重复、${result.invalid} 条无效记录`)
})

function importFromSettings() {
  settingsOpen.value = false
  chooseImport()
}

function exportBackup() {
  void perform(async () => {
    if (await PromptService.ExportBackup()) notify('双库备份已保存')
  })
}

function exportMarkdown(item: Prompt) {
  void perform(async () => {
    if (await PromptService.ExportMarkdown(item)) notify('Markdown 已保存')
  })
}

function changeTheme(next: string | number | boolean) {
  if (next !== 'light' && next !== 'dark' && next !== 'system') return
  void perform(async () => {
    await PromptService.SetTheme(next)
    theme.value = next
    notify('主题设置已保存')
  })
}

function openDataFolder() {
  void perform(() => PromptService.OpenDataFolder())
}

function openGitHub() {
  void perform(() => PromptService.OpenGitHub())
}
</script>

<template>
  <NConfigProvider :theme="isDark ? darkTheme : null" :theme-overrides="themeOverrides">
    <main class="workspace" :class="{ 'is-dark': isDark, 'card-layout': layout === 'cards', 'no-detail': !selected }">
      <header class="app-header">
        <div class="brand"><img class="brand-icon" src="/icons/prompt-next.svg" alt=""><div><strong>Prompt Next</strong><small>本地工作台</small></div></div>
        <div class="actions">
          <NButton size="small" secondary :disabled="busy" @click="chooseImport">导入 JSON</NButton>
          <NButton size="small" secondary :disabled="busy" @click="exportBackup">备份</NButton>
          <NButton size="small" secondary @click="settingsOpen = true"><template #icon><AppIcon name="settings" /></template>设置</NButton>
        </div>
      </header>

      <aside class="sidebar" aria-label="提示词库导航">
        <div class="section-label">工作区</div>
        <button class="library-button" :class="{ active: homeOpen }" :aria-pressed="homeOpen" @click="showHome">
          <span class="library-icon"><AppIcon name="home" /></span>工作台首页
        </button>
        <button v-for="item in libraries" :key="item.kind" class="library-button" :class="{ active: !homeOpen && kind === item.kind }" :aria-pressed="!homeOpen && kind === item.kind" @click="switchLibrary(item.kind)">
          <span class="library-icon" :class="item.kind"><AppIcon :name="item.icon" /></span>
          <span>{{ item.label }}</span>
        </button>
        <template v-if="!homeOpen">
        <div class="section-label filters-label">当前库</div>
        <button class="filter-button" :class="{ active: !category && !favoritesOnly && !tagFilter }" @click="filter()">全部<span>{{ libraryItems.length }}</span></button>
        <button class="filter-button" :class="{ active: favoritesOnly }" @click="filter('', true)">收藏<span>{{ libraryItems.filter(item => item.favorite).length }}</span></button>
        <details v-if="groupedTags.length" class="tag-groups">
          <summary class="section-label filters-label">按标签聚合</summary>
          <button v-for="tag in groupedTags" :key="tag.name" class="filter-button" :class="{ active: tagFilter === tag.name }" @click="applyTagFilter(tag.name)">
            {{ tag.name }}<span>{{ tag.count }}</span>
          </button>
        </details>
        <div class="section-label filters-label">分类</div>
        <div v-if="!categories.length" class="no-categories">保存时可填写分类</div>
        <button v-for="name in categories" :key="name" class="filter-button" :class="{ active: category === name }" @click="filter(name)">{{ name }}</button>
        </template>
      </aside>

      <section v-if="homeOpen" class="home-pane" aria-label="工作台首页">
        <div class="hero">
          <div><p>离线创作工作台</p><h1>Prompt Next</h1><p>整理绘画提示词、参考图和角色设定。</p>
            <div class="actions"><NButton type="primary" @click="switchLibrary('art')">进入 AI 绘画库</NButton><NButton @click="switchLibrary('art'); startEdit()">新建绘画提示词</NButton></div>
          </div>
          <img class="hero-logo" src="/icons/prompt-next.svg" alt="Prompt Next Logo">
        </div>
        <div class="overview">
          <button v-for="item in libraries" :key="item.kind" class="stat-card" @click="switchLibrary(item.kind)">
            <span class="library-icon" :class="item.kind"><AppIcon :name="item.icon" /></span>
            <span>{{ item.label }}</span><strong>{{ prompts.filter(p => p.kind === item.kind).length }} 条</strong>
          </button>
          <div class="stat-card"><span class="library-icon"><AppIcon name="disk" /></span>
            <span>数据库占用</span><strong>{{ formatBytes(stats?.dbBytes) }}</strong>
          </div>
          <div class="stat-card"><span class="library-icon art"><AppIcon name="art" /></span>
            <span>图片占用 · {{ stats?.imageFiles ?? 0 }} 张</span><strong>{{ formatBytes(stats?.imageBytes) }}</strong>
          </div>
        </div>
        <div class="dashboard-charts">
          <ImageTrend :days="stats?.recentImages ?? []" />
          <DistributionChart title="绘画分类" subtitle="按绘画条目统计" :items="categoryDistribution" />
          <DistributionChart title="AI 模型" subtitle="多选模型分别计数" :items="modelDistribution" />
        </div>
        <div class="home-section">
          <h2>最近更新</h2>
          <p v-if="!recent.length" class="section-label">还没有提示词</p>
          <button v-for="item in recent" :key="item.id" class="recent-row" @click="switchLibrary(item.kind === 'art' ? 'art' : 'persona', item.id)">
            <span>{{ item.title }}</span><small>{{ item.kind === 'art' ? 'AI 绘画' : '人设' }} · {{ formatDate(item.updatedAt) }}</small>
          </button>
        </div>
        <div v-if="stats" class="data-path"><strong>本地共占用 {{ formatBytes(stats.dbBytes + stats.imageBytes) }}</strong><code>{{ stats.directory }}</code></div>
      </section>

      <section v-if="!homeOpen" class="list-pane" aria-label="提示词列表">
        <div class="pane-heading">
          <p class="section-label">{{ library.description }}</p>
          <div class="list-heading"><h1>{{ library.label }}</h1><NButton type="primary" size="small" :disabled="busy || loading" @click="startEdit()">＋ 新建</NButton></div>
          <NInput ref="searchInput" v-model:value="query" clearable :disabled="!!draft" placeholder="搜索标题、正文、编号、参数…" aria-label="搜索提示词"><template #prefix><AppIcon name="search" class="search-icon" /></template></NInput>
          <div class="filters">
            <NSelect v-model:value="sort" size="small" :options="sortOptions" aria-label="排序" />
            <NSelect v-model:value="tagFilter" size="small" :options="tagOptions" clearable filterable placeholder="标签" aria-label="按标签筛选" />
            <NSelect v-if="kind === 'art'" v-model:value="modelFilter" size="small" :options="modelOptions" clearable filterable placeholder="模型" aria-label="按模型筛选" />
          </div>
          <div class="list-controls">
            <div class="result-count">{{ tagFilter || category || (favoritesOnly ? '收藏' : '全部') }} · {{ results.length }} 条</div>
            <NButton size="small" :disabled="busy || !!draft" @click="batchMode = !batchMode; batchIds = []">{{ batchMode ? '结束多选' : '多选' }}</NButton>
            <NRadioGroup size="small" :value="layout" @update:value="setLayout">
              <NRadioButton value="cards"><AppIcon class="ui-icon" name="cards" />卡片</NRadioButton>
              <NRadioButton value="list"><AppIcon class="ui-icon" name="list" />列表</NRadioButton>
            </NRadioGroup>
            <NPopover trigger="click" placement="bottom-end">
              <template #trigger><NButton size="small" :disabled="busy">展示字段</NButton></template>
              <div class="view-fields-picker">
                <strong>{{ layout === 'cards' ? '卡片' : '列表' }}展示字段</strong>
                <NCheckboxGroup :value="viewFields[layout]" :disabled="busy" @update:value="changeViewFields">
                  <div class="view-field-grid">
                    <NCheckbox v-for="field in viewFieldOptions" :key="field.value" :value="field.value" :label="field.label" />
                  </div>
                </NCheckboxGroup>
                <NButton size="tiny" :disabled="busy" @click="changeViewFields(defaultViewFields)">恢复默认</NButton>
              </div>
            </NPopover>
          </div>
        </div>
        <div v-if="batchMode" class="actions batch-actions">
          <NButton size="small" :disabled="busy || !results.length" @click="batchIds = visibleBatchIds.length === results.length ? [] : results.map(item => item.id)">{{ visibleBatchIds.length === results.length ? '取消全选' : '全选当前结果' }}</NButton>
          <span>已选 {{ visibleBatchIds.length }} 条</span>
          <NButton size="small" :disabled="busy || !visibleBatchIds.length" @click="batchFavorite(true)">收藏</NButton>
          <NButton size="small" :disabled="busy || !visibleBatchIds.length" @click="batchFavorite(false)">取消收藏</NButton>
          <NSelect v-model:value="batchCategory" size="small" class="batch-category" placeholder="目标分类" :options="[{ label: '未分类', value: '' }, ...configuredCategories[kind].map(name => ({ label: name, value: name }))]" />
          <NButton size="small" :disabled="busy || !visibleBatchIds.length || batchCategory === null" @click="applyBatchCategory">改分类</NButton>
          <NButton size="small" :disabled="busy || !visibleBatchIds.length" @click="exportSelected">导出选中</NButton>
          <NPopconfirm negative-text="取消" positive-text="删除" @positive-click="remove(visibleBatchIds)">
            <template #trigger><NButton size="small" type="error" :disabled="busy || !visibleBatchIds.length">删除选中</NButton></template>
            确定删除这 {{ visibleBatchIds.length }} 条提示词？
          </NPopconfirm>
        </div>
        <div v-if="loading" class="empty list-empty">正在加载提示词…</div>
        <div v-else-if="!results.length" class="empty list-empty">{{ query ? '没有匹配的提示词' : '这里还没有提示词' }}</div>
        <NCheckboxGroup v-model:value="batchIds" :class="{ gallery: layout === 'cards' }">
        <div v-for="item in results" :key="item.id" class="prompt-entry">
        <NCheckbox v-if="batchMode" :value="item.id" class="batch-check" :aria-label="'选中 ' + item.title" />
        <button class="prompt-row" :class="{ active: item.id === selectedId }" :aria-pressed="item.id === selectedId" :aria-label="item.title" @click="select(item)">
          <div v-if="layout === 'cards'" class="card-cover" :class="item.kind">
            <img v-if="item.images?.length" :src="imageURL(item.number + '/' + item.images[0]!, true)" alt="" loading="lazy">
            <img v-else class="cover-avatar" :src="'/icons/' + item.kind + '-avatar.svg'" alt="" loading="lazy">
            <span v-if="item.images?.length" class="image-count">{{ item.images.length }} 张</span>
          </div>
          <span v-if="visibleFields.has('title')" class="row-title" :title="item.title">{{ item.title }} <span v-if="item.favorite" class="star">★</span></span>
          <span v-if="visibleFields.has('body') && (item.body || item.english)" class="row-summary">{{ item.body || item.english }}</span>
          <span v-if="visibleFields.has('description') && item.description" class="row-summary">{{ item.description }}</span>
          <span v-if="visibleFields.has('models') && promptModels(item).length" class="row-category" :title="promptModels(item).join(' · ')">{{ promptModels(item).join(' · ') }}</span>
          <span v-if="visibleFields.has('category') && item.category" class="row-category">{{ item.category }}</span>
          <span v-if="visibleFields.has('tags') && item.tags?.length" class="row-category">{{ item.tags.join(' · ') }}</span>
          <span v-if="visibleFields.has('number') || visibleFields.has('created') || visibleFields.has('updated')" class="row-meta">
            <span v-if="visibleFields.has('number')" class="row-number">{{ formatPromptNumber(item.number) }}</span>
            <span v-if="visibleFields.has('created') || visibleFields.has('updated')" class="row-times">
              <span v-if="visibleFields.has('created')">创建 {{ formatDate(item.createdAt) }}</span>
              <span v-if="visibleFields.has('updated')">更新 {{ formatDate(item.updatedAt) }}</span>
            </span>
          </span>
        </button>
        </div>
        </NCheckboxGroup>
      </section>

      <section v-if="!homeOpen && (selected || layout === 'list')" class="detail-pane" aria-label="提示词详情">
        <article v-if="selected">
          <div class="detail-heading">
            <div><p class="section-label">{{ library.label }} · {{ formatPromptNumber(selected.number) }}</p><h2>{{ selected.title }}</h2></div>
            <div class="actions">
              <NButton size="small" :disabled="busy" @click="toggleFavorite(selected)">{{ selected.favorite ? '★ 已收藏' : '☆ 收藏' }}</NButton>
              <NButton size="small" :disabled="busy" @click="startEdit(selected)">编辑</NButton>
              <NButton size="small" type="primary" :disabled="busy" @click="copyText(selected.body || selected.english)">{{ selected.kind === 'art' ? (selected.body ? '复制中文' : '复制英文') : '复制提示词' }}</NButton>
              <NButton v-if="layout === 'cards'" size="small" @click="selectedId = ''">关闭</NButton>
            </div>
          </div>
          <div class="metadata">
            <NTag v-if="selected.category" type="info" size="small">{{ selected.category }}</NTag>
            <NTag v-for="tag in selected.tags" :key="tag" size="small">{{ tag }}</NTag>
            <span v-if="selected.author">作者 · {{ selected.author }}</span>
            <NTag v-for="model in promptModels(selected)" :key="'model:' + model" size="small" type="info">{{ model }}</NTag>
          </div>
          <p class="entry-dates">创建 {{ formatDate(selected.createdAt) }} · 更新 {{ formatDate(selected.updatedAt) }}</p>
          <div v-if="selected.images?.length" class="attached-images">
            <button v-for="name in selected.images" :key="name" type="button" aria-label="预览图片" @click="imagePreview = imageURL(selected.number + '/' + name)">
              <img :src="imageURL(selected.number + '/' + name)" alt="参考图" loading="lazy">
            </button>
          </div>
          <p v-if="selected.description" class="description">{{ selected.description }}</p>
          <div v-if="variables.length" class="variable-panel">
            <div class="block-heading"><strong>模板变量</strong><NButton size="tiny" secondary @click="fillVariables = !fillVariables">{{ fillVariables ? '收起' : '填写并预览' }}</NButton></div>
            <div v-if="fillVariables" class="fields">
              <div class="actions preset-actions">
                <NSelect :value="activePreset" :options="presetOptions" clearable placeholder="载入变量预设" @update:value="loadPreset" />
                <NInput v-model:value="presetName" maxlength="60" placeholder="预设名称" />
                <NButton size="small" :disabled="busy || !presetName.trim()" @click="savePreset(selected.id)">保存</NButton>
                <NButton size="small" :disabled="busy || !activePreset" @click="savePreset(selected.id, true)">移除</NButton>
              </div>
              <label v-for="name in variables" :key="name">{{ name }}<NInput v-model:value="values[name]" :placeholder="name" /></label>
            </div>
          </div>
          <div v-if="selected.body" class="prompt-block">
            <div class="block-heading"><strong>{{ selected.kind === 'art' ? '中文提示词' : '人设提示词' }}</strong><span v-if="fillVariables">替换预览</span></div>
            <pre>{{ fillVariables ? rendered(selected.body) : selected.body }}</pre>
          </div>
          <div v-if="selected.english" class="prompt-block">
            <div class="block-heading"><strong>英文提示词</strong><NButton size="tiny" secondary :disabled="busy" @click="copyText(selected.english)">复制英文</NButton></div>
            <pre>{{ fillVariables ? rendered(selected.english) : selected.english }}</pre>
          </div>
          <div v-if="selected.kind === 'art' && selected.negative" class="prompt-block">
            <div class="block-heading"><strong>负向提示词</strong><NButton size="tiny" secondary :disabled="busy" @click="copyText(selected.negative)">复制负向</NButton></div>
            <pre>{{ fillVariables ? rendered(selected.negative) : selected.negative }}</pre>
          </div>
          <div v-if="selected.parameters" class="prompt-block"><strong>参数备注</strong><pre>{{ selected.parameters }}</pre></div>
          <div class="actions detail-footer">
            <NButton v-if="selected.kind === 'art'" size="small" :disabled="busy" @click="copyRendered(selected)">复制全部提示词</NButton>
            <NButton size="small" :disabled="busy" @click="startEdit(selected, true)">复制为新条目</NButton>
            <NButton size="small" :disabled="busy" @click="exportMarkdown(selected)">导出 Markdown</NButton>
            <NPopconfirm negative-text="取消" positive-text="删除" :positive-button-props="{ type: 'error' }" @positive-click="remove([selected.id])">
              <template #trigger><NButton size="small" quaternary type="error" :disabled="busy">删除</NButton></template>
              确定删除「{{ selected.title }}」？
            </NPopconfirm>
          </div>
        </article>

        <div v-else class="empty detail-empty">
          <div class="empty-icon"><AppIcon class="ui-icon" :name="library.icon" /></div><h2>{{ library.label }}</h2>
          <p>{{ loading ? '正在加载本地数据库' : '从列表选择一条提示词，或创建自己的记录。' }}</p>
          <div v-if="!loading" class="actions"><NButton type="primary" @click="startEdit()">新建提示词</NButton><NButton v-if="kind === 'persona'" @click="chooseImport">导入旧人设</NButton></div>
        </div>
      </section>

      <footer class="status"><span class="status-dot"></span> 本地数据 <span class="status-divider">·</span> 共 {{ prompts.length }} 条提示词 <NButton v-if="undoBatch" size="tiny" text :disabled="busy" @click="undoDelete">撤销上次删除</NButton></footer>
      <NAlert v-if="notice" class="notice" :type="notice.type" closable @close="notice = undefined">{{ notice.text }}</NAlert>

      <NModal :show="!!draft && !discardOpen" preset="card" :title="draft?.id ? '编辑提示词' : '新建提示词'" class="editor-modal" :mask-closable="false" :close-on-esc="!busy && !imagePreview && !modelPickerOpen" :closable="!busy" @update:show="navigate(() => {})">
        <form v-if="draft" id="prompt-editor-form" class="editor" @submit.prevent="save">
          <label>提示词标题 <NInput v-model:value="draft.title" placeholder="输入标题" /></label>
          <label>分类 <NSelect :value="draft.category || null" :options="categoryOptions" clearable filterable placeholder="选择分类，可在设置中添加" @update:value="value => draft && (draft.category = value ?? '')" /></label>
          <label>标签 <NInput v-model:value="tagsText" placeholder="使用逗号分隔" /></label>
          <template v-if="draft.kind === 'art'">
            <div class="editor-field">
              <span id="model-field-label">AI 模型</span>
              <NPopover v-model:show="modelPickerOpen" trigger="click" placement="bottom-start" :show-arrow="false" class="model-picker">
                <template #trigger>
                  <button class="model-trigger" type="button" :aria-expanded="modelPickerOpen" aria-labelledby="model-field-label" :disabled="busy">
                    <span v-if="!draft.models?.length" class="model-placeholder">选择一个或多个模型</span>
                    <NTag v-for="model in draft.models" :key="model" size="small"><span class="model-name" :title="model">{{ model }}</span></NTag>
                    <span class="model-chevron" aria-hidden="true">⌄</span>
                  </button>
                </template>
                <NInput v-model:value="modelQuery" clearable size="small" placeholder="查找模型" aria-label="查找模型" />
                <NCheckboxGroup :value="draft.models ?? []" @update:value="names => draft && (draft.models = names.map(String))">
                  <div class="model-option-grid">
                    <NCheckbox v-for="name in editorModels" :key="name" :value="name"><span class="model-name" :title="name">{{ name }}</span></NCheckbox>
                  </div>
                </NCheckboxGroup>
                <p v-if="!editorModels.length" class="model-placeholder">没有匹配的模型</p>
                <div class="actions model-picker-footer"><NButton size="small" type="primary" @click="modelPickerOpen = false">完成</NButton></div>
              </NPopover>
            </div>
            <div>
              <div class="block-heading"><strong>参考图片</strong><NButton size="small" secondary :disabled="busy" @click="imageInput?.click()">添加图片</NButton></div>
              <input ref="imageInput" class="hidden" type="file" accept="image/png,image/jpeg,image/gif,image/webp" multiple @change="addImages">
              <p class="section-label">点击放大，拖动排序；首张作为卡片封面。每张最多 12 MB。</p>
              <div v-if="draftImages.length" class="attached-images editor-images">
                <div v-for="(item, index) in draftImages" :key="item.url" class="draft-image" :class="{ 'drop-target': dragOver === index && dragFrom !== index }" :draggable="!busy" @dragstart="startDrag($event, index)" @dragover="overImage($event, index)" @drop="dropImage($event, index)" @dragend="endDrag">
                  <button class="draft-image-preview" type="button" :aria-label="'放大第 ' + (index + 1) + ' 张参考图'" @click="imagePreview = item.url">
                    <img :src="item.url" :alt="'参考图 ' + (index + 1)" draggable="false">
                    <span v-if="index === 0" class="cover-badge">封面</span>
                  </button>
                  <div class="image-actions">
                    <button type="button" title="前移" aria-label="前移图片" :disabled="busy || index === 0" @click="moveImage(index, index - 1)">←</button>
                    <button type="button" title="后移" aria-label="后移图片" :disabled="busy || index === draftImages.length - 1" @click="moveImage(index, index + 1)">→</button>
                    <button type="button" title="移除" aria-label="移除图片" :disabled="busy" @click="removeDraftImage(index)">×</button>
                  </div>
                </div>
              </div>
            </div>
          </template>
          <label v-if="draft.kind === 'persona'">作者 <NInput v-model:value="draft.author" /></label>
          <label>{{ draft.kind === 'art' ? '中文提示词' : '人设提示词' }}
            <NInput v-model:value="draft.body" type="textarea" :autosize="{ minRows: 4, maxRows: 10 }" placeholder="可用双花括号标记变量，例如 {{角色名}}" />
          </label>
          <label v-if="draft.kind === 'art'">英文提示词 <NInput v-model:value="draft.english" type="textarea" :autosize="{ minRows: 4, maxRows: 10 }" /></label>
          <details class="extra-fields">
            <summary>更多字段</summary>
            <label>说明 <NInput v-model:value="draft.description" type="textarea" /></label>
            <template v-if="draft.kind === 'art'">
              <label>负向提示词 <NInput v-model:value="draft.negative" type="textarea" /></label>
              <label>参数备注 <NInput v-model:value="draft.parameters" type="textarea" /></label>
            </template>
          </details>
          <p class="entry-dates">创建 {{ formatDate(draft.createdAt) }} · 更新 {{ formatDate(draft.updatedAt) }}</p>
        </form>
        <template #footer>
          <div v-if="draft" class="actions editor-footer">
            <NButton :disabled="busy" @click="navigate(() => {})">取消</NButton>
            <NButton form="prompt-editor-form" attr-type="submit" type="primary" :loading="busy"
              :disabled="!draft.title.trim() || (!draft.body.trim() && (draft.kind !== 'art' || !draft.english.trim()))">保存</NButton>
          </div>
        </template>
      </NModal>

      <NModal :show="!!imagePreview" preset="card" title="图片预览" class="image-modal" @update:show="imagePreview = ''">
        <img v-if="imagePreview" :src="imagePreview" alt="参考图预览">
      </NModal>

      <NModal v-model:show="settingsOpen" preset="card" title="偏好设置" class="settings-modal" :class="{ 'is-dark': isDark }" :bordered="false" :mask-closable="false">
        <div class="settings-layout">
          <nav class="settings-nav" aria-label="设置分类">
            <button v-for="section in settingsSections" :key="section.key" type="button" :aria-pressed="settingsSection === section.key" @click="settingsSection = section.key">
              <AppIcon :name="section.icon" class="ui-icon" />{{ section.label }}
            </button>
          </nav>
          <section class="settings-content" :aria-label="settingsSections.find(section => section.key === settingsSection)?.label">
            <template v-if="settingsSection === 'appearance'">
              <div class="setting-field">
                <h3>软件主题</h3>
                <NRadioGroup :value="theme" :disabled="busy" @update:value="changeTheme"><NRadioButton value="light">浅色</NRadioButton><NRadioButton value="dark">深色</NRadioButton><NRadioButton value="system">跟随系统</NRadioButton></NRadioGroup>
              </div>
              <div class="setting-field">
                <h3>提示词视图</h3>
                <NRadioGroup :value="layout" @update:value="setLayout"><NRadioButton value="cards">卡片</NRadioButton><NRadioButton value="list">列表</NRadioButton></NRadioGroup>
              </div>
              <p class="setting-hint">主题和视图更改即时生效。</p>
            </template>
            <template v-else-if="settingsSection === 'models'">
              <div class="setting-field">
                <h3>添加 AI 绘画模型</h3>
                <form class="settings-model-form" @submit.prevent="addModel">
                  <NInput v-model:value="newModel" placeholder="输入模型名称" maxlength="100" aria-label="模型名称" />
                  <NButton attr-type="submit" type="primary" :disabled="!newModel.trim() || busy">添加</NButton>
                </form>
                <p class="setting-hint">添加后可在绘画条目的模型下拉框中选择。</p>
              </div>
              <div class="setting-field">
                <h3>自定义模型</h3>
                <p class="setting-hint">修改和删除只影响下拉选项，已有提示词中的模型名称仍保留。</p>
                <p v-if="!customModels.length" class="setting-hint">还没有自定义模型</p>
                <div v-for="name in customModels" :key="name" class="model-setting-row">
                  <form v-if="editingModel === name" class="settings-model-form" @submit.prevent="renameModel">
                    <NInput v-model:value="modelName" size="small" maxlength="100" aria-label="修改模型名称" />
                    <NButton size="small" attr-type="submit" type="primary" :disabled="busy || !modelName.trim()">保存</NButton>
                    <NButton size="small" @click="editingModel = ''">取消</NButton>
                  </form>
                  <template v-else>
                    <span>{{ name }}</span>
                    <div class="actions">
                      <NButton size="tiny" :disabled="busy" @click="editModel(name)">修改</NButton>
                      <NPopconfirm negative-text="取消" positive-text="删除" :positive-button-props="{ type: 'error' }" @positive-click="deleteModel(name)">
                        <template #trigger><NButton size="tiny" type="error" quaternary :disabled="busy">删除</NButton></template>
                        从模型选项删除「{{ name }}」？
                      </NPopconfirm>
                    </div>
                  </template>
                </div>
              </div>
              <div class="setting-field">
                <h3>内置模型</h3>
                <div class="metadata"><NTag v-for="model in models.filter(name => !customModels.includes(name))" :key="model" size="small">{{ model }}</NTag></div>
              </div>
            </template>
            <template v-else-if="settingsSection === 'categories'">
              <div class="setting-field">
                <h3>提示词库</h3>
                <NRadioGroup v-model:value="settingsCategoryKind">
                  <NRadioButton value="art">AI 绘画</NRadioButton>
                  <NRadioButton value="persona">人设</NRadioButton>
                </NRadioGroup>
              </div>
              <div class="setting-field">
                <h3>添加分类</h3>
                <form class="settings-model-form" @submit.prevent="addCategory">
                  <NInput v-model:value="newCategory" placeholder="输入分类名称" maxlength="60" aria-label="分类名称" />
                  <NButton attr-type="submit" type="primary" :disabled="!newCategory.trim() || busy">添加</NButton>
                </form>
                <p class="setting-hint">新建和编辑时可选择本库的分类。</p>
              </div>
              <div class="setting-field">
                <h3>自定义分类</h3>
                <p v-if="!customCategories[settingsCategoryKind].length" class="setting-hint">还没有自定义分类。</p>
                <div v-for="name in customCategories[settingsCategoryKind]" :key="name" class="actions management-row">
                  <template v-if="editingCategory === name">
                    <NInput v-model:value="categoryName" size="small" maxlength="60" @keyup.enter="updateCategory(name)" />
                    <NButton size="small" :disabled="busy || !categoryName.trim()" @click="updateCategory(name)">保存</NButton>
                    <NButton size="small" @click="editingCategory = ''">取消</NButton>
                  </template>
                  <template v-else>
                    <span>{{ name }}</span>
                    <NButton size="small" :disabled="busy" @click="editingCategory = name; categoryName = name">改名</NButton>
                    <NPopconfirm negative-text="取消" positive-text="删除" @positive-click="updateCategory(name, true)">
                      <template #trigger><NButton size="small" type="error" :disabled="busy">删除</NButton></template>
                      原有条目将保留并移至未分类。
                    </NPopconfirm>
                  </template>
                </div>
                <h3>内置分类</h3>
                <div class="metadata"><NTag v-for="name in configuredCategories[settingsCategoryKind].filter(item => !customCategories[settingsCategoryKind].includes(item))" :key="name" size="small">{{ name }}</NTag></div>
              </div>
            </template>
            <template v-else-if="settingsSection === 'tags'">
              <div class="setting-field">
                <h3>标签管理</h3>
                <NRadioGroup v-model:value="settingsCategoryKind">
                  <NRadioButton value="art">AI 绘画</NRadioButton>
                  <NRadioButton value="persona">人设</NRadioButton>
                </NRadioGroup>
                <p class="setting-hint">输入已有标签名即可合并，计数按条目统计。</p>
                <p v-if="!managedTags.length" class="setting-hint">当前库还没有标签。</p>
                <div v-for="tag in managedTags" :key="tag.name" class="actions management-row">
                  <template v-if="editingTag === tag.name">
                    <NInput v-model:value="tagName" size="small" maxlength="60" @keyup.enter="renameTag" />
                    <NButton size="small" :disabled="busy || !tagName.trim()" @click="renameTag">保存</NButton>
                    <NButton size="small" @click="editingTag = ''">取消</NButton>
                  </template>
                  <template v-else>
                    <span>{{ tag.name }} · {{ tag.count }} 条</span>
                    <NButton size="small" @click="editingTag = tag.name; tagName = tag.name">改名或合并</NButton>
                  </template>
                </div>
              </div>
            </template>
            <template v-else-if="settingsSection === 'data'">
              <div class="setting-field">
                <div class="block-heading">
                  <h3>应用数据目录</h3>
                  <NButton size="small" :disabled="busy" @click="openDataFolder">打开文件夹</NButton>
                </div>
                <code class="settings-path">{{ stats?.directory || '正在读取目录…' }}</code>
                <p class="setting-hint">数据库 {{ formatBytes(stats?.dbBytes) }} · 图片 {{ formatBytes(stats?.imageBytes) }}</p>
              </div>
              <div class="setting-field">
                <h3>导入和备份</h3>
                <div class="actions"><NButton :disabled="busy" @click="importFromSettings">导入 JSON</NButton><NButton :disabled="busy" @click="exportBackup">导出备份</NButton></div>
              </div>
              <div class="setting-field">
                <h3>图片清理</h3>
                <p class="setting-hint">先查看文件和占用，再确认清理。</p>
                <NButton :disabled="busy" @click="previewCleanup">查看未被引用的图片</NButton>
              </div>
            </template>
            <template v-else>
              <div class="settings-about">
                <img src="/icons/prompt-next.svg" alt="" width="40" height="40">
                <div><strong>Prompt Next</strong><p class="setting-hint">版本 {{ appVersion || "—" }} · Daneliya</p></div>
              </div>
              <div class="setting-field"><h3>GitHub</h3><NButton text type="primary" @click="openGitHub">github.com/Daneliya ↗</NButton></div>
            </template>
          </section>
        </div>
        <template #footer><div class="settings-footer"><NButton type="primary" @click="settingsOpen = false">完成</NButton></div></template>
      </NModal>

      <ImageCleanupDialog :files="cleanupFiles ?? undefined" :busy="busy" :format-bytes="formatBytes" @close="cleanupFiles = undefined" @confirm="confirmCleanup" />

      <NModal :show="!!pendingImport" preset="card" title="导入预览" class="app-modal" :mask-closable="!busy" :closable="!busy" :close-on-esc="!busy" @update:show="closeImport">
        <template v-if="pendingImport">
          <p>{{ pendingImport.name }}</p>
          <p>新增 {{ pendingImport.added }} 条 · 重复 {{ pendingImport.duplicate }} 条 · 无效 {{ pendingImport.invalid }} 条</p>
          <p>重复和无效记录会跳过，已有编辑会保留。</p>
          <div class="actions"><NButton :disabled="busy" @click="closeImport">关闭</NButton><NButton v-if="pendingImport.canRestore" type="warning" secondary :disabled="busy" @click="restoreOpen = true">覆盖恢复…</NButton><NButton type="primary" :disabled="!pendingImport.added" :loading="busy" @click="confirmImport()">确认导入</NButton></div>
        </template>
      </NModal>

      <NModal v-model:show="restoreOpen" preset="card" title="覆盖恢复双库？" class="app-modal" :mask-closable="!busy" :closable="!busy">
        <p>将替换双库中的提示词，并丢弃未保存的草稿。新版备份也会恢复设置。覆盖前会在应用数据目录的 backups 文件夹自动保存一份安全快照。</p>
        <div class="actions"><NButton :disabled="busy" @click="restoreOpen = false">取消</NButton><NButton type="error" :loading="busy" @click="confirmImport(true)">确认覆盖恢复</NButton></div>
      </NModal>

      <NModal v-model:show="discardOpen" preset="card" title="放弃未保存的修改？" class="app-modal">
        <p>当前草稿还没有保存。</p>
        <div class="actions"><NButton @click="discardOpen = false">继续编辑</NButton><NButton type="error" @click="discard">放弃修改</NButton></div>
      </NModal>
    </main>
  </NConfigProvider>
</template>
