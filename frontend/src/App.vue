<script setup lang="ts">
import { translateBackendError } from './backendErrors'
import { language, t } from './i18n'
import { computed, defineAsyncComponent, onMounted, onUnmounted, nextTick, ref, shallowRef, watch } from 'vue'
import {
  darkTheme, zhCN, enUS, dateZhCN, dateEnUS, NAlert, NButton, NCheckbox, NCheckboxGroup, NConfigProvider, NInput, NModal, NPopconfirm, NPopover,
  NRadioButton, NRadioGroup, NSelect, NTag,
} from 'naive-ui'
import { PromptService } from '../bindings/github.com/Daneliya/prompt-next/internal/desktop'
import { Events, System, Window } from '@wailsio/runtime'
import { formatPromptNumber, libraries, videoAsset, type Kind, type Prompt } from './prompt'
import PromptSource from './components/PromptSource.vue'
import PromptResults from './components/PromptResults.vue'
import PromptRow from './components/PromptRow.vue'
import { countLabels, promptModels, usePromptFilters } from './composables/usePromptFilters'
import { imageURL, readImage, useEditorImages, type DraftImage } from './composables/useEditorImages'
import { useDebouncedQuery } from './composables/useDebouncedQuery'
import { useStorageStats } from './composables/useStorageStats'
import { useBackupImport } from './composables/useBackupImport'
import ImageCleanupDialog from './components/ImageCleanupDialog.vue'
import PromptTrash from './components/PromptTrash.vue'
import PromptCollections from './components/PromptCollections.vue'
import PromptFold from './components/PromptFold.vue'
import PromptEditor from './components/PromptEditor.vue'
import ImagePreviewViewport from './components/ImagePreviewViewport.vue'
import VideoCover from './components/VideoCover.vue'
import AppIcon from './components/AppIcon.vue'
import WorkspaceHome from './components/WorkspaceHome.vue'
import WorkspaceSettings, { type SettingsSection } from './components/WorkspaceSettings.vue'
import { accentTokens } from './designTokens'
import { desktopTheme } from './desktopTheme'
import './workspace.css'

const AuthorDirectory = defineAsyncComponent(() => import('./components/AuthorDirectory.vue'))

type Theme = 'system' | 'light' | 'dark'
const isMac = System.IsMac() || /\bMacintosh\b/.test(navigator.userAgent)
const isWindows = System.IsWindows() || /\bWindows\b/.test(navigator.userAgent)
const windowMaximised = ref(false)
const maximiseLabel = computed(() => t(windowMaximised.value ? '还原窗口' : '最大化窗口'))

function runWindowAction(action: Promise<void>) {
  void action.catch(showError)
}

type Collection = Awaited<ReturnType<typeof PromptService.SaveCollection>>
const collections = ref<Collection[]>([])
const collectionsOpen = ref(false)
const collectionId = ref('')
const collectionSeed = ref('')
const collectionsPageKey = ref(0)

const prompts = shallowRef<Prompt[]>([])
const kind = ref<Kind>('text')
const homeOpen = ref(true)
type Author = NonNullable<Awaited<ReturnType<typeof PromptService.ListAuthors>>>[number]
const authors = ref<Author[]>([])
const authorsOpen = ref(false)
const authorDirectory = ref<InstanceType<typeof AuthorDirectory>>()
const authorDraft = ref<Author>()
const authorPage = ref<{ sourceName: string; previousId: string; previousScroll: number; previousPage: number; fromDirectory: boolean }>()
const authorKind = ref<Kind | ''>('')
const listPane = ref<HTMLElement>()
const promptResults = ref<InstanceType<typeof PromptResults>>()
const browsePage = ref(1)
const layoutOptions = [
  { value: 'cards', label: '卡片', icon: 'cards' },
  { value: 'list', label: '列表', icon: 'list' },
  { value: 'waterfall', label: '瀑布流', icon: 'waterfall' },
] as const
type Layout = (typeof layoutOptions)[number]['value']

function isLayout(value: unknown): value is Layout {
  return layoutOptions.some(option => option.value === value)
}
const libraryLayout = ref<Layout>('cards')
const authorLayout = ref<Layout>('cards')
const layout = computed(() => authorPage.value ? authorLayout.value : libraryLayout.value)
const viewFieldOptions = [
  { value: 'title', label: '标题' },
  { value: 'body', label: '提示词' },
  { value: 'description', label: '说明' },
  { value: 'models', label: 'AI 模型' },
  { value: 'category', label: '分类' },
  { value: 'tags', label: '标签' },
  { value: 'author', label: '原作者' },
  { value: 'number', label: '编号' },
  { value: 'created', label: '创建时间' },
  { value: 'updated', label: '更新时间' },
]
const defaultViewFields = ['title', 'body', 'models', 'category', 'number']
const waterfallFields = [...defaultViewFields, 'tags']
const personaFields = ['title', 'category', 'characters', 'description', 'number', 'created', 'updated']
const personaFieldOptions = viewFieldOptions
  .filter(field => personaFields.includes(field.value) || field.value === 'author')
  .map(field => field.value === 'description' ? { ...field, label: '简介' } : field)
  .concat({ value: 'characters', label: '字数' })
const viewFieldDefaults = {
  cards: defaultViewFields, list: defaultViewFields, waterfall: waterfallFields,
  'persona:cards': personaFields, 'persona:list': personaFields, 'persona:waterfall': personaFields,
}
type ViewFieldKey = keyof typeof viewFieldDefaults
const viewFields = ref<Partial<Record<ViewFieldKey, string[]>>>({})
const fieldKey = computed<ViewFieldKey>(() => kind.value === 'persona' ? `persona:${layout.value}` : layout.value)
const currentFields = computed(() => viewFields.value[fieldKey.value] ?? viewFieldDefaults[fieldKey.value])
const layoutLabel = computed(() => t(layoutOptions.find(option => option.value === layout.value)?.label))
const authorFields = [...defaultViewFields, 'description']
const visibleFields = computed(() => new Set(authorPage.value ? authorFields : currentFields.value))
const writtenLibrary = computed(() => authorPage.value ? !!authorKind.value && !hasMedia(authorKind.value) : !hasMedia(kind.value))

type ModelKind = 'art' | 'video'
type ModelCatalog = Awaited<ReturnType<typeof PromptService.Models>>
const modelCatalogs = ref<Partial<Record<ModelKind, ModelCatalog>>>({})
const settingsModelKind = ref<ModelKind>('art')
const settingsModels = computed(() => modelCatalogs.value[settingsModelKind.value])
const customModels = computed(() => settingsModels.value?.custom ?? [])

function modelsFor(library: string) {
  const catalog = library === 'art' || library === 'video' ? modelCatalogs.value[library] : undefined
  return [...(catalog?.builtIn ?? []), ...(catalog?.custom ?? [])]
}
const newModel = ref('')
const editingModel = ref('')
const modelName = ref('')
watch(settingsModelKind, () => {
  newModel.value = ''
  editingModel.value = ''
  modelName.value = ''
})
const configuredCategories = ref<Record<Kind, string[]>>({ text: [], art: [], video: [], persona: [] })
const settingsCategoryKind = ref<Kind>('text')
const newCategory = ref('')
const customCategories = ref<Record<Kind, string[]>>({ text: [], art: [], video: [], persona: [] })
const editingCategory = ref('')
const categoryName = ref('')
const tagName = ref('')
const editingTag = ref('')
const batchMode = ref(false)
const batchIds = ref<string[]>([])
const batchCategory = ref<string | null>(null)
type TrashEntry = NonNullable<Awaited<ReturnType<typeof PromptService.ListTrash>>>[number]
const trashOpen = ref(false)
const trashEntries = ref<TrashEntry[]>([])
type ImageDetail = NonNullable<Awaited<ReturnType<typeof PromptService.ImageDetails>>>[number]
type PreviewImage = { url: string; name: string; number?: number; file?: File }
const imageDetails = ref(new Map<string, ImageDetail>())
const previewImages = ref<PreviewImage[]>([])
const previewIndex = ref(0)
const previewPixels = ref<{ width: number; height: number }>()
const currentPreview = computed(() => previewImages.value[previewIndex.value])
const imagePreview = computed(() => currentPreview.value?.url ?? '')
const previewDetail = computed(() => {
  const item = currentPreview.value
  return item?.number ? imageDetails.value.get(item.number + '/' + item.name) : undefined
})
const previewBytes = computed(() => currentPreview.value?.file?.size ?? previewDetail.value?.bytes)
const previewDimensions = computed(() => previewDetail.value ?? previewPixels.value)
const previewFormat = computed(() => {
  const item = currentPreview.value
  return (previewDetail.value?.format || item?.file?.type.split('/')[1] || item?.name.split('.').pop() || '').toUpperCase()
})
watch(imagePreview, () => { previewPixels.value = undefined })
const searchInput = ref<InstanceType<typeof NInput>>()
const appVersion = ref('')
const cleanupFiles = ref<Awaited<ReturnType<typeof PromptService.ListUnreferencedImages>>>()
const presetName = ref('')
const activePreset = ref<string | null>(null)
const presets = ref<Awaited<ReturnType<typeof PromptService.VariablePresets>>>({})
let menuUnsubscribe: (() => void)[] = []

const modelQuery = ref('')
const modelPickerOpen = ref(false)
const sortOptions = computed(() => [
  { label: '最近更新', englishLabel: 'Updated', value: 'updated' },
  { label: '最新创建', value: 'created' },
  { label: '最早创建', value: 'oldest' },
  { label: '按标题', value: 'title' },
].map(option => ({ ...option, label: language.value === 'en-US' && option.englishLabel ? option.englishLabel : t(option.label) })))
const selectedId = ref('')
const draft = ref<Prompt | null>(null)
const draftCreated = ref<number | null>(null)
const draftUpdated = ref<number | null>(null)
type ChosenVideo = NonNullable<Awaited<ReturnType<typeof PromptService.ChooseVideoFiles>>>[number]
const draftVideoFiles = ref<ChosenVideo[]>([])
const videoDetails = ref(new Map<string, number>())
const videoPreview = ref<{ item: Prompt; index: number }>()
const videoPreviewError = ref(false)
const activeVideo = computed(() => videoPreview.value?.item.videos?.[videoPreview.value.index])
const draftVideoSource = ref(0)
const tagsText = ref('')
const initialDraft = ref('')
const values = ref<Record<string, string>>(Object.create(null))
const fillVariables = ref(false)
const settingsOpen = ref(false)
const dataFolderChange = ref<{ directory: string } | 'default'>()
const settingsSection = ref<SettingsSection>('appearance')
const { stats, refresh: refreshStats, invalidate: invalidateStats } = useStorageStats(
  computed(() => (homeOpen.value && !collectionsOpen.value) || (settingsOpen.value && settingsSection.value === 'data')),
  showError,
)
const discardOpen = ref(false)
const theme = ref<Theme>('system')
const media = window.matchMedia('(prefers-color-scheme: dark)')
const systemDark = ref(media.matches)
const loading = ref(true)
const busy = ref(false)
const effectImages = useEditorImages(busy, message => notify(message, 'error'), () => draftReferences.value)
const referenceImages = useEditorImages(busy, message => notify(message, 'error'), () => draftImages.value)
const draftImages = effectImages.draftImages
const draftReferences = referenceImages.draftImages
const canSaveDraft = computed(() => {
  const item = draft.value
  if (!item?.title.trim() || (!item.body.trim() && (!hasMedia(item.kind) || !item.english.trim()))) return false
  if (item.kind === 'art') return draftImages.value.length > 0
  if (item.kind === 'video') return !!item.videos?.length || draftVideoFiles.value.length > 0
  return true
})

function clearDraftImages() {
  effectImages.clearDraftImages()
  referenceImages.clearDraftImages()
}

const notice = ref<{ text: string; type: 'success' | 'error' }>()
let noticeTimer: ReturnType<typeof setTimeout> | undefined
let nextAction: (() => void) | undefined

const isDark = computed(() => theme.value === 'dark' || (theme.value === 'system' && systemDark.value))
const themeOverrides = computed(() => desktopTheme(isDark.value))
const workspaceTheme = computed(() => ({ '--accent': accentTokens(isDark.value).accent }))
const library = computed(() => libraries.find((item) => item.kind === kind.value)!)
const libraryItems = computed(() => prompts.value.filter((item) => item.kind === kind.value))
const { sort, modelFilter, tagFilter, category, favoritesOnly, query, results } = usePromptFilters(libraryItems)
const searchText = ref('')
const commitSearch = useDebouncedQuery(searchText, query)
const authorsByName = computed(() => new Map(authors.value.map(author => [author.sourceName, author])))
const promptsByAuthor = computed(() => {
  const grouped = new Map<string, Prompt[]>()
  for (const item of prompts.value) {
    const name = item.author?.trim()
    if (!name) continue
    const group = grouped.get(name)
    if (group) group.push(item)
    else grouped.set(name, [item])
  }
  return grouped
})
const activeAuthor = computed(() => authorsByName.value.get(authorPage.value?.sourceName ?? ''))
const authorPrompts = computed(() => promptsByAuthor.value.get(authorPage.value?.sourceName ?? '') ?? [])
const authorOptions = computed(() => authors.value.map(author => ({
  value: author.sourceName,
  label: author.name === author.sourceName ? author.name : `${author.name} (${author.sourceName})`,
})))
const authorCounts = computed(() => ({ text: 0, art: 0, video: 0, persona: 0, ...Object.fromEntries(
  countLabels(authorPrompts.value.map(item => item.kind)).map(({ name, count }) => [name, count]),
) }))
const authorItems = computed(() => authorPrompts.value.filter(item => !authorKind.value || item.kind === authorKind.value))
const { sort: authorSort, query: authorQuery, results: authorResults } = usePromptFilters(authorItems)
const authorSearchText = ref('')
const commitAuthorSearch = useDebouncedQuery(authorSearchText, authorQuery)
const displayedResults = computed(() => authorPage.value ? authorResults.value : results.value)
const activeQuery = computed(() => authorPage.value ? authorQuery.value : query.value)
function resultScroller() {
  if (authorsOpen.value) return authorDirectory.value?.scrollContainer()
  return (layout.value === 'list' ? promptResults.value?.scrollContainer() : undefined) ?? listPane.value
}

watch([kind, layout, activeQuery, sort, authorSort, category, modelFilter, tagFilter, favoritesOnly, authorKind], () => {
  browsePage.value = 1
  const scroller = resultScroller()
  if (scroller) scroller.scrollTop = 0
}, { flush: 'sync' })
watch(browsePage, () => {
  if (listPane.value) listPane.value.scrollTop = 0
})

const authorName = computed(() => activeAuthor.value?.name ?? authorPage.value?.sourceName ?? '')
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
const modelOptions = computed(() => options([...modelsFor(kind.value), ...libraryItems.value.flatMap(promptModels)]))
const categoryOptions = computed(() => options([
  ...configuredCategories.value[kind.value], ...categories.value,
]))
const editorModels = computed(() => [...new Set([...modelsFor(draft.value?.kind ?? kind.value), ...(draft.value?.models ?? [])])]
  .filter((name) => name.toLowerCase().includes(modelQuery.value.trim().toLowerCase())))
const selected = computed(() => displayedResults.value.find((item) => item.id === selectedId.value))
watch(selected, item => {
  if (item) void loadImageDetails(item.number, [...(item.images ?? []), ...(item.referenceImages ?? [])])
})
const relatedCollections = computed(() => collections.value.filter(item => item.promptIds?.includes(selectedId.value)))
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
  return JSON.stringify([draft.value, draftCreated.value, draftUpdated.value, tagsText.value, draftVideoFiles.value.map(file => file.id), draftImages.value.map(item => item.url), draftReferences.value.map(item => item.url)])
}

function notify(text: string, type: 'success' | 'error' = 'success') {
  notice.value = { text, type }
  clearTimeout(noticeTimer)
  if (type === 'success') noticeTimer = setTimeout(() => { notice.value = undefined }, 3000)
}

function showError(error: unknown) {
  const message = error instanceof Error ? error.message : String(error)
  notify(message.includes('unknown bound method id') ? t('前端与桌面程序版本不一致，请退出并重新启动最新版。') : language.value === 'en-US' ? translateBackendError(t(message)) : message, 'error')
}

async function perform(action: () => Promise<void>) {
  if (busy.value) return
  busy.value = true
  try { await action() } catch (error) { showError(error) } finally { busy.value = false }
}

async function reload() {
  const [items, profiles] = await Promise.all([PromptService.ListPrompts(), PromptService.ListAuthors()])
  prompts.value = items ?? []
  authors.value = profiles ?? []
}

function replacePrompts(items: Prompt[], remove: (item: Prompt) => boolean) {
  prompts.value = [...prompts.value.filter(item => !remove(item)), ...items]
    .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt) || a.id.localeCompare(b.id))
}

async function refreshPrompts(ids: string[]) {
  const wanted = new Set(ids)
  const items = await PromptService.PromptsByIDs(ids)
  replacePrompts(items ?? [], item => wanted.has(item.id))
}

async function refreshKind(target: Kind) {
  const items = await PromptService.ListPromptsByKind(target)
  replacePrompts(items ?? [], item => item.kind === target)
}

const memoryUsage = ref<Awaited<ReturnType<typeof PromptService.MemoryUsage>>>()
let memoryTimer: ReturnType<typeof setInterval>
let readingMemory = false

async function refreshMemoryUsage() {
  if (readingMemory) return
  readingMemory = true
  try {
    memoryUsage.value = await PromptService.MemoryUsage()
  } catch {
    memoryUsage.value = undefined
  } finally {
    readingMemory = false
  }
}

function formatMemory(bytes?: number) {
  if (!bytes) return '—'
  return bytes >= 1024 ** 3 ? `${(bytes / 1024 ** 3).toFixed(1)} GB` : formatBytes(bytes)
}

function onSystemTheme(event: MediaQueryListEvent) {
  systemDark.value = event.matches
}

onMounted(async () => {
  void refreshMemoryUsage()
  memoryTimer = setInterval(refreshMemoryUsage, 5000)
  media.addEventListener('change', onSystemTheme)
  menuUnsubscribe = [
    Events.On('prompt-next:new', () => {
      if (!loading.value && !settingsOpen.value && !pendingImport.value && !imagePreview.value && !videoPreview.value && !collectionsOpen.value && !trashOpen.value && !authorPage.value && !authorsOpen.value && !authorDraft.value) startEdit()
    }),
    Events.On('prompt-next:search', () => {
      if (draft.value || settingsOpen.value || pendingImport.value || collectionsOpen.value || trashOpen.value || authorDraft.value) return
      if (homeOpen.value) switchLibrary(kind.value)
      void nextTick(() => {
        if (authorsOpen.value) authorDirectory.value?.focusSearch()
        else searchInput.value?.focus()
      })
    }),
  ]
  if (isWindows) {
    menuUnsubscribe.push(
      Events.On(Events.Types.Windows.WindowMaximise, () => { windowMaximised.value = true }),
      Events.On(Events.Types.Windows.WindowUnMaximise, () => { windowMaximised.value = false }),
    )
    void Window.IsMaximised().then(value => { windowMaximised.value = value }).catch(showError)
  }
  try {
    const savedLanguage = await PromptService.GetLanguage()
    if (savedLanguage === 'en-US') language.value = savedLanguage
    await Promise.all([reload(), refreshStats()])
    if (stats.value?.locationWarning) showError(stats.value.locationWarning)
    const legacyLayout = localStorage.getItem('prompt-layout')
    if (legacyLayout === 'cards' || legacyLayout === 'list') {
      if (!await PromptService.GetLayout()) await PromptService.SetLayout(legacyLayout)
      localStorage.removeItem('prompt-layout')
    }
    await reloadPreferences()
    appVersion.value = (await PromptService.AppInfo()).version
  } catch (error) { showError(error) } finally { loading.value = false }
})

onUnmounted(() => {
  clearInterval(memoryTimer)
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
  if (!homeOpen.value && !collectionsOpen.value && !authorsOpen.value && !authorPage.value && kind.value === next && !id) return
  navigate(() => {
    homeOpen.value = false
    collectionsOpen.value = false
    authorPage.value = undefined
    authorsOpen.value = false
    kind.value = next
    category.value = ''
    favoritesOnly.value = false
    searchText.value = ''
    query.value = ''
    modelFilter.value = null
    tagFilter.value = null
    selectedId.value = id
    resetVariables()
  })
}

function authorDisplayName(sourceName: string) {
  const name = sourceName?.trim()
  return authorsByName.value.get(name)?.name ?? name
}

function showAuthors() {
  navigate(() => {
    homeOpen.value = false
    collectionsOpen.value = false
    authorPage.value = undefined
    authorsOpen.value = true
    selectedId.value = ''
  })
}

function openAuthor(sourceName: string) {
  const name = sourceName?.trim()
  if (!name || authorPage.value?.sourceName === name) return
  navigate(() => {
    authorPage.value = { sourceName: name, previousId: selectedId.value, previousScroll: resultScroller()?.scrollTop ?? 0, previousPage: browsePage.value, fromDirectory: authorsOpen.value }
    authorsOpen.value = false
    authorLayout.value = libraryLayout.value
    browsePage.value = 1
    authorKind.value = ''
    authorSearchText.value = ''
    authorSort.value = 'updated'
    selectedId.value = ''
    void nextTick(() => { const scroller = resultScroller(); if (scroller) scroller.scrollTop = 0 })
  })
}

function closeAuthor() {
  const page = authorPage.value
  if (!page) return
  navigate(() => {
    authorPage.value = undefined
    authorsOpen.value = page.fromDirectory
    selectedId.value = page.previousId
    browsePage.value = page.previousPage
    void nextTick(() => { const scroller = resultScroller(); if (scroller) scroller.scrollTop = page.previousScroll })
  })
}

function editAuthor(author: Author) {
  authorDraft.value = { ...author }
}

function saveAuthor() {
  const draft = authorDraft.value
  if (!draft?.name.trim()) return
  void perform(async () => {
    const saved = await PromptService.SaveAuthor(draft)
    authors.value = authors.value.map(author => author.sourceName === saved.sourceName ? saved : author)
    authorDraft.value = undefined
    notify(t('作者资料已保存'))
  })
}

function filterAuthor(next: Kind | '') {
  if (authorKind.value === next) return
  navigate(() => {
    authorKind.value = next
    selectedId.value = ''
    if (listPane.value) listPane.value.scrollTop = 0
  })
}

function submitSearch() {
  if (authorPage.value) commitAuthorSearch()
  else commitSearch()
}

function updateSearchText(value: string) {
  if (authorPage.value) {
    authorSearchText.value = value
    return
  }
  searchText.value = value
}

function resetSearch() {
  if (authorPage.value) {
    authorSearchText.value = ''
    authorSort.value = 'updated'
    return
  }
  searchText.value = ''
  query.value = ''
  category.value = ''
  tagFilter.value = null
  modelFilter.value = null
  favoritesOnly.value = false
  sort.value = 'updated'
}

function openCollections(seed = '', id = '') {
  collectionSeed.value = seed
  collectionId.value = id
  collectionsPageKey.value++
  collectionsOpen.value = true
}

function saveCollection(collection: Collection) {
  void perform(async () => {
    const saved = await PromptService.SaveCollection(collection)
    collections.value = await PromptService.ListCollections() ?? []
    if (saved.promptIds?.includes(collectionSeed.value)) collectionSeed.value = ''
    collectionId.value = saved.id
    notify(t('合集已保存'))
  })
}

function deleteCollection(id: string) {
  void perform(async () => {
    await PromptService.DeleteCollection(id)
    collections.value = await PromptService.ListCollections() ?? []
    collectionId.value = collections.value[0]?.id ?? ''
    notify(t('合集已删除'))
  })
}

function copyCollection(text: string) {
  void perform(async () => {
    await PromptService.CopyText(text)
    notify(t('已复制到剪贴板'))
  })
}

function exportCollection(ids: string[]) {
  void perform(async () => {
    if (await PromptService.ExportSelected(ids)) notify(t('合集成员已导出'))
  })
}

function openCollectionPrompt(item: Prompt) {
  collectionsOpen.value = false
  switchLibrary(item.kind as Kind, item.id)
}

function showHome() {
  navigate(() => { homeOpen.value = true; collectionsOpen.value = false; authorPage.value = undefined; authorsOpen.value = false })
}

function setLayout(next: string | number | boolean) {
  if (!isLayout(next)) return
  if (authorPage.value) {
    authorLayout.value = next
    return
  }
  void perform(async () => {
    await PromptService.SetLayout(next)
    libraryLayout.value = next
  })
}

function options(values: string[]) {
  return [...new Set(values.filter(Boolean))].map((value) => ({ label: value, value }))
}

function formatDate(value?: string) {
  if (!value) return t('保存时自动记录')
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

async function reloadModels(library: ModelKind) {
  modelCatalogs.value[library] = await PromptService.Models(library)
}

function addModel() {
  void perform(async () => {
    const library = settingsModelKind.value
    await PromptService.AddModel(library, newModel.value)
    await reloadModels(library)
    newModel.value = ''
    notify(t('模型已添加'))
  })
}

function editModel(name: string) {
  editingModel.value = name
  modelName.value = name
}

function renameModel() {
  void perform(async () => {
    const library = settingsModelKind.value
    await PromptService.RenameModel(library, editingModel.value, modelName.value)
    await reloadModels(library)
    editingModel.value = ''
    notify(t('模型选项已更新'))
  })
}

function deleteModel(name: string) {
  void perform(async () => {
    const library = settingsModelKind.value
    await PromptService.DeleteModel(library, name)
    await reloadModels(library)
    if (editingModel.value === name) editingModel.value = ''
    notify(t('已从模型选项移除'))
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
    notify(t('分类已添加'))
  })
}

function filter(nextCategory = '', nextFavorites = false) {
  navigate(() => {
    category.value = nextCategory
    favoritesOnly.value = nextFavorites
    tagFilter.value = null
    selectedId.value = displayedResults.value[0]?.id ?? ''
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
    id: '', number: 0, kind: kind.value, title: '', body: '', english: '', images: [], referenceImages: [], description: '', category: category.value,
    tags: [], favorite: false, author: '', authorUrl: '', videoUrl: '', videos: [], negative: '', model: '', models: [], parameters: '',
    createdAt: '', updatedAt: '',
  }
}

function startEdit(item?: Prompt, copy = false) {
  navigate(() => {
    draft.value = item ? { ...item, english: item.english ?? '', models: [...promptModels(item)], videos: [...(item.videos ?? [])] } : blankPrompt()
    draftVideoFiles.value = []
    draftVideoSource.value = item?.number ?? 0
    if (item?.videos?.length) void loadVideoDetails(item.number, item.videos)
    const savedImage = (name: string) => ({ name, url: imageURL(item?.number + '/' + name), sourceNumber: item?.number })
    draftImages.value = (item?.images ?? []).map(savedImage)
    draftReferences.value = (item?.referenceImages ?? []).map(savedImage)
    modelQuery.value = ''
    modelPickerOpen.value = false
    if (copy) {
      draft.value.id = ''
      draft.value.number = 0
      draft.value.createdAt = ''
      draft.value.updatedAt = ''
      draft.value.favorite = false
      draft.value.title += t(' · 副本')
    }
    const created = Date.parse(draft.value.createdAt)
    draftCreated.value = Number.isFinite(created) ? created : null
    draftUpdated.value = null
    tagsText.value = (draft.value.tags ?? []).join('，')
    initialDraft.value = snapshotDraft()
  })
}

function save() {
  if (!draft.value || !canSaveDraft.value) return
  const data = {
    ...draft.value, tags: tagsText.value.split(/[,，、]/),
    createdAt: draftCreated.value === null || draftCreated.value === Date.parse(draft.value.createdAt)
      ? '' : new Date(draftCreated.value).toISOString(),
    updatedAt: draftUpdated.value === null ? '' : new Date(draftUpdated.value).toISOString(),
  }
  void perform(async () => {
    const images = [...draftImages.value.map(image => ({ image, reference: false })),
      ...draftReferences.value.map(image => ({ image, reference: true }))]
    const uploadIds: string[] = []
    let saved: Prompt
    try {
      const attachments: NonNullable<Parameters<typeof PromptService.SaveOrderedImages>[1]> = []
      for (const { image, reference } of images) {
        const uploadId = image.file ? crypto.getRandomValues(new Uint32Array(4)).join('-') : ''
        if (image.file) {
          uploadIds.push(uploadId)
          await PromptService.StageImageUpload(uploadId, await readImage(image.file))
        }
        attachments.push({ name: image.name, sourceNumber: image.sourceNumber ?? 0, data: '', reference, uploadId })
      }
      saved = data.kind === 'video'
        ? await PromptService.SaveVideoWithFiles(data, attachments, draftVideoSource.value, draftVideoFiles.value.map(file => file.id))
        : await PromptService.SaveOrderedImages(data, attachments)
    } finally {
      if (uploadIds.length) await PromptService.DiscardImageUploads(uploadIds).catch(() => {})
    }
    const previous = prompts.value.find(item => item.id === saved.id)
    replacePrompts([saved], item => item.id === saved.id)
    const previousMedia = new Set([...(previous?.images ?? []), ...(previous?.referenceImages ?? []), ...(previous?.videos ?? [])])
    const addedMedia = [...(saved.images ?? []), ...(saved.referenceImages ?? []), ...(saved.videos ?? [])]
      .some(name => !previousMedia.has(name))
    invalidateStats(addedMedia)
    if (!authorPage.value && !results.value.some((item) => item.id === saved.id)) {
      category.value = ''
      favoritesOnly.value = false
      searchText.value = ''
      query.value = ''
      modelFilter.value = null
      tagFilter.value = null
    }
    draft.value = null
    clearDraftImages()
    selectedId.value = saved.id
    resetVariables()
    notify(t('已保存提示词'))
    if (saved.author && !authorsByName.value.has(saved.author)) authors.value = await PromptService.ListAuthors() ?? []
  })
}

function chooseVideos() {
  void perform(async () => {
    const files = await PromptService.ChooseVideoFiles(
      draftVideoFiles.value.map(file => file.id), draft.value?.videos?.length ?? 0,
    )
    draftVideoFiles.value.push(...(files ?? []))
  })
}

function removeDraftVideo(index: number) {
  if (draft.value?.videos) draft.value.videos.splice(index, 1)
}

async function loadVideoDetails(number: number, names: string[]) {
  const missing = names.filter(name => !videoDetails.value.has(videoAsset(number, name)))
  if (!missing.length) return
  try {
    for (const detail of await PromptService.VideoDetails(number, missing) ?? []) {
      videoDetails.value.set(videoAsset(number, detail.name), detail.bytes)
    }
  } catch (error) {
    showError(error)
  }
}

watch(selected, (item) => {
  if (item?.videos?.length) void loadVideoDetails(item.number, item.videos)
})

function openVideo(item: Prompt, index: number) {
  if (!item.videos?.[index]) return
  videoPreviewError.value = false
  videoPreview.value = { item, index }
  void loadVideoDetails(item.number, item.videos)
}

function moveVideo(offset: number) {
  if (!videoPreview.value) return
  const { item, index } = videoPreview.value
  openVideo(item, index + offset)
}

function selectCard(event: MouseEvent, item: Prompt) {
  if (!authorPage.value && !batchMode.value && (event.target as Element).closest('[data-video-cover]')) {
    openVideo(item, 0)
    return
  }
  select(item)
}

function toggleFavorite(item: Prompt) {
  void perform(async () => {
    await PromptService.SetFavorite(item.id, !item.favorite)
    replacePrompts([{ ...item, favorite: !item.favorite }], entry => entry.id === item.id)
    invalidateStats()
    if (!selected.value) selectedId.value = displayedResults.value[0]?.id ?? ''
    notify(item.favorite ? t('已取消收藏') : t('已收藏'))
  })
}

function remove(ids: string[]) {
  void perform(async () => {
    await PromptService.DeletePrompts(ids)
    batchIds.value = []
    const removed = new Set(ids)
    replacePrompts([], item => removed.has(item.id))
    invalidateStats()
    selectedId.value = displayedResults.value[0]?.id ?? ''
    resetVariables()
    notify(t('已删除 {0} 条，24 小时内可从回收站恢复', ids.length))
  })
}

function openTrash() {
  void perform(async () => {
    trashEntries.value = await PromptService.ListTrash() ?? []
    trashOpen.value = true
  })
}

function restoreTrash(refs: TrashEntry['ref'][]) {
  void perform(async () => {
    const restored = await PromptService.RestoreTrash(refs)
    trashEntries.value = await PromptService.ListTrash() ?? []
    if (restored) {
      await refreshPrompts(refs.map(ref => ref.id))
      authors.value = await PromptService.ListAuthors() ?? []
      invalidateStats()
    }
    const skipped = refs.length - restored
    notify(skipped ? t('已恢复 {0} 条，跳过 {1} 条已过期或已恢复的提示词', restored, skipped) : t('已恢复 {0} 条提示词', restored))
  })
}

function purgeTrash(refs: TrashEntry['ref'][]) {
  void perform(async () => {
    const result = await PromptService.DeleteTrashPermanently(refs)
    trashEntries.value = await PromptService.ListTrash() ?? []
    invalidateStats(true)
    const skipped = refs.length - result.deleted
    const outcome = t('已彻底删除 {0} 条{1}', result.deleted, skipped ? t('，跳过 {0} 条已移出回收站的提示词', skipped) : '')
    notify(result.cleanupPending ? outcome + t('，部分附件未能清理') : outcome, result.cleanupPending ? 'error' : 'success')
  })
}

async function loadImageDetails(number: number, names: string[]) {
  const missing = names.filter(name => !imageDetails.value.has(number + '/' + name))
  if (!missing.length) return
  try {
    const details = await PromptService.ImageDetails(number, missing)
    for (const detail of details ?? []) imageDetails.value.set(number + '/' + detail.name, detail)
  } catch {
    // The image itself remains usable when metadata is unavailable.
  }
}

function openPreview(images: PreviewImage[], index: number) {
  previewImages.value = images
  previewIndex.value = index
  const number = images[index]?.number
  if (number) void loadImageDetails(number, images.filter(item => item.number === number).map(item => item.name))
}

function downloadPreviewImage() {
  const item = currentPreview.value
  if (!item) return
  void perform(async () => {
    const upload = item.file ? await readImage(item.file) : ''
    if (await PromptService.ExportImage(item.number ?? 0, item.name, upload)) {
      notify(t('原图已保存'))
    }
  })
}

function capturePreviewPixels(event: Event) {
  const image = event.currentTarget as HTMLImageElement
  previewPixels.value = { width: image.naturalWidth, height: image.naturalHeight }
}

function openPromptPreview(number: number, names: string[], index: number) {
  openPreview(names.map(name => ({ name, number, url: imageURL(number + '/' + name) })), index)
}

function openDraftPreview(images: DraftImage[], index: number) {
  openPreview(images.map(image => ({
    url: image.url, name: image.file?.name || image.name, number: image.sourceNumber, file: image.file,
  })), index)
}

function movePreview(offset: number) {
  previewIndex.value = Math.max(0, Math.min(previewImages.value.length - 1, previewIndex.value + offset))
}

function openWebLink(url: string) {
  void perform(() => PromptService.OpenWebLink(url))
}

function libraryLabel(kind: string) {
  return t(libraries.find(item => item.kind === kind)?.label)
}

function hasMedia(kind: string) {
  return kind === 'art' || kind === 'video'
}

function batchFavorite(favorite: boolean) {
  void perform(async () => {
    const ids = visibleBatchIds.value
    const changed = new Set(ids)
    await PromptService.BatchFavorite(ids, favorite)
    replacePrompts(prompts.value.filter(item => changed.has(item.id)).map(item => ({ ...item, favorite })), item => changed.has(item.id))
    batchIds.value = []
    invalidateStats()
    notify(favorite ? t('选中条目已收藏') : t('选中条目已取消收藏'))
  })
}

function applyBatchCategory() {
  if (batchCategory.value === null) return
  void perform(async () => {
    const ids = visibleBatchIds.value
    await PromptService.BatchCategory(ids, kind.value, batchCategory.value ?? '')
    batchIds.value = []
    batchCategory.value = null
    await refreshPrompts(ids)
    invalidateStats()
    notify(t('选中条目的分类已更新'))
  })
}

function exportSelected() {
  void perform(async () => {
    if (await PromptService.ExportSelected(visibleBatchIds.value)) notify(t('选中条目已导出'))
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
    invalidateStats(true)
    const skipped = result.skipped ?? []
    notify(t('已清理 {0} 张，释放 {1}{2}', result.removed, formatBytes(result.bytes), skipped.length ? t('，跳过 {0} 张', skipped.length) : ''))
  })
}

async function reloadPreferences() {
  const fieldKeys = Object.keys(viewFieldDefaults) as ViewFieldKey[]
  const [saved, savedLayout, savedFields] = await Promise.all([
    PromptService.GetTheme(), PromptService.GetLayout(),
    Promise.all(fieldKeys.map(async key => [key, await PromptService.ViewFields(key)] as const)),
  ])
  viewFields.value = Object.fromEntries(savedFields.filter(([, fields]) => fields?.length))
  libraryLayout.value = isLayout(savedLayout) ? savedLayout : 'cards'
  if (saved === 'dark' || saved === 'light' || saved === 'system') theme.value = saved
  await Promise.all([
    reloadModels('art'), reloadModels('video'),
    ...libraries.map(item => reloadCategories(item.kind)),
    PromptService.ListCollections().then(items => { collections.value = items ?? [] }),
  ])
}

function changeViewFields(fields: (string | number)[]) {
  const current = fieldKey.value
  const names = fields.map(String)
  if (!names.length) {
    notify(t('至少保留一个展示字段'), 'error')
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
    await Promise.all([reloadCategories(targetKind), refreshKind(targetKind)])
    invalidateStats()
    notify(deleting ? t('已删除分类，原条目保留为未分类') : t('分类已更新'))
  })
}

function applyTagFilter(name: string) {
  navigate(() => {
    tagFilter.value = tagFilter.value === name ? null : name
    category.value = ''
    favoritesOnly.value = false
    searchText.value = ''
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
    await refreshKind(targetKind)
    invalidateStats()
    notify(t('标签已更新，同名标签已合并'))
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
    notify(deleting ? t('预设已删除') : t('预设已保存'))
  })
}

function copyRendered(item: Prompt) {
  const fields = [[hasMedia(item.kind) ? t('中文提示词') : t('人设提示词'), item.body], [t('英文提示词'), item.english], [t('负向提示词'), item.negative]]
  const output = fields.filter(([, value]) => value?.trim())
    .map(([label, value]) => `【${label}】\n${rendered(value ?? '')}`).join('\n\n')
  void perform(async () => {
    await PromptService.CopyText(output)
    notify(t('已复制全部提示词'))
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
    notify(t('已复制到剪贴板'))
  })
}

const { pendingImport, restoreOpen, chooseImport, closeImport, confirmImport } = useBackupImport(busy, perform, async (result, replace) => {
  await reload()
  invalidateStats(true)
  if (!selected.value) selectedId.value = displayedResults.value[0]?.id ?? ''
  if (replace) {
    draft.value = null
    clearDraftImages()
    resetVariables()
    batchIds.value = []
    await reloadPreferences()
    if (selected.value) presets.value = await PromptService.VariablePresets(selected.value.id) ?? {}
  }
  notify(replace
    ? t('已恢复 {0} 条，安全快照已保存在数据目录的 backups 文件夹', result.added)
    : t('已导入 {0} 条，跳过 {1} 条重复、{2} 条无效记录', result.added, result.duplicate, result.invalid))
})

function importFromSettings() {
  settingsOpen.value = false
  chooseImport()
}

function exportBackup() {
  void perform(async () => {
    if (await PromptService.ExportBackup()) notify(t('提示词备份已保存'))
  })
}

function exportMarkdown(item: Prompt) {
  void perform(async () => {
    if (await PromptService.ExportMarkdown(item)) notify(t('Markdown 已保存'))
  })
}

function changeLanguage(next: string | number | boolean) {
  if ((next !== 'zh-CN' && next !== 'en-US') || next === language.value) return
  void perform(async () => {
    await PromptService.SetLanguage(next)
    language.value = next
    notice.value = undefined
  })
}

function changeTheme(next: string | number | boolean) {
  if (next !== 'light' && next !== 'dark' && next !== 'system') return
  void perform(async () => {
    await PromptService.SetTheme(next)
    theme.value = next
    notify(t('主题设置已保存'))
  })
}

function openDataFolder() {
  void perform(() => PromptService.OpenDataFolder())
}

function chooseDataFolder() {
  void perform(async () => {
    const directory = await PromptService.ChooseDataFolder()
    if (directory) dataFolderChange.value = { directory }
  })
}

function changeDataFolder() {
  const target = dataFolderChange.value
  if (!target) return
  void perform(async () => {
    const changed = target === 'default'
      ? await PromptService.ResetDataFolder()
      : await PromptService.ChangeDataFolder(target.directory)
    dataFolderChange.value = undefined
    if (!changed) notify(target === 'default' ? t('当前已使用默认数据目录') : t('所选文件夹已经是当前数据目录'))
  })
}

function openGitHub() {
  void perform(() => PromptService.OpenGitHub())
}
</script>

<template>
  <NConfigProvider :locale="language === 'en-US' ? enUS : zhCN" :date-locale="language === 'en-US' ? dateEnUS : dateZhCN" :theme="isDark ? darkTheme : null" :theme-overrides="themeOverrides">
    <main class="workspace" :style="workspaceTheme" :class="{ 'is-dark': isDark, 'card-layout': layout !== 'list' || collectionsOpen, 'no-detail': !selected, 'author-workspace': !!authorPage }">
      <header class="app-header" :class="{ 'windows-title-bar': isWindows, 'mac-title-bar': isMac }">
        <div v-if="isWindows || isMac" class="brand title-drag-area" @dblclick="isWindows && runWindowAction(Window.ToggleMaximise())">
          <img class="brand-icon" src="/icons/prompt-next.svg" alt="" draggable="false">
          <strong>Prompt Next</strong>
        </div>
        <div class="header-tools">
          <NButton size="small" quaternary :title="t('导入 JSON')" :aria-label="t('导入 JSON')" :disabled="busy" @click="chooseImport">
            <template #icon><AppIcon name="import" /></template>
          </NButton>
          <NButton size="small" quaternary :title="t('备份')" :aria-label="t('备份')" :disabled="busy" @click="exportBackup">
            <template #icon><AppIcon name="backup" /></template>
          </NButton>
          <NButton size="small" quaternary :title="t('设置')" :aria-label="t('设置')" @click="settingsOpen = true">
            <template #icon><AppIcon name="settings" /></template>
          </NButton>
        </div>
        <div v-if="isWindows" class="window-controls">
          <button type="button" :title="t('最小化窗口')" :aria-label="t('最小化窗口')" @click="runWindowAction(Window.Minimise())">
            <AppIcon name="window-minimise" />
          </button>
          <button type="button" :title="maximiseLabel" :aria-label="maximiseLabel" @click="runWindowAction(Window.ToggleMaximise())">
            <AppIcon :name="windowMaximised ? 'window-restore' : 'window-maximise'" />
          </button>
          <button type="button" class="window-close" :title="t('关闭窗口')" :aria-label="t('关闭窗口')" @click="runWindowAction(Window.Close())">
            <AppIcon name="window-close" />
          </button>
        </div>
      </header>

      <aside class="sidebar" :aria-label="t('提示词库导航')">
        <div class="section-label">{{ t('工作区') }}</div>
        <button class="library-button" :title="t('工作台首页')" :aria-label="t('工作台首页')" :class="{ active: homeOpen && !collectionsOpen }" :aria-pressed="homeOpen && !collectionsOpen" @click="showHome">
          <span class="library-icon"><AppIcon name="home" /></span><span class="nav-label">{{ t('工作台首页') }}</span><span class="nav-short">{{ t('首页') }}</span>
        </button>
        <button v-for="item in libraries" :key="item.kind" class="library-button" :title="t(item.label)" :aria-label="t(item.label)" :class="{ active: !homeOpen && !collectionsOpen && !authorsOpen && !authorPage && kind === item.kind }" :aria-pressed="!homeOpen && !collectionsOpen && !authorsOpen && !authorPage && kind === item.kind" @click="switchLibrary(item.kind)">
          <span class="library-icon" :class="item.kind"><AppIcon :name="item.icon" /></span>
          <span class="nav-label">{{ t(item.label) }}</span><span class="nav-short">{{ t(item.shortLabel) }}</span>
        </button>
        <button class="library-button" :title="t('提示词合集')" :aria-label="t('提示词合集')" :class="{ active: collectionsOpen }" type="button" :aria-pressed="collectionsOpen" @click="openCollections()">
          <span class="library-icon"><AppIcon name="folder" /></span><span class="nav-label">{{ t('提示词合集') }}</span><span class="nav-short">{{ t('合集') }}</span>
        </button>
        <button class="library-button" type="button" :title="t('所有作者')" :aria-label="t('所有作者')" :class="{ active: !collectionsOpen && (authorsOpen || authorPage) }"
          :aria-pressed="!collectionsOpen && (authorsOpen || !!authorPage)" @click="showAuthors">
          <span class="library-icon"><AppIcon name="persona" /></span><span class="nav-label">{{ t('所有作者') }}</span><span class="nav-short">{{ t('作者') }}</span>
        </button>
        <template v-if="!homeOpen && !collectionsOpen && !authorsOpen && !authorPage">
          <section class="sidebar-group" :aria-label="t('当前库')">
            <h2 class="section-label">{{ t('当前库') }}</h2>
            <button class="filter-button" :class="{ active: !category && !favoritesOnly && !tagFilter }" @click="filter()">{{ t('全部') }}<span>{{ libraryItems.length }}</span></button>
            <button class="filter-button" :class="{ active: favoritesOnly }" @click="filter('', true)">{{ t('收藏') }}<span>{{ libraryItems.filter(item => item.favorite).length }}</span></button>
          </section>
          <details v-if="groupedTags.length" class="sidebar-group tag-groups">
            <summary class="section-label"><AppIcon name="chevron-right" />{{ t('按标签聚合') }}</summary>
            <button v-for="tag in groupedTags" :key="tag.name" class="filter-button" :class="{ active: tagFilter === tag.name }" @click="applyTagFilter(tag.name)">
              {{ tag.name }}<span>{{ tag.count }}</span>
            </button>
          </details>
          <section class="sidebar-group" :aria-label="t('分类')">
            <h2 class="section-label">{{ t('分类') }}</h2>
            <div v-if="!categories.length" class="no-categories">{{ t('保存时可填写分类') }}</div>
            <button v-for="name in categories" :key="name" class="filter-button" :class="{ active: category === name }" @click="filter(name)">{{ name }}</button>
          </section>
        </template>
      </aside>

      <WorkspaceHome v-if="homeOpen && !collectionsOpen" :prompts="prompts" :stats="stats"
        :format-date="formatDate" :format-bytes="formatBytes" @open="switchLibrary"
        @new-artwork="switchLibrary('art'); startEdit()" />

      <AuthorDirectory v-if="(authorsOpen || authorPage?.fromDirectory) && !collectionsOpen" v-show="authorsOpen"
        ref="authorDirectory" :authors="authors" :prompts="prompts" :prompts-by-author="promptsByAuthor" :busy="busy"
        :format-date="formatDate" @open="openAuthor" @edit="editAuthor" />
      <section v-if="!homeOpen && !collectionsOpen && !authorsOpen" ref="listPane" class="list-pane" :class="{ 'waterfall-pane': layout === 'waterfall', 'virtual-pane': layout === 'list' }" :aria-label="t(authorPage ? '作者主页' : '提示词列表')">
        <div class="pane-heading">
          <template v-if="authorPage">
            <div class="author-topline">
              <NButton size="small" @click="closeAuthor">‹ {{ t('返回') }}</NButton>
              <span class="section-label">{{ t('作者主页') }}</span>
            </div>
            <div class="author-profile">
              <span class="author-avatar" aria-hidden="true">{{ Array.from(authorName)[0]?.toUpperCase() }}</span>
              <div class="author-profile-copy">
                <h1>{{ authorName }}</h1>
                <p>{{ t('本地收录 {0} 条提示词', authorPrompts.length) }}</p>
              </div>
              <NButton v-if="activeAuthor" size="small" :disabled="busy" @click="editAuthor(activeAuthor)">{{ t('编辑作者') }}</NButton>
              <p v-if="activeAuthor?.bio" class="author-bio">{{ activeAuthor.bio }}</p>
            </div>
            <div class="author-filters" :aria-label="t('按提示词库筛选')">
              <NButton size="small" :type="authorKind === '' ? 'primary' : 'default'" :aria-pressed="authorKind === ''" @click="filterAuthor('')">{{ t('全部') }} {{ authorPrompts.length }}</NButton>
              <NButton v-for="item in libraries" :key="item.kind" size="small" :type="authorKind === item.kind ? 'primary' : 'default'"
                :aria-pressed="authorKind === item.kind" @click="filterAuthor(item.kind)">{{ t(item.label) }} {{ authorCounts[item.kind] }}</NButton>
            </div>
          </template>
          <template v-else>
            <p class="section-label">{{ t(library.description) }}</p>
            <div class="list-heading"><h1>{{ t(library.label) }}</h1><NButton type="primary" size="small" :disabled="busy || loading" @click="startEdit()">{{ t('＋ 新建') }}</NButton></div>
          </template>
          <form class="search-toolbar" @submit.prevent="submitSearch">
            <NInput ref="searchInput" class="search-keyword" :value="authorPage ? authorSearchText : searchText" clearable :disabled="!!draft"
              :placeholder="t('搜索标题、正文、编号…')" :aria-label="t('搜索提示词')" @update:value="updateSearchText">
              <template #prefix><AppIcon name="search" class="search-icon" /></template>
            </NInput>
            <NSelect v-if="!authorPage" class="search-filter search-category" :consistent-menu-width="false" :value="category || null" :options="categoryOptions" clearable filterable :placeholder="t('分类')" :aria-label="t('按分类筛选')" @update:value="value => category = value ?? ''" />
            <NSelect class="search-filter search-sort" :value="authorPage ? authorSort : sort" :options="sortOptions" @update:value="value => authorPage ? authorSort = value : sort = value" :aria-label="t('排序')" />
            <NSelect v-if="!authorPage" class="search-filter search-tag" :consistent-menu-width="false" v-model:value="tagFilter" :options="tagOptions" clearable filterable :placeholder="t('标签')" :aria-label="t('按标签筛选')" />
            <NSelect v-if="!authorPage && hasMedia(kind)" class="search-filter search-model" :consistent-menu-width="false" v-model:value="modelFilter" :options="modelOptions" clearable filterable :placeholder="t('模型')" :aria-label="t('按模型筛选')" />
            <div class="search-actions">
              <NButton v-if="!authorPage" type="primary" attr-type="submit" :disabled="!!draft"><template #icon><AppIcon name="search" /></template>{{ t('搜索') }}</NButton>
              <NButton @click="resetSearch">{{ t('重置') }}</NButton>
            </div>
          </form>
          <div class="list-controls">
            <div class="result-count">{{ authorPage ? t('筛选结果') : (tagFilter || category || (favoritesOnly ? t('收藏') : t('全部'))) }} · {{ displayedResults.length }} {{ t('条') }}</div>
            <NButton v-if="!authorPage" size="small" class="narrow-favorite" :aria-pressed="favoritesOnly"
              :type="favoritesOnly ? 'primary' : 'default'" @click="filter('', !favoritesOnly)">{{ t('收藏') }}</NButton>
            <NButton v-if="!authorPage" size="small" :disabled="busy || !!draft" @click="batchMode = !batchMode; batchIds = []">{{ batchMode ? t('结束多选') : t('多选') }}</NButton>
            <NRadioGroup size="small" :value="layout" @update:value="setLayout">
              <NRadioButton v-for="option in layoutOptions" :key="option.value" :value="option.value"><AppIcon class="ui-icon" :name="option.icon" />{{ t(option.label) }}</NRadioButton>
            </NRadioGroup>
            <NPopover v-if="!authorPage" trigger="click" placement="bottom-end">
              <template #trigger><NButton size="small" :disabled="busy">{{ t('展示字段') }}</NButton></template>
              <div class="view-fields-picker">
                <strong>{{ layoutLabel }}{{ t('展示字段') }}</strong>
                <NCheckboxGroup :value="currentFields" :disabled="busy" @update:value="changeViewFields">
                  <div class="view-field-grid">
                    <NCheckbox v-for="field in kind === 'persona' ? personaFieldOptions : viewFieldOptions" :key="field.value" :value="field.value" :label="t(field.label)" />
                  </div>
                </NCheckboxGroup>
                <NButton size="tiny" :disabled="busy" @click="changeViewFields(viewFieldDefaults[fieldKey])">{{ t('恢复默认') }}</NButton>
              </div>
            </NPopover>
          </div>
        </div>
        <div v-if="batchMode && !authorPage" class="actions batch-actions">
          <NButton size="small" :disabled="busy || !results.length" @click="batchIds = visibleBatchIds.length === results.length ? [] : results.map(item => item.id)">{{ visibleBatchIds.length === results.length ? t('取消全选') : t('全选当前结果') }}</NButton>
          <span>{{ t('已选') }} {{ visibleBatchIds.length }} {{ t('条') }}</span>
          <NButton size="small" :disabled="busy || !visibleBatchIds.length" @click="batchFavorite(true)">{{ t('收藏') }}</NButton>
          <NButton size="small" :disabled="busy || !visibleBatchIds.length" @click="batchFavorite(false)">{{ t('取消收藏') }}</NButton>
          <NSelect v-model:value="batchCategory" size="small" class="batch-category" :placeholder="t('目标分类')" :options="[{ label: t('未分类'), value: '' }, ...configuredCategories[kind].map(name => ({ label: name, value: name }))]" />
          <NButton size="small" :disabled="busy || !visibleBatchIds.length || batchCategory === null" @click="applyBatchCategory">{{ t('改分类') }}</NButton>
          <NButton size="small" :disabled="busy || !visibleBatchIds.length" @click="exportSelected">{{ t('导出选中') }}</NButton>
          <NPopconfirm :negative-text="t('取消')" :positive-text="t('删除')" @positive-click="remove(visibleBatchIds)">
            <template #trigger><NButton size="small" type="error" :disabled="busy || !visibleBatchIds.length">{{ t('删除选中') }}</NButton></template>
            {{ t('确定删除这 {0} 条提示词？', visibleBatchIds.length) }}
          </NPopconfirm>
        </div>
        <div v-if="loading" class="gallery browse-skeleton" role="status" :aria-label="t('正在加载提示词…')">
          <span v-for="index in 6" :key="index" aria-hidden="true" />
        </div>
        <div v-else-if="!displayedResults.length" class="empty list-empty">{{ activeQuery ? t('没有匹配的提示词') : t('这里还没有提示词') }}</div>
        <PromptResults v-else ref="promptResults" v-model:checked="batchIds" v-model:page="browsePage"
          :items="displayedResults" :layout="layout" :written-library="writtenLibrary">
          <template #default="{ item, ratio }">
            <PromptRow :item="item" :layout="layout" :cover-ratio="ratio" :visible-fields="visibleFields" :selected-id="selectedId"
              :batch-mode="batchMode && !authorPage" :author-view="!!authorPage" :show-library="!!authorPage && !authorKind"
              :author-display-name="authorDisplayName" :library-label="libraryLabel" :video-asset="videoAsset" :format-date="formatDate"
              @select="selectCard" @author="openAuthor" />
          </template>
        </PromptResults>
      </section>

      <button v-if="selected && !homeOpen && !collectionsOpen && !authorsOpen" type="button" class="detail-scrim" :aria-label="t('关闭')" @click="selectedId = ''" />
      <section v-if="!homeOpen && !collectionsOpen && !authorsOpen && (selected || (!authorPage && layout === 'list'))" class="detail-pane" :class="{ 'detail-open': !!selected }" :aria-label="t('提示词详情')">
        <article v-if="selected" :key="selected.id" class="detail-article">
          <div class="detail-scroll">
            <header class="detail-card">
              <div class="detail-heading">
                <p class="section-label detail-number">{{ libraryLabel(selected.kind) }} · {{ formatPromptNumber(selected.number) }}</p>
                <div class="actions">
                  <NButton size="small" :disabled="busy" @click="toggleFavorite(selected)"><template #icon><AppIcon :name="selected.favorite ? 'star' : 'star-outline'" /></template>{{ selected.favorite ? t('已收藏') : t('收藏') }}</NButton>
                  <NButton size="small" :disabled="busy" @click="startEdit(selected)">{{ t('编辑') }}</NButton>
                  <NButton size="small" type="primary" :disabled="busy" @click="copyText(selected.body || selected.english)">{{ hasMedia(selected.kind) ? (selected.body ? t('复制中文') : t('复制英文')) : t('复制提示词') }}</NButton>
                  <NButton class="detail-close" :class="{ 'narrow-only': !authorPage && layout === 'list' }" size="small" @click="selectedId = ''">{{ t('关闭') }}</NButton>
                </div>
              </div>
              <h2 class="detail-title">{{ selected.title }}</h2>
              <div class="metadata">
                <NTag v-if="selected.category" type="info" size="small">{{ selected.category }}</NTag>
                <NTag v-for="tag in selected.tags" :key="tag" size="small">{{ tag }}</NTag>
                <span v-if="selected.author?.trim()" class="author-attribution">
                  <span>{{ t('原作者 ·') }}</span>
                  <NButton v-if="!authorPage" text type="primary" size="tiny" :title="t('查看作者主页')" @click="openAuthor(selected.author)">{{ authorDisplayName(selected.author) }}</NButton>
                  <span v-else>{{ authorName }}</span>
                </span>
                <NTag v-for="model in promptModels(selected)" :key="'model:' + model" size="small" type="info">{{ model }}</NTag>
              </div>
              <div v-if="relatedCollections.length" class="actions detail-collections">
                <NButton v-for="collection in relatedCollections" :key="collection.id" size="tiny" secondary
                  @click="openCollections('', collection.id)">{{ collection.name }}</NButton>
              </div>
              <p class="entry-dates">{{ t('创建') }} {{ formatDate(selected.createdAt) }} {{ t('· 更新') }} {{ formatDate(selected.updatedAt) }}</p>
              <PromptSource v-if="selected.authorUrl" :url="selected.authorUrl" @open="openWebLink" />
            </header>
            <section v-if="selected.description" class="detail-content-section">
              <h3 class="detail-section-heading"><AppIcon name="text" />{{ t('简介') }}</h3>
              <p class="detail-intro">{{ selected.description }}</p>
            </section>
            <section v-if="selected.videos?.length" class="detail-content-section">
              <h3 class="detail-section-heading"><AppIcon name="video" />{{ t('效果视频') }}
                <span class="detail-section-count">{{ selected.videos.length }} {{ t('段') }}</span>
              </h3>
              <div class="video-attachments">
                <figure v-for="(name, index) in selected.videos" :key="name" class="detail-image">
                  <button type="button" class="video-thumb" :aria-label="t('播放视频 ') + (index + 1)" @click="openVideo(selected, index)">
                    <VideoCover :src="videoAsset(selected.number, name)" />
                  </button>
                  <figcaption>{{ t('视频') }} {{ index + 1 }} · {{ formatBytes(videoDetails.get(videoAsset(selected.number, name))) }}</figcaption>
                </figure>
              </div>
            </section>
            <section v-for="group in [
              { label: '效果图', names: selected.images },
              { label: '参考图片', names: selected.referenceImages },
            ].filter(group => group.names?.length)" :key="group.label" class="detail-content-section">
              <h3 class="detail-section-heading"><AppIcon name="art" />{{ t(group.label) }}
                <span class="detail-section-count">{{ group.names?.length }} {{ t('张') }}</span>
              </h3>
              <div class="attached-images">
                <figure v-for="(name, index) in group.names" :key="name" class="detail-image">
                  <button type="button" :aria-label="t('预览图片')" @click="openPromptPreview(selected.number, group.names ?? [], index)">
                    <img :src="imageURL(selected.number + '/' + name, true)" :alt="t(group.label)" loading="lazy">
                  </button>
                  <figcaption>{{ formatBytes(imageDetails.get(selected.number + '/' + name)?.bytes) }}</figcaption>
                </figure>
              </div>
            </section>
            <div v-if="variables.length" class="variable-panel">
              <div class="block-heading"><strong>{{ t('模板变量') }}</strong><NButton size="tiny" secondary @click="fillVariables = !fillVariables">{{ fillVariables ? t('收起') : t('填写并预览') }}</NButton></div>
              <div v-if="fillVariables" class="fields">
                <div class="actions preset-actions">
                  <NSelect :value="activePreset" :options="presetOptions" clearable :placeholder="t('载入变量预设')" @update:value="loadPreset" />
                  <NInput v-model:value="presetName" maxlength="60" :placeholder="t('预设名称')" />
                  <NButton size="small" :disabled="busy || !presetName.trim()" @click="savePreset(selected.id)">{{ t('保存') }}</NButton>
                  <NButton size="small" :disabled="busy || !activePreset" @click="savePreset(selected.id, true)">{{ t('移除') }}</NButton>
                </div>
                <label v-for="name in variables" :key="name">{{ name }}<NInput v-model:value="values[name]" :placeholder="name" /></label>
              </div>
            </div>
            <PromptFold v-if="selected.body" :title="hasMedia(selected.kind) ? t('中文提示词') : selected.kind === 'persona' ? t('人设提示词') : t('提示词正文')">
              <template #actions>
                <span v-if="fillVariables" class="section-label">{{ t('替换预览') }}</span>
                <NButton size="tiny" secondary :disabled="busy" @click="copyText(selected.body)">{{ t('复制') }}</NButton>
              </template>
              <pre>{{ fillVariables ? rendered(selected.body) : selected.body }}</pre>
            </PromptFold>
            <PromptFold v-if="selected.english" :title="t('英文提示词')">
              <template #actions><NButton size="tiny" secondary :disabled="busy" @click="copyText(selected.english)">{{ t('复制英文') }}</NButton></template>
              <pre>{{ fillVariables ? rendered(selected.english) : selected.english }}</pre>
            </PromptFold>
            <PromptFold v-if="hasMedia(selected.kind) && selected.negative" :title="t('负向提示词')" :initially-open="false">
              <template #actions><NButton size="tiny" secondary :disabled="busy" @click="copyText(selected.negative)">{{ t('复制负向') }}</NButton></template>
              <pre>{{ fillVariables ? rendered(selected.negative) : selected.negative }}</pre>
            </PromptFold>
            <PromptFold v-if="selected.parameters" :title="t('参数备注')" :initially-open="false">
              <pre>{{ selected.parameters }}</pre>
            </PromptFold>
          </div>
          <footer class="actions detail-footer">
            <NButton size="small" :disabled="busy" @click="openCollections(selected.id)">{{ t('加入合集') }}</NButton>
            <NButton v-if="hasMedia(selected.kind)" size="small" :disabled="busy" @click="copyRendered(selected)">{{ t('复制全部提示词') }}</NButton>
            <NButton size="small" :disabled="busy" @click="startEdit(selected, true)">{{ t('复制为新条目') }}</NButton>
            <NButton size="small" :disabled="busy" @click="exportMarkdown(selected)">{{ t('导出 Markdown') }}</NButton>
            <NPopconfirm :negative-text="t('取消')" :positive-text="t('删除')" :positive-button-props="{ type: 'error' }" @positive-click="remove([selected.id])">
              <template #trigger><NButton size="small" quaternary type="error" :disabled="busy">{{ t('删除') }}</NButton></template>
              {{ t('确定删除「{0}」？', selected.title) }}
            </NPopconfirm>
          </footer>
        </article>

        <div v-else class="empty detail-empty">
          <div class="empty-icon"><AppIcon class="ui-icon" :name="library.icon" /></div><h2>{{ t(library.label) }}</h2>
          <p>{{ loading ? t('正在加载本地数据库') : t('从列表选择一条提示词，或创建自己的记录。') }}</p>
          <div v-if="!loading" class="actions"><NButton type="primary" @click="startEdit()">{{ t('新建提示词') }}</NButton><NButton v-if="kind === 'persona'" @click="chooseImport">{{ t('导入旧人设') }}</NButton></div>
        </div>
      </section>

      <PromptCollections v-if="collectionsOpen" :key="collectionsPageKey" :selected-id="collectionId" :seed-id="collectionSeed"
        :collections="collections" :prompts="prompts" :busy="busy" :format-date="formatDate" :author-display-name="authorDisplayName"
        @select="id => collectionId = id" @save="saveCollection" @remove="deleteCollection" @open-prompt="openCollectionPrompt"
        @copy="copyCollection" @export="exportCollection" />

      <NModal :show="!!authorDraft" preset="card" :title="t('编辑作者')" class="app-modal" :mask-closable="!busy"
        @update:show="show => { if (!show && !busy) authorDraft = undefined }">
        <form v-if="authorDraft" class="editor" @submit.prevent="saveAuthor">
          <label>{{ t('作者名称') }} <NInput v-model:value="authorDraft.name" :maxlength="200" /></label>
          <label>{{ t('作者简介') }} <NInput v-model:value="authorDraft.bio" type="textarea" :maxlength="2000" /></label>
          <div class="actions">
            <NButton :disabled="busy" @click="authorDraft = undefined">{{ t('取消') }}</NButton>
            <NButton type="primary" attr-type="submit" :disabled="busy || !authorDraft.name.trim()">{{ t('保存') }}</NButton>
          </div>
        </form>
      </NModal>

      <footer class="status">
        <NButton size="tiny" text :disabled="busy" @click="openTrash"><template #icon><AppIcon name="trash" /></template>{{ t('回收站') }}</NButton>
        <span class="status-divider">·</span> {{ t('共') }} {{ prompts.length }} {{ t('条提示词') }}
        <span class="status-memory" :title="t('应用主进程占用与系统物理内存总量，每 5 秒刷新')">
          <span>{{ t('应用内存') }} <strong>{{ formatMemory(memoryUsage?.appBytes) }}</strong></span>
          <span aria-hidden="true">/</span>
          <span>{{ t('系统总内存') }} <strong>{{ formatMemory(memoryUsage?.totalBytes) }}</strong></span>
        </span>
      </footer>
      <PromptTrash :show="trashOpen" :entries="trashEntries" :busy="busy" :format-date="formatDate"
        :library-label="libraryLabel" @close="trashOpen = false" @restore="restoreTrash" @purge="purgeTrash" />
      <NAlert v-if="notice" class="notice" :type="notice.type" closable @close="notice = undefined">{{ notice.text }}</NAlert>

      <PromptEditor :show="!discardOpen" :is-dark="isDark" :busy="busy" :media-preview-open="!!imagePreview || !!videoPreview"
        :can-save-draft="canSaveDraft" :category-options="categoryOptions" :author-options="authorOptions" :editor-models="editorModels"
        :draft-video-source="draftVideoSource" :video-details="videoDetails" :effect-images="effectImages" :reference-images="referenceImages"
        :video-asset="videoAsset" :format-bytes="formatBytes" :format-date="formatDate"
        v-model:draft="draft" v-model:tags-text="tagsText" v-model:created="draftCreated" v-model:updated="draftUpdated"
        v-model:video-files="draftVideoFiles" v-model:model-picker-open="modelPickerOpen" v-model:model-query="modelQuery"
        @close="navigate(() => {})" @save="save" @choose-videos="chooseVideos" @remove-video="removeDraftVideo" @preview="openDraftPreview" />

      <NModal :show="!!imagePreview" preset="card" :title="t('图片预览 · {0} / {1}', previewIndex + 1, previewImages.length)"
        class="image-modal" @update:show="previewImages = []">
        <div class="image-preview-stage" :class="{ 'multi-image': previewImages.length > 1 }" @keydown.left.prevent="movePreview(-1)" @keydown.right.prevent="movePreview(1)">
          <NButton v-if="previewImages.length > 1" circle class="preview-arrow" :aria-label="t('上一张')" :disabled="previewIndex === 0" @click="movePreview(-1)"><template #icon><AppIcon name="chevron-left" /></template></NButton>
          <ImagePreviewViewport v-if="imagePreview" :src="imagePreview" @load="capturePreviewPixels" />
          <NButton v-if="previewImages.length > 1" circle class="preview-arrow" :aria-label="t('下一张')" :disabled="previewIndex === previewImages.length - 1" @click="movePreview(1)"><template #icon><AppIcon name="chevron-right" /></template></NButton>
        </div>
        <dl v-if="currentPreview" class="image-preview-info">
          <div class="preview-filename"><dt>{{ t('文件名') }}</dt><dd>{{ currentPreview.name }}</dd></div>
          <div><dt>{{ t('大小') }}</dt><dd>{{ formatBytes(previewBytes) }}</dd></div>
          <div><dt>{{ t('分辨率') }}</dt><dd>{{ previewDimensions ? previewDimensions.width + ' × ' + previewDimensions.height : '—' }}</dd></div>
          <div><dt>{{ t('格式') }}</dt><dd>{{ previewFormat || '—' }}</dd></div>
        </dl>
        <template #footer>
          <div class="image-preview-actions">
            <NButton type="primary" :loading="busy" :disabled="!currentPreview" @click="downloadPreviewImage">{{ t('下载原图') }}</NButton>
          </div>
        </template>
      </NModal>

      <NModal :show="!!activeVideo" preset="card" class="image-modal video-modal"
        :title="t('视频播放 · {0} / {1}', (videoPreview?.index ?? 0) + 1, videoPreview?.item.videos?.length ?? 0)"
        @update:show="videoPreview = undefined">
        <template v-if="videoPreview && activeVideo">
          <video :key="videoAsset(videoPreview.item.number, activeVideo)" class="prompt-video"
            :src="videoAsset(videoPreview.item.number, activeVideo)" controls autoplay playsinline
            :aria-label="t('提示词视频')" @error="videoPreviewError = true" />
          <NAlert v-if="videoPreviewError" type="error">{{ t('无法播放此视频。文件可能已丢失，或当前系统不支持它的编码。') }}</NAlert>
          <div class="actions video-player-footer">
            <span>{{ formatBytes(videoDetails.get(videoAsset(videoPreview.item.number, activeVideo))) }}</span>
            <template v-if="(videoPreview.item.videos?.length ?? 0) > 1">
              <NButton size="small" :disabled="videoPreview.index === 0" @click="moveVideo(-1)">{{ t('上一段') }}</NButton>
              <NButton size="small" :disabled="videoPreview.index === (videoPreview.item.videos?.length ?? 0) - 1" @click="moveVideo(1)">{{ t('下一段') }}</NButton>
            </template>
          </div>
        </template>
      </NModal>

      <WorkspaceSettings v-model:show="settingsOpen" v-model:section="settingsSection" :is-dark="isDark" :is-mac="isMac">
            <template v-if="settingsSection === 'appearance'">
              <div class="setting-field">
                <h3>{{ t('语言') }}</h3>
                <NRadioGroup :value="language" :disabled="busy" @update:value="changeLanguage" :aria-label="t('语言')">
                  <NRadioButton value="zh-CN">简体中文</NRadioButton>
                  <NRadioButton value="en-US">English</NRadioButton>
                </NRadioGroup>
              </div>
              <div class="setting-field">
                <h3>{{ t('软件主题') }}</h3>
                <NRadioGroup :value="theme" :disabled="busy" @update:value="changeTheme"><NRadioButton value="system">{{ t('跟随系统') }}</NRadioButton><NRadioButton value="light">{{ t('浅色') }}</NRadioButton><NRadioButton value="dark">{{ t('深色') }}</NRadioButton></NRadioGroup>
              </div>
              <div class="setting-field">
                <h3>{{ t('提示词视图') }}</h3>
                <NRadioGroup :value="layout" @update:value="setLayout"><NRadioButton v-for="option in layoutOptions" :key="option.value" :value="option.value">{{ t(option.label) }}</NRadioButton></NRadioGroup>
              </div>
              <p class="setting-hint">{{ t('主题和视图更改即时生效。') }}</p>
            </template>
            <template v-else-if="settingsSection === 'models'">
              <div class="setting-field">
                <h3>{{ t('提示词库') }}</h3>
                <NRadioGroup v-model:value="settingsModelKind" :disabled="busy">
                  <NRadioButton value="art">{{ t('绘画提示词库') }}</NRadioButton>
                  <NRadioButton value="video">{{ t('视频提示词库') }}</NRadioButton>
                </NRadioGroup>
              </div>
              <div class="setting-field">
                <h3>{{ t('添加自定义模型') }}</h3>
                <form class="settings-model-form" @submit.prevent="addModel">
                  <NInput v-model:value="newModel" :placeholder="t('输入模型名称')" maxlength="100" :aria-label="t('模型名称')" />
                  <NButton attr-type="submit" type="primary" :disabled="!newModel.trim() || busy">{{ t('添加') }}</NButton>
                </form>
                <p class="setting-hint">{{ t('添加后可在所选提示词库的模型下拉框中使用。') }}</p>
              </div>
              <div class="setting-field">
                <h3>{{ t('自定义模型') }}</h3>
                <p class="setting-hint">{{ t('修改和删除只影响下拉选项，已有提示词中的模型名称仍保留。') }}</p>
                <p v-if="!customModels.length" class="setting-hint">{{ t('还没有自定义模型') }}</p>
                <div v-for="name in customModels" :key="name" class="model-setting-row">
                  <form v-if="editingModel === name" class="settings-model-form" @submit.prevent="renameModel">
                    <NInput v-model:value="modelName" size="small" maxlength="100" :aria-label="t('修改模型名称')" />
                    <NButton size="small" attr-type="submit" type="primary" :disabled="busy || !modelName.trim()">{{ t('保存') }}</NButton>
                    <NButton size="small" @click="editingModel = ''">{{ t('取消') }}</NButton>
                  </form>
                  <template v-else>
                    <span>{{ name }}</span>
                    <div class="actions">
                      <NButton size="tiny" :disabled="busy" @click="editModel(name)">{{ t('修改') }}</NButton>
                      <NPopconfirm :negative-text="t('取消')" :positive-text="t('删除')" :positive-button-props="{ type: 'error' }" @positive-click="deleteModel(name)">
                        <template #trigger><NButton size="tiny" type="error" quaternary :disabled="busy">{{ t('删除') }}</NButton></template>
                        {{ t('从模型选项删除「{0}」？', name) }}
                      </NPopconfirm>
                    </div>
                  </template>
                </div>
              </div>
              <div class="setting-field">
                <h3>{{ t('内置模型') }}</h3>
                <div class="metadata"><NTag v-for="model in settingsModels?.builtIn" :key="model" size="small">{{ model }}</NTag></div>
              </div>
            </template>
            <template v-else-if="settingsSection === 'categories'">
              <div class="setting-field">
                <h3>{{ t('提示词库') }}</h3>
                <NRadioGroup v-model:value="settingsCategoryKind">
                  <NRadioButton value="text">{{ t('文本') }}</NRadioButton>
                  <NRadioButton value="art">{{ t('绘画') }}</NRadioButton>
                  <NRadioButton value="video">{{ t('视频') }}</NRadioButton>
                  <NRadioButton value="persona">{{ t('人设') }}</NRadioButton>
                </NRadioGroup>
              </div>
              <div class="setting-field">
                <h3>{{ t('添加分类') }}</h3>
                <form class="settings-model-form" @submit.prevent="addCategory">
                  <NInput v-model:value="newCategory" :placeholder="t('输入分类名称')" maxlength="60" :aria-label="t('分类名称')" />
                  <NButton attr-type="submit" type="primary" :disabled="!newCategory.trim() || busy">{{ t('添加') }}</NButton>
                </form>
                <p class="setting-hint">{{ t('新建和编辑时可选择本库的分类。') }}</p>
              </div>
              <div class="setting-field">
                <h3>{{ t('自定义分类') }}</h3>
                <p v-if="!customCategories[settingsCategoryKind].length" class="setting-hint">{{ t('还没有自定义分类。') }}</p>
                <div v-for="name in customCategories[settingsCategoryKind]" :key="name" class="actions management-row">
                  <template v-if="editingCategory === name">
                    <NInput v-model:value="categoryName" size="small" maxlength="60" @keyup.enter="updateCategory(name)" />
                    <NButton size="small" :disabled="busy || !categoryName.trim()" @click="updateCategory(name)">{{ t('保存') }}</NButton>
                    <NButton size="small" @click="editingCategory = ''">{{ t('取消') }}</NButton>
                  </template>
                  <template v-else>
                    <span>{{ name }}</span>
                    <NButton size="small" :disabled="busy" @click="editingCategory = name; categoryName = name">{{ t('改名') }}</NButton>
                    <NPopconfirm :negative-text="t('取消')" :positive-text="t('删除')" @positive-click="updateCategory(name, true)">
                      <template #trigger><NButton size="small" type="error" :disabled="busy">{{ t('删除') }}</NButton></template>
                      {{ t('原有条目将保留并移至未分类。') }}
                    </NPopconfirm>
                  </template>
                </div>
                <h3>{{ t('内置分类') }}</h3>
                <div class="metadata"><NTag v-for="name in configuredCategories[settingsCategoryKind].filter(item => !customCategories[settingsCategoryKind].includes(item))" :key="name" size="small">{{ name }}</NTag></div>
              </div>
            </template>
            <template v-else-if="settingsSection === 'tags'">
              <div class="setting-field">
                <h3>{{ t('标签管理') }}</h3>
                <NRadioGroup v-model:value="settingsCategoryKind">
                  <NRadioButton value="text">{{ t('文本') }}</NRadioButton>
                  <NRadioButton value="art">{{ t('绘画') }}</NRadioButton>
                  <NRadioButton value="video">{{ t('视频') }}</NRadioButton>
                  <NRadioButton value="persona">{{ t('人设') }}</NRadioButton>
                </NRadioGroup>
                <p class="setting-hint">{{ t('输入已有标签名即可合并，计数按条目统计。') }}</p>
                <p v-if="!managedTags.length" class="setting-hint">{{ t('当前库还没有标签。') }}</p>
                <div v-for="tag in managedTags" :key="tag.name" class="actions management-row">
                  <template v-if="editingTag === tag.name">
                    <NInput v-model:value="tagName" size="small" maxlength="60" @keyup.enter="renameTag" />
                    <NButton size="small" :disabled="busy || !tagName.trim()" @click="renameTag">{{ t('保存') }}</NButton>
                    <NButton size="small" @click="editingTag = ''">{{ t('取消') }}</NButton>
                  </template>
                  <template v-else>
                    <span>{{ tag.name }} · {{ tag.count }} {{ t('条') }}</span>
                    <NButton size="small" @click="editingTag = tag.name; tagName = tag.name">{{ t('改名或合并') }}</NButton>
                  </template>
                </div>
              </div>
            </template>
            <template v-else-if="settingsSection === 'data'">
              <div class="setting-field">
                <h3>{{ t('应用数据目录') }}</h3>
                <code class="settings-path">{{ stats?.directory || t('正在读取目录…') }}</code>
                <p v-if="stats?.locationWarning" class="setting-hint">{{ language === 'en-US' ? translateBackendError(stats.locationWarning) : stats.locationWarning }}</p>
                <p class="setting-hint">{{ t('数据库') }} {{ formatBytes(stats?.dbBytes) }} {{ t('· 图片') }} {{ formatBytes(stats?.imageBytes) }} {{ t('· 视频') }} {{ formatBytes(stats?.videoBytes) }}</p>
                <div class="actions">
                  <NButton size="small" :disabled="busy" @click="openDataFolder">{{ t('打开文件夹') }}</NButton>
                  <NButton size="small" type="primary" secondary :disabled="busy" @click="chooseDataFolder">{{ t('更换目录') }}</NButton>
                  <NButton size="small" :disabled="busy" @click="dataFolderChange = 'default'">{{ t('恢复默认目录') }}</NButton>
                </div>
              </div>
              <div class="setting-field">
                <h3>{{ t('导入和备份') }}</h3>
                <div class="actions"><NButton :disabled="busy" @click="importFromSettings">{{ t('导入 JSON') }}</NButton><NButton :disabled="busy" @click="exportBackup">{{ t('导出备份') }}</NButton></div>
              </div>
              <div class="setting-field">
                <h3>{{ t('图片清理') }}</h3>
                <p class="setting-hint">{{ t('先查看文件和占用，再确认清理。') }}</p>
                <NButton :disabled="busy" @click="previewCleanup">{{ t('查看未被引用的图片') }}</NButton>
              </div>
            </template>
            <template v-else>
              <div class="settings-about">
                <img src="/icons/prompt-next.svg" alt="" width="40" height="40">
                <div><strong>Prompt Next</strong><p class="setting-hint">{{ t('版本') }} {{ appVersion || "—" }} · Daneliya</p></div>
              </div>
              <div class="setting-field"><h3>GitHub</h3><NButton text type="primary" @click="openGitHub">github.com/Daneliya ↗</NButton></div>
            </template>
      </WorkspaceSettings>

      <NModal :show="!!dataFolderChange" preset="card" :title="dataFolderChange === 'default' ? t('恢复默认数据目录') : t('更换数据目录')" class="app-modal"
        :mask-closable="!busy" :close-on-esc="!busy" :closable="!busy" @update:show="dataFolderChange = undefined">
        <p v-if="dataFolderChange === 'default'">{{ t('应用将重新启动并打开默认目录中的提示词库。当前目录中的数据会保留。') }}</p>
        <template v-else-if="dataFolderChange">
          <code class="settings-path">{{ dataFolderChange.directory }}</code>
          <p>{{ t('选择已有数据目录会打开其中的提示词库。选择空文件夹会创建一个新库。原目录中的数据会保留。') }}</p>
        </template>
        <div class="actions">
          <NButton :disabled="busy" @click="dataFolderChange = undefined">{{ t('取消') }}</NButton>
          <NButton type="primary" :loading="busy" @click="changeDataFolder">{{ t('切换并重新启动') }}</NButton>
        </div>
      </NModal>

      <ImageCleanupDialog :files="cleanupFiles ?? undefined" :busy="busy" :format-bytes="formatBytes" @close="cleanupFiles = undefined" @confirm="confirmCleanup" />

      <NModal :show="!!pendingImport" preset="card" :title="t('导入预览')" class="app-modal" :mask-closable="!busy" :closable="!busy" :close-on-esc="!busy" @update:show="closeImport">
        <template v-if="pendingImport">
          <p>{{ pendingImport.name }}</p>
          <p>{{ t('新增') }} {{ pendingImport.added }} {{ t('条 · 重复') }} {{ pendingImport.duplicate }} {{ t('条 · 无效') }} {{ pendingImport.invalid }} {{ t('条') }}</p>
          <p>{{ t('重复和无效记录会跳过，已有编辑会保留。') }}</p>
          <div class="actions"><NButton :disabled="busy" @click="closeImport">{{ t('关闭') }}</NButton><NButton v-if="pendingImport.canRestore" type="warning" secondary :disabled="busy" @click="restoreOpen = true">{{ t('覆盖恢复…') }}</NButton><NButton type="primary" :disabled="!pendingImport.added" :loading="busy" @click="confirmImport()">{{ t('确认导入') }}</NButton></div>
        </template>
      </NModal>

      <NModal v-model:show="restoreOpen" preset="card" :title="t('覆盖恢复提示词？')" class="app-modal" :mask-closable="!busy" :closable="!busy">
        <p>{{ pendingImport?.includesVideo ? t('将替换四个库的提示词。') : pendingImport?.includesText ? t('此备份会替换文本、绘画和人设库，保留视频提示词。') : t('此备份会替换绘画和人设库，保留文本和视频提示词。') }} {{ t('应用会丢弃未保存的草稿，并在覆盖前自动保存一份安全快照。') }}</p>
        <div class="actions"><NButton :disabled="busy" @click="restoreOpen = false">{{ t('取消') }}</NButton><NButton type="error" :loading="busy" @click="confirmImport(true)">{{ t('确认覆盖恢复') }}</NButton></div>
      </NModal>

      <NModal v-model:show="discardOpen" preset="card" :title="t('放弃未保存的修改？')" class="app-modal">
        <p>{{ t('当前草稿还没有保存。') }}</p>
        <div class="actions"><NButton @click="discardOpen = false">{{ t('继续编辑') }}</NButton><NButton type="error" @click="discard">{{ t('放弃修改') }}</NButton></div>
      </NModal>
    </main>
  </NConfigProvider>
</template>
