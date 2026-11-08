<script setup lang="ts">
import { language, t } from './i18n'
import { computed, onMounted, onUnmounted, nextTick, ref, watch } from 'vue'
import {
  darkTheme, zhCN, enUS, dateZhCN, dateEnUS, NAlert, NButton, NCheckbox, NCheckboxGroup, NConfigProvider, NDatePicker, NInput, NModal, NPopconfirm, NPopover,
  NRadioButton, NRadioGroup, NSelect, NTag,
} from 'naive-ui'
import { PromptService } from '../bindings/github.com/Daneliya/prompt-next/internal/desktop'
import { Events } from '@wailsio/runtime'
import { formatPromptNumber, promptCharacterCount, type Kind, type Prompt } from './prompt'
import { countLabels, promptModels, usePromptFilters } from './composables/usePromptFilters'
import { imageURL, readImage, useEditorImages, type DraftImage } from './composables/useEditorImages'
import { useBackupImport } from './composables/useBackupImport'
import ImageCleanupDialog from './components/ImageCleanupDialog.vue'
import PromptTrash from './components/PromptTrash.vue'
import PromptCollections from './components/PromptCollections.vue'
import PromptFold from './components/PromptFold.vue'
import EditorImageGroup from './components/EditorImageGroup.vue'
import ImagePreviewViewport from './components/ImagePreviewViewport.vue'
import VideoCover from './components/VideoCover.vue'
import AppIcon from './components/AppIcon.vue'
import DistributionChart from './components/DistributionChart.vue'
import PromptTrend from './components/PromptTrend.vue'
import { desktopTheme } from './desktopTheme'
import './workspace.css'

type Theme = 'system' | 'light' | 'dark'
const libraries = [
  { kind: 'text', label: '文本提示词库', icon: 'text', description: '写作、翻译、学习与日常任务' },
  { kind: 'art', label: '绘画提示词库', icon: 'art', description: '画面描述、风格与生成参数' },
  { kind: 'video', label: '视频提示词库', icon: 'video', description: '镜头、动作、分镜与视频生成提示词' },
  { kind: 'persona', label: '大肥鱼人设库', icon: 'fish', description: '角色设定与助手指令' },
] as const

type Collection = Awaited<ReturnType<typeof PromptService.SaveCollection>>
const collections = ref<Collection[]>([])
const collectionsOpen = ref(false)
const collectionId = ref('')
const collectionSeed = ref('')
const collectionsPageKey = ref(0)

const prompts = ref<Prompt[]>([])
const kind = ref<Kind>('text')
const homeOpen = ref(true)
const layoutOptions = [
  { value: 'cards', label: '卡片', icon: 'cards' },
  { value: 'list', label: '列表', icon: 'list' },
  { value: 'waterfall', label: '瀑布流', icon: 'waterfall' },
] as const
type Layout = (typeof layoutOptions)[number]['value']

function isLayout(value: unknown): value is Layout {
  return layoutOptions.some(option => option.value === value)
}
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
const waterfallFields = [...defaultViewFields, 'tags']
const personaFields = ['title', 'category', 'characters', 'description', 'number', 'created', 'updated']
const personaFieldOptions = viewFieldOptions
  .filter(field => personaFields.includes(field.value))
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
const visibleFields = computed(() => new Set(currentFields.value))
const writtenLibrary = computed(() => kind.value === 'text' || kind.value === 'persona')

const models = ref<string[]>([])
const customModels = ref<string[]>([])
const newModel = ref('')
const editingModel = ref('')
const modelName = ref('')
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
const stats = ref<Awaited<ReturnType<typeof PromptService.StorageStats>>>()
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
const settingsHeading = ref<HTMLElement>()
const settingsSections = [
  { key: 'appearance', label: '外观', icon: 'settings' },
  { key: 'models', label: 'AI 模型', icon: 'art' },
  { key: 'categories', label: '分类', icon: 'folder' },
  { key: 'tags', label: '标签', icon: 'tag' },
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
const library = computed(() => libraries.find((item) => item.kind === kind.value)!)
const libraryItems = computed(() => prompts.value.filter((item) => item.kind === kind.value))
const { sort, modelFilter, tagFilter, category, favoritesOnly, query, results } = usePromptFilters(libraryItems)
const searchText = ref('')
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
const categoryDistribution = computed(() => countLabels(artwork.value.map(item => item.category || t('未分类'))))
const modelDistribution = computed(() => countLabels(artwork.value.flatMap(item => {
  const names = promptModels(item)
  return names.length ? names : [t('未选模型')]
})))

const selected = computed(() => results.value.find((item) => item.id === selectedId.value))
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
  notify(message.includes('unknown bound method id') ? t('前端与桌面程序版本不一致，请退出并重新启动最新版。') : message, 'error')
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
      if (!loading.value && !settingsOpen.value && !pendingImport.value && !imagePreview.value && !videoPreview.value && !collectionsOpen.value && !trashOpen.value) startEdit()
    }),
    Events.On('prompt-next:search', () => {
      if (draft.value || settingsOpen.value || pendingImport.value || collectionsOpen.value || trashOpen.value) return
      if (homeOpen.value) switchLibrary(kind.value)
      void nextTick(() => searchInput.value?.focus())
    }),
  ]
  try {
    const savedLanguage = await PromptService.GetLanguage()
    if (savedLanguage === 'en-US') language.value = savedLanguage
    await reload()
    if (stats.value?.locationWarning) notify(stats.value.locationWarning, 'error')
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
  if (!homeOpen.value && !collectionsOpen.value && kind.value === next && !id) return
  navigate(() => {
    homeOpen.value = false
    collectionsOpen.value = false
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

function submitSearch() {
  query.value = searchText.value.trim()
}

function updateSearchText(value: string) {
  searchText.value = value
  if (!value.trim()) query.value = ''
}

function resetSearch() {
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
    collectionSeed.value = ''
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

function openCollectionPrompt(item: Prompt) {
  collectionsOpen.value = false
  switchLibrary(item.kind as Kind, item.id)
}

function showHome() {
  navigate(() => { homeOpen.value = true; collectionsOpen.value = false })
}

function setLayout(next: string | number | boolean) {
  if (!isLayout(next)) return
  void perform(async () => {
    await PromptService.SetLayout(next)
    layout.value = next
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
    notify(t('模型已添加'))
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
    notify(t('模型选项已更新'))
  })
}

function deleteModel(name: string) {
  void perform(async () => {
    await PromptService.DeleteModel(name)
    await reloadModels()
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
    await reload()
    if (!results.value.some((item) => item.id === saved.id)) {
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

function videoAsset(number: number, name: string) {
  return '/prompt-videos/' + number + '/' + encodeURIComponent(name)
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
  if (!batchMode.value && (event.target as Element).closest('[data-video-cover]')) {
    openVideo(item, 0)
    return
  }
  select(item)
}

function toggleFavorite(item: Prompt) {
  void perform(async () => {
    await PromptService.SetFavorite(item.id, !item.favorite)
    await reload()
    if (!selected.value) selectedId.value = results.value[0]?.id ?? ''
    notify(item.favorite ? t('已取消收藏') : t('已收藏'))
  })
}

function remove(ids: string[]) {
  void perform(async () => {
    await PromptService.DeletePrompts(ids)
    batchIds.value = []
    await reload()
    selectedId.value = results.value[0]?.id ?? ''
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
    await reload()
    const skipped = refs.length - restored
    notify(skipped ? t('已恢复 {0} 条，跳过 {1} 条已过期或已恢复的提示词', restored, skipped) : t('已恢复 {0} 条提示词', restored))
  })
}

function purgeTrash(refs: TrashEntry['ref'][]) {
  void perform(async () => {
    const result = await PromptService.DeleteTrashPermanently(refs)
    trashEntries.value = await PromptService.ListTrash() ?? []
    await reload()
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

function sizeWaterfallCover(event: Event) {
  const image = event.currentTarget as HTMLImageElement
  if (image.naturalWidth && image.naturalHeight) {
    image.parentElement?.style.setProperty('aspect-ratio', image.naturalWidth + ' / ' + image.naturalHeight)
  }
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
    await PromptService.BatchFavorite(visibleBatchIds.value, favorite)
    batchIds.value = []
    await reload()
    notify(favorite ? t('选中条目已收藏') : t('选中条目已取消收藏'))
  })
}

function applyBatchCategory() {
  if (batchCategory.value === null) return
  void perform(async () => {
    await PromptService.BatchCategory(visibleBatchIds.value, kind.value, batchCategory.value ?? '')
    batchIds.value = []
    batchCategory.value = null
    await reload()
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
    await reload()
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
  layout.value = isLayout(savedLayout) ? savedLayout : 'cards'
  if (saved === 'dark' || saved === 'light' || saved === 'system') theme.value = saved
  await Promise.all([
    reloadModels(),
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
    await Promise.all([reloadCategories(targetKind), reload()])
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
    await reload()
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
    <main class="workspace" :class="{ 'is-dark': isDark, 'card-layout': layout !== 'list' || collectionsOpen, 'no-detail': !selected }">
      <header class="app-header">
        <div class="brand"><img class="brand-icon" src="/icons/prompt-next.svg" alt=""><div><strong>Prompt Next</strong><small>{{ t('本地创作工作台') }}</small></div></div>
        <div class="actions">
          <NButton size="small" secondary :disabled="busy" @click="chooseImport">{{ t('导入 JSON') }}</NButton>
          <NButton size="small" secondary :disabled="busy" @click="exportBackup">{{ t('备份') }}</NButton>
          <NButton size="small" secondary @click="settingsOpen = true"><template #icon><AppIcon name="settings" /></template>{{ t('设置') }}</NButton>
        </div>
      </header>

      <aside class="sidebar" :aria-label="t('提示词库导航')">
        <div class="section-label">{{ t('工作区') }}</div>
        <button class="library-button" :class="{ active: homeOpen && !collectionsOpen }" :aria-pressed="homeOpen && !collectionsOpen" @click="showHome">
          <span class="library-icon"><AppIcon name="home" /></span>{{ t('工作台首页') }}
        </button>
        <button v-for="item in libraries" :key="item.kind" class="library-button" :class="{ active: !homeOpen && !collectionsOpen && kind === item.kind }" :aria-pressed="!homeOpen && !collectionsOpen && kind === item.kind" @click="switchLibrary(item.kind)">
          <span class="library-icon" :class="item.kind"><AppIcon :name="item.icon" /></span>
          <span>{{ t(item.label) }}</span>
        </button>
        <button class="library-button" :class="{ active: collectionsOpen }" type="button" :aria-pressed="collectionsOpen" @click="openCollections()">
          <span class="library-icon"><AppIcon name="folder" /></span><span>{{ t('提示词合集') }}</span>
        </button>
        <template v-if="!homeOpen && !collectionsOpen">
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

      <section v-if="homeOpen && !collectionsOpen" class="home-pane" :aria-label="t('工作台首页')">
        <div class="hero">
          <div><p>{{ t('本地创作工作台') }}</p><h1>Prompt Next</h1><p>{{ t('整理文本提示词、绘画参考图和角色设定。') }}</p>
            <div class="actions"><NButton type="primary" @click="switchLibrary('text')">{{ t('进入文本库') }}</NButton><NButton @click="switchLibrary('art')">{{ t('进入绘画库') }}</NButton><NButton @click="switchLibrary('art'); startEdit()">{{ t('新建绘画提示词') }}</NButton></div>
          </div>
          <img class="hero-logo" src="/icons/prompt-next.svg" alt="Prompt Next Logo">
        </div>
        <div class="overview">
          <button v-for="item in libraries" :key="item.kind" class="stat-card" @click="switchLibrary(item.kind)">
            <span class="library-icon" :class="item.kind"><AppIcon :name="item.icon" /></span>
            <span>{{ t(item.label) }}</span><strong>{{ prompts.filter(p => p.kind === item.kind).length }} {{ t('条') }}</strong>
          </button>
          <div class="stat-card"><span class="library-icon"><AppIcon name="disk" /></span>
            <span>{{ t('数据库占用') }}</span><strong>{{ formatBytes(stats?.dbBytes) }}</strong>
          </div>
          <div class="stat-card"><span class="library-icon art"><AppIcon name="art" /></span>
            <span>{{ t('图片占用 ·') }} {{ stats?.imageFiles ?? 0 }} {{ t('张') }}</span><strong>{{ formatBytes(stats?.imageBytes) }}</strong>
          </div>
          <div v-if="stats?.videoFiles" class="stat-card"><span class="library-icon"><AppIcon name="video" /></span>
            <span>{{ t('视频占用 ·') }} {{ stats.videoFiles }} {{ t('个') }}</span><strong>{{ formatBytes(stats.videoBytes) }}</strong>
          </div>
        </div>
        <div class="dashboard-charts">
          <PromptTrend :days="stats?.recentPrompts ?? []" />
          <DistributionChart :title="t('绘画分类')" :subtitle="t('按绘画条目统计')" :items="categoryDistribution" />
          <DistributionChart :title="t('AI 模型')" :subtitle="t('多选模型分别计数')" :items="modelDistribution" />
        </div>
        <div class="home-section">
          <h2>{{ t('最近更新') }}</h2>
          <p v-if="!recent.length" class="section-label">{{ t('还没有提示词') }}</p>
          <button v-for="item in recent" :key="item.id" class="recent-row" @click="switchLibrary(item.kind as Kind, item.id)">
            <span>{{ item.title }}</span><small>{{ libraryLabel(item.kind) }} · {{ formatDate(item.updatedAt) }}</small>
          </button>
        </div>
        <div v-if="stats" class="data-path"><strong>{{ t('本地共占用') }} {{ formatBytes(stats.dbBytes + stats.imageBytes + stats.videoBytes) }}</strong><code>{{ stats.directory }}</code></div>
      </section>

      <section v-if="!homeOpen && !collectionsOpen" class="list-pane" :aria-label="t('提示词列表')">
        <div class="pane-heading">
          <p class="section-label">{{ t(library.description) }}</p>
          <div class="list-heading"><h1>{{ t(library.label) }}</h1><NButton type="primary" size="small" :disabled="busy || loading" @click="startEdit()">{{ t('＋ 新建') }}</NButton></div>
          <form class="search-toolbar" @submit.prevent="submitSearch">
            <NInput ref="searchInput" class="search-keyword" :value="searchText" clearable :disabled="!!draft"
              :placeholder="t('搜索标题、正文、编号…')" :aria-label="t('搜索提示词')" @update:value="updateSearchText">
              <template #prefix><AppIcon name="search" class="search-icon" /></template>
            </NInput>
            <NSelect class="search-filter search-category" :consistent-menu-width="false" :value="category || null" :options="categoryOptions" clearable filterable :placeholder="t('分类')" :aria-label="t('按分类筛选')" @update:value="value => category = value ?? ''" />
            <NSelect class="search-filter search-sort" v-model:value="sort" :options="sortOptions" :aria-label="t('排序')" />
            <NSelect class="search-filter search-tag" :consistent-menu-width="false" v-model:value="tagFilter" :options="tagOptions" clearable filterable :placeholder="t('标签')" :aria-label="t('按标签筛选')" />
            <NSelect v-if="hasMedia(kind)" class="search-filter search-model" :consistent-menu-width="false" v-model:value="modelFilter" :options="modelOptions" clearable filterable :placeholder="t('模型')" :aria-label="t('按模型筛选')" />
            <div class="search-actions">
              <NButton type="primary" attr-type="submit" :disabled="!!draft"><template #icon><AppIcon name="search" /></template>{{ t('搜索') }}</NButton>
              <NButton @click="resetSearch">{{ t('重置') }}</NButton>
            </div>
          </form>
          <div class="list-controls">
            <div class="result-count">{{ tagFilter || category || (favoritesOnly ? t('收藏') : t('全部')) }} · {{ results.length }} {{ t('条') }}</div>
            <NButton size="small" :disabled="busy || !!draft" @click="batchMode = !batchMode; batchIds = []">{{ batchMode ? t('结束多选') : t('多选') }}</NButton>
            <NRadioGroup size="small" :value="layout" @update:value="setLayout">
              <NRadioButton v-for="option in layoutOptions" :key="option.value" :value="option.value"><AppIcon class="ui-icon" :name="option.icon" />{{ t(option.label) }}</NRadioButton>
            </NRadioGroup>
            <NPopover trigger="click" placement="bottom-end">
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
        <div v-if="batchMode" class="actions batch-actions">
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
        <div v-if="loading" class="empty list-empty">{{ t('正在加载提示词…') }}</div>
        <div v-else-if="!results.length" class="empty list-empty">{{ query ? t('没有匹配的提示词') : t('这里还没有提示词') }}</div>
        <NCheckboxGroup v-model:value="batchIds" :class="{ gallery: layout === 'cards', waterfall: layout === 'waterfall', 'written-gallery': writtenLibrary }">
        <div v-for="item in results" :key="item.id" class="prompt-entry">
        <NCheckbox v-if="batchMode" :value="item.id" class="batch-check" :aria-label="t('选中 ') + item.title" />
        <button class="prompt-row" :class="{ active: item.id === selectedId, 'waterfall-photo': !writtenLibrary && layout === 'waterfall' && !!(item.videos?.length || item.images?.length), 'written-preview': writtenLibrary }" :aria-pressed="item.id === selectedId" :aria-label="item.title" @click="selectCard($event, item)">
          <span v-if="!writtenLibrary && layout === 'waterfall' && (item.videos?.length || item.images?.length)" class="waterfall-cover" :class="{ 'waterfall-video': item.videos?.length }" :data-video-cover="item.videos?.length ? '' : undefined">
            <VideoCover v-if="item.videos?.length" :key="item.videos[0]" :src="videoAsset(item.number, item.videos[0]!)" natural-ratio />
            <img v-else-if="item.images?.length" :src="imageURL(item.number + '/' + item.images[0]!, true)" alt="" loading="lazy" @load="sizeWaterfallCover">
            <span v-if="visibleFields.has('category') && item.category" class="waterfall-category">{{ item.category }}</span>
            <span v-if="item.favorite" class="waterfall-favorite" :aria-label="t('已收藏')">★</span>
          </span>
          <span v-else-if="layout === 'cards' && !writtenLibrary" class="card-cover" :class="item.kind" :data-video-cover="item.videos?.length ? '' : undefined">
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
            <span v-if="writtenLibrary && layout !== 'list'" class="written-heading">
              <span class="written-icon"><AppIcon :name="item.kind === 'persona' ? 'fish' : 'text'" /></span>
              <span v-if="visibleFields.has('category') && (item.category || item.kind === 'persona')" class="written-category" :title="item.category">{{ item.category || t('未分类') }}</span>
              <span v-if="visibleFields.has('characters')" class="written-length">{{ t('{0} 字', promptCharacterCount(item.body)) }}</span>
            </span>
            <span v-if="visibleFields.has('title')" class="row-title" :title="item.title">{{ item.title }} <span v-if="item.favorite && (writtenLibrary || layout !== 'waterfall' || !(item.videos?.length || item.images?.length))" class="star">★</span></span>
            <span v-if="visibleFields.has('body') && (item.body || item.english)" class="row-summary">{{ item.body || item.english }}</span>
            <span v-if="visibleFields.has('description') && (item.description || item.kind === 'persona')" class="row-summary" :class="{ 'persona-summary': item.kind === 'persona' }">{{ item.description?.trim() || (item.kind === 'persona' ? item.body : '') }}</span>
            <span v-if="visibleFields.has('models') && promptModels(item).length" class="row-category" :title="promptModels(item).join(' · ')">{{ promptModels(item).join(' · ') }}</span>
            <span v-if="visibleFields.has('category') && (item.category || item.kind === 'persona') && (layout === 'list' || !writtenLibrary) && (layout !== 'waterfall' || !(item.videos?.length || item.images?.length))" class="row-category">{{ item.category || t('未分类') }}</span>
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
        </div>
        </NCheckboxGroup>
      </section>

      <section v-if="!homeOpen && !collectionsOpen && (selected || layout === 'list')" class="detail-pane" :class="{ 'detail-open': !!selected }" :aria-label="t('提示词详情')">
        <article v-if="selected" :key="selected.id" class="detail-article">
          <div class="detail-scroll">
            <header class="detail-card">
              <div class="detail-heading">
                <p class="section-label detail-number">{{ t(library.label) }} · {{ formatPromptNumber(selected.number) }}</p>
                <div class="actions">
                  <NButton size="small" :disabled="busy" @click="toggleFavorite(selected)">{{ selected.favorite ? t('★ 已收藏') : t('☆ 收藏') }}</NButton>
                  <NButton size="small" :disabled="busy" @click="startEdit(selected)">{{ t('编辑') }}</NButton>
                  <NButton size="small" type="primary" :disabled="busy" @click="copyText(selected.body || selected.english)">{{ hasMedia(selected.kind) ? (selected.body ? t('复制中文') : t('复制英文')) : t('复制提示词') }}</NButton>
                  <NButton v-if="layout !== 'list'" size="small" @click="selectedId = ''">{{ t('关闭') }}</NButton>
                </div>
              </div>
              <h2 class="detail-title">{{ selected.title }}</h2>
              <div class="metadata">
                <NTag v-if="selected.category" type="info" size="small">{{ selected.category }}</NTag>
                <NTag v-for="tag in selected.tags" :key="tag" size="small">{{ tag }}</NTag>
                <NButton v-if="selected.authorUrl" text size="tiny" @click="openWebLink(selected.authorUrl)">{{ t('原作者 ·') }} {{ selected.author || t('查看来源') }} ↗</NButton>
                <span v-else-if="selected.author">{{ t('原作者 ·') }} {{ selected.author }}</span>
                <NTag v-for="model in promptModels(selected)" :key="'model:' + model" size="small" type="info">{{ model }}</NTag>
              </div>
              <div v-if="relatedCollections.length" class="actions detail-collections">
                <NButton v-for="collection in relatedCollections" :key="collection.id" size="tiny" secondary
                  @click="openCollections('', collection.id)">{{ collection.name }}</NButton>
              </div>
              <p class="entry-dates">{{ t('创建') }} {{ formatDate(selected.createdAt) }} {{ t('· 更新') }} {{ formatDate(selected.updatedAt) }}</p>
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
        :collections="collections" :prompts="prompts" :busy="busy" @close="collectionsOpen = false"
        @select="id => collectionId = id" @save="saveCollection" @remove="deleteCollection" @open-prompt="openCollectionPrompt" />

      <footer class="status">
        <NButton size="tiny" text :disabled="busy" @click="openTrash"><template #icon><AppIcon name="trash" /></template>{{ t('回收站') }}</NButton>
        <span class="status-divider">·</span> {{ t('共') }} {{ prompts.length }} {{ t('条提示词') }}
      </footer>
      <PromptTrash :show="trashOpen" :entries="trashEntries" :busy="busy" :format-date="formatDate"
        :library-label="libraryLabel" @close="trashOpen = false" @restore="restoreTrash" @purge="purgeTrash" />
      <NAlert v-if="notice" class="notice" :type="notice.type" closable @close="notice = undefined">{{ notice.text }}</NAlert>

      <NModal :show="!!draft && !discardOpen" preset="card" :title="draft?.id ? t('编辑提示词') : t('新建提示词')" class="editor-modal" :class="{ 'is-dark': isDark }" :mask-closable="false" :close-on-esc="!busy && !imagePreview && !videoPreview && !modelPickerOpen" :closable="!busy" @update:show="navigate(() => {})">
        <form v-if="draft" id="prompt-editor-form" class="editor" @submit.prevent="save">
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
              <NButton size="small" secondary :disabled="busy || (draft.videos?.length ?? 0) + draftVideoFiles.length >= 12" @click="chooseVideos">{{ t('添加视频') }}</NButton>
            </div>
            <p class="section-label">{{ t('可多选 MP4、WebM，最多 12 段。单段最大 64 MB，每次新增合计最大 192 MB，首段作为封面。') }}</p>
            <div v-for="(name, index) in draft.videos" :key="name" class="draft-video-row">
              <span>{{ t('视频') }} {{ index + 1 }} · {{ formatBytes(videoDetails.get(videoAsset(draftVideoSource, name))) }}</span>
              <NButton size="tiny" quaternary type="error" :disabled="busy" @click="removeDraftVideo(index)">{{ t('移除') }}</NButton>
            </div>
            <div v-for="(file, index) in draftVideoFiles" :key="file.id" class="draft-video-row">
              <span :title="file.name">{{ file.name }} · {{ formatBytes(file.bytes) }}</span>
              <NButton size="tiny" quaternary type="error" :disabled="busy" @click="draftVideoFiles.splice(index, 1)">{{ t('移除') }}</NButton>
            </div>
          </section>
          <EditorImageGroup v-if="draft.kind === 'art' || (draft.kind === 'video' && draftImages.length)"
            :editor="effectImages" :title="t('效果图')" :required="draft.kind === 'art'" :cover="draft.kind === 'art'" :busy="busy"
            :hint="t(draft.kind === 'art' ? '至少添加一张，首张效果图作为封面。可拖动排序。' : '可保留或移除已有的效果图。')"
            @preview="openDraftPreview" />
          <EditorImageGroup v-if="hasMedia(draft.kind)"
            :editor="referenceImages" :title="t('参考图片')" :hint="t('用于记录生成前的原始图片。可添加多张并拖动排序。')"
            :busy="busy" @preview="openDraftPreview" />
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
              <label>{{ t('原作者名称') }} <NInput v-model:value="draft.author" :placeholder="t('选填')" /></label>
              <label>{{ t('原作者链接') }} <NInput v-model:value="draft.authorUrl" :placeholder="t('输入网页链接')" /></label>
            </div>
          </section>

        </form>
        <template #footer>
          <div v-if="draft" class="actions editor-footer">
            <NButton :disabled="busy" @click="navigate(() => {})">{{ t('取消') }}</NButton>
            <NButton form="prompt-editor-form" attr-type="submit" type="primary" :loading="busy"
              :disabled="!canSaveDraft">{{ t('保存') }}</NButton>
          </div>
        </template>
      </NModal>

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

      <NModal v-model:show="settingsOpen" preset="card" class="settings-modal" :auto-focus="false" @after-enter="settingsHeading?.focus()" :class="{ 'is-dark': isDark }" :bordered="false" :mask-closable="false">
        <template #header><span ref="settingsHeading" class="settings-heading" tabindex="-1">{{ t('偏好设置') }}</span></template>
        <div class="settings-layout">
          <nav class="settings-nav" :aria-label="t('设置分类')">
            <button v-for="section in settingsSections" :key="section.key" type="button" :aria-pressed="settingsSection === section.key" @click="settingsSection = section.key">
              <AppIcon :name="section.icon" class="ui-icon" />{{ t(section.label) }}
            </button>
          </nav>
          <section class="settings-content" :aria-label="t(settingsSections.find(section => section.key === settingsSection)?.label)">
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
                <h3>{{ t('添加自定义模型') }}</h3>
                <form class="settings-model-form" @submit.prevent="addModel">
                  <NInput v-model:value="newModel" :placeholder="t('输入模型名称')" maxlength="100" :aria-label="t('模型名称')" />
                  <NButton attr-type="submit" type="primary" :disabled="!newModel.trim() || busy">{{ t('添加') }}</NButton>
                </form>
                <p class="setting-hint">{{ t('添加后可在绘画条目的模型下拉框中选择。') }}</p>
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
                <div class="metadata"><NTag v-for="model in models.filter(name => !customModels.includes(name))" :key="model" size="small">{{ model }}</NTag></div>
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
                <p v-if="stats?.locationWarning" class="setting-hint">{{ stats.locationWarning }}</p>
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
          </section>
        </div>
        <template #footer><div class="settings-footer"><NButton type="primary" @click="settingsOpen = false">{{ t('完成') }}</NButton></div></template>
      </NModal>

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
