<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { pinyin } from 'pinyin-pro'
import { NButton, NInput, NRadioButton, NRadioGroup, NSelect } from 'naive-ui'
import type { Author } from '../../bindings/github.com/Daneliya/prompt-next/internal/library'
import { t } from '../i18n'
import { libraries, type Prompt } from '../prompt'
import { promptSourceIcon } from '../promptSourceIcons'
import AppIcon from './AppIcon.vue'
import CollectionMedia from './CollectionMedia.vue'

const props = defineProps<{
  authors: Author[]
  prompts: Prompt[]
  promptsByAuthor: Map<string, Prompt[]>
  busy: boolean
  formatDate: (value: string) => string
}>()
const emit = defineEmits<{ open: [name: string]; edit: [author: Author] }>()
const pane = ref<HTMLElement>()
const searchInput = ref<InstanceType<typeof NInput>>()
defineExpose({ scrollContainer: () => pane.value, focusSearch: () => searchInput.value?.focus() })

const search = ref('')
const sort = ref('works')
const sourceFilter = ref('')
const withoutWorks = ref(false)
const layout = ref('cards')
const letters = [...'ABCDEFGHIJKLMNOPQRSTUVWXYZ', '#']
const selectedLetter = ref('')
const siteLabels: Readonly<Record<string, string>> = {
  'prompthub.png': 'PromptHub', 'civitai.png': 'Civitai', 'liblib.png': 'LiblibAI', 'jimeng.png': '即梦',
  'midjourney.svg': 'Midjourney', 'xiaohongshu.png': '小红书', 'github.png': 'GitHub',
  'huggingface.png': 'Hugging Face', 'bilibili.png': '哔哩哔哩', 'douyin.png': '抖音',
}
const sortOptions = computed(() => [
  { label: t('按作品数'), value: 'works' }, { label: t('最近活跃'), value: 'recent' }, { label: t('按名称'), value: 'name' },
])

function sourceSite(url: string) {
  try {
    const address = new URL(url)
    if (!['http:', 'https:'].includes(address.protocol) || address.username || address.password) return ''
    return promptSourceIcon(address.hostname) || 'other'
  } catch {
    return ''
  }
}

function sourceLabel(site: string) {
  return t(siteLabels[site.split('/').at(-1) ?? ''] ?? '其他网站')
}

function authorInitial(name: string) {
  const normalized = name.normalize('NFKD').replace(/^[\p{P}\p{S}\p{M}\s]+/u, '')
  const first = Array.from(normalized)[0] ?? ''
  const initial = /\p{Script=Han}/u.test(first)
    ? pinyin(normalized, { pattern: 'first', toneType: 'none', type: 'array' })[0] ?? ''
    : first
  return /^[a-z]$/i.test(initial) ? initial.toUpperCase() : '#'
}

const summaries = computed(() => props.authors.map(author => {
  const members = props.promptsByAuthor.get(author.sourceName) ?? []
  const counts = new Map<string, number>()
  const sites = new Set<string>()
  let featured: Prompt | undefined
  let lastActive = ''
  for (const item of members) {
    counts.set(item.kind, (counts.get(item.kind) ?? 0) + 1)
    const site = sourceSite(item.authorUrl)
    if (site) sites.add(site)
    if (item.updatedAt > lastActive) lastActive = item.updatedAt
    if (!featured || item.updatedAt > featured.updatedAt) featured = item
  }
  const breakdown = libraries.flatMap(library => {
    const count = counts.get(library.kind)
    return count ? [{ ...library, count }] : []
  })
  return { author, initial: authorInitial(author.name), count: members.length, breakdown, sites: [...sites], featured, lastActive }
}))

const byName = (a: typeof summaries.value[number], b: typeof summaries.value[number]) =>
  a.author.name.localeCompare(b.author.name, 'zh-CN') || a.author.sourceName.localeCompare(b.author.sourceName, 'zh-CN')
const byWorks = (a: typeof summaries.value[number], b: typeof summaries.value[number]) => b.count - a.count || byName(a, b)
const topAuthor = computed(() => summaries.value.filter(entry => entry.count).reduce<typeof summaries.value[number] | undefined>(
  (top, entry) => !top || byWorks(entry, top) < 0 ? entry : top, undefined))
const recentAuthors = computed(() => {
  const cutoff = Date.now() - 30 * 24 * 60 * 60 * 1000
  return summaries.value.filter(entry => Date.parse(entry.lastActive) >= cutoff).length
})
const unsigned = computed(() => props.prompts.filter(item => !item.author?.trim()).length)
const emptyAuthors = computed(() => summaries.value.filter(entry => !entry.count).length)
const sources = computed(() => [...new Set(summaries.value.flatMap(entry => entry.sites))]
  .sort((a, b) => a === 'other' ? 1 : b === 'other' ? -1 : sourceLabel(a).localeCompare(sourceLabel(b), 'zh-CN')))
const filtered = computed(() => {
  const keyword = search.value.trim().toLocaleLowerCase()
  return summaries.value.filter(entry =>
    (!withoutWorks.value || !entry.count) &&
    (!sourceFilter.value || entry.sites.includes(sourceFilter.value)) &&
    [entry.author.name, entry.author.sourceName, entry.author.bio].some(value => value.toLocaleLowerCase().includes(keyword)))
    .sort((a, b) => sort.value === 'name' ? byName(a, b)
      : sort.value === 'recent' ? b.lastActive.localeCompare(a.lastActive) || byName(a, b)
        : byWorks(a, b))
})

const letterGroups = computed(() => {
  const groups = new Map<string, typeof filtered.value>()
  for (const entry of filtered.value) {
    const group = groups.get(entry.initial)
    if (group) group.push(entry)
    else groups.set(entry.initial, [entry])
  }
  return groups
})
const sections = computed(() => sort.value === 'name'
  ? letters.flatMap(letter => {
    const entries = letterGroups.value.get(letter)
    return entries ? [{ letter, entries }] : []
  })
  : [{ letter: '', entries: filtered.value }])

watch(filtered, () => { selectedLetter.value = '' })

async function jumpToLetter(letter: string) {
  sort.value = 'name'
  await nextTick()
  const container = pane.value
  const heading = container?.querySelector<HTMLElement>(`[data-author-letter="${letter}"]`)
  if (!container || !heading) return
  selectedLetter.value = letter
  container.scrollTop += heading.getBoundingClientRect().top - container.getBoundingClientRect().top - 12
}

function reset() {
  search.value = ''
  sourceFilter.value = ''
  withoutWorks.value = false
}
</script>

<template>
  <section ref="pane" class="home-pane authors-pane" :aria-label="t('所有作者')">
    <header class="authors-heading"><h1>{{ t('所有作者') }}</h1></header>
    <div class="author-stats">
      <div><span>{{ t('本地收录作者') }}</span><strong>{{ authors.length }}</strong></div>
      <div><span>{{ t('作品最多') }}</span><strong :title="topAuthor?.author.name">{{ topAuthor ? topAuthor.author.name + ' · ' + topAuthor.count : '—' }}</strong></div>
      <div><span>{{ t('近 30 天活跃作者') }}</span><strong>{{ recentAuthors }}</strong></div>
      <div><span>{{ t('未署名条目') }}</span><strong>{{ unsigned }}</strong></div>
    </div>

    <div class="author-toolbar">
      <NInput ref="searchInput" v-model:value="search" class="author-directory-search" size="small" clearable
        :placeholder="t('搜索作者名称或简介…')" :aria-label="t('搜索作者名称或简介…')">
        <template #prefix><AppIcon name="search" class="search-icon" /></template>
      </NInput>
      <NSelect v-model:value="sort" class="author-directory-sort" size="small" :options="sortOptions" :aria-label="t('排序')" />
      <NButton v-if="emptyAuthors" size="small" :type="withoutWorks ? 'primary' : 'default'" :aria-pressed="withoutWorks"
        @click="withoutWorks = !withoutWorks; if (withoutWorks) sourceFilter = ''">{{ t('无在库作品') }} · {{ emptyAuthors }}</NButton>
      <NRadioGroup v-model:value="layout" class="author-directory-layout" size="small" :aria-label="t('作者视图')">
        <NRadioButton value="cards">{{ t('卡片') }}</NRadioButton>
        <NRadioButton value="list">{{ t('列表') }}</NRadioButton>
      </NRadioGroup>
    </div>
    <div class="author-filter-row">
      <div v-if="sources.length" class="author-source-filters" :aria-label="t('按作品来源筛选')">
        <NButton size="tiny" :type="sourceFilter ? 'default' : 'primary'" :aria-pressed="!sourceFilter" @click="sourceFilter = ''">{{ t('全部来源') }}</NButton>
        <NButton v-for="site in sources" :key="site" size="tiny" :type="sourceFilter === site ? 'primary' : 'default'"
          :aria-pressed="sourceFilter === site" @click="sourceFilter = sourceFilter === site ? '' : site; withoutWorks = false">
          <img v-if="site !== 'other'" class="author-site-icon" :src="site" alt="">{{ sourceLabel(site) }}
        </NButton>
      </div>
      <span class="author-result-count">{{ t('{0} 位作者', filtered.length) }}</span>
    </div>

    <div v-if="!filtered.length" class="empty list-empty guided-empty">
      <strong>{{ authors.length ? t('没有匹配的作者') : t('还没有作者') }}</strong>
      <p v-if="!authors.length">{{ t('在提示词的原作者字段填写名称，作者就会出现在这里。') }}</p>
      <NButton v-else size="small" @click="reset">{{ t('重置') }}</NButton>
    </div>
    <div v-else class="author-index-layout">
      <div class="author-sections">
        <section v-for="section in sections" :key="section.letter" class="author-letter-section">
          <h2 v-if="section.letter" :data-author-letter="section.letter">{{ section.letter }}</h2>
          <div class="authors-grid" :class="layout === 'list' ? 'authors-list' : 'authors-cards'">
            <article v-for="entry in section.entries" :key="entry.author.sourceName" class="author-tile">
              <button class="author-tile-open" type="button" @click="emit('open', entry.author.sourceName)">
                <span class="author-avatar" aria-hidden="true">{{ Array.from(entry.author.name)[0]?.toUpperCase() }}</span>
                <span class="author-tile-copy">
                  <strong :title="entry.author.name">{{ entry.author.name }}</strong>
                  <small v-if="layout === 'cards' && entry.author.bio" :title="entry.author.bio">{{ entry.author.bio }}</small>
                  <small v-else-if="entry.author.name !== entry.author.sourceName">{{ entry.author.sourceName }}</small>
                </span>
              </button>
              <NButton size="tiny" class="author-tile-edit" :disabled="busy" @click="emit('edit', entry.author)">{{ t('编辑') }}</NButton>
              <div v-if="layout === 'list'" class="author-breakdown">
                <span v-if="entry.count" class="author-breakdown-bar" aria-hidden="true">
                  <span v-for="item in entry.breakdown" :key="item.kind" :class="'author-kind-' + item.kind" :style="{ flexGrow: item.count }" />
                </span>
                <span>{{ t('{0} 条 · 跨 {1} 库', entry.count, entry.breakdown.length) }}</span>
              </div>
              <template v-if="layout === 'cards'">
                <button v-if="entry.featured" class="author-featured-work" type="button" :aria-label="t('查看作者主页')"
                  @click="emit('open', entry.author.sourceName)">
                  <CollectionMedia :item="entry.featured" />
                  <span class="author-featured-copy">
                    <strong :title="entry.featured.title">{{ entry.featured.title }}</strong>
                    <small>{{ entry.featured.description || entry.featured.body || entry.featured.english }}</small>
                  </span>
                </button>
                <div v-else class="author-featured-work author-featured-empty">
                  <CollectionMedia /><span>{{ t('无在库作品') }}</span>
                </div>
                <div class="author-tile-legend">
                  <span v-for="item in entry.breakdown" :key="item.kind"><i :class="'author-kind-' + item.kind" aria-hidden="true" />{{ t(item.shortLabel) }} {{ item.count }}</span>
                </div>
              </template>
              <p v-else-if="entry.author.bio" class="author-tile-bio" :title="entry.author.bio">{{ entry.author.bio }}</p>
              <footer class="author-tile-footer">
                <div class="author-tile-sources">
                  <span v-for="site in entry.sites.slice(0, 3)" :key="site" :title="sourceLabel(site)">
                    <img v-if="site !== 'other'" class="author-site-icon" :src="site" alt="">{{ sourceLabel(site) }}
                  </span>
                  <span v-if="entry.sites.length > 3" :title="entry.sites.slice(3).map(sourceLabel).join(' · ')">+{{ entry.sites.length - 3 }}</span>
                </div>
                <small v-if="entry.lastActive">{{ t('最近活跃') }} {{ formatDate(entry.lastActive).slice(0, 10) }}</small>
                <small v-else-if="layout === 'list'">{{ t('无在库作品') }}</small>
              </footer>
            </article>
          </div>
        </section>
      </div>
      <nav class="author-alphabet" :aria-label="t('作者字母索引')">
        <button v-for="letter in letters" :key="letter" type="button" :disabled="!letterGroups.has(letter)"
          :aria-current="selectedLetter === letter ? 'location' : undefined" @click="jumpToLetter(letter)">{{ letter }}</button>
      </nav>
    </div>
  </section>
</template>
