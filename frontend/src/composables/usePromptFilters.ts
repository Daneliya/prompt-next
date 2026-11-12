import { computed, ref, type Ref } from 'vue'
import { formatPromptNumber, type Prompt } from '../prompt'

export function promptModels(item: Prompt) {
  return item.models ?? (item.model ? [item.model] : [])
}

export function countLabels(values: string[]) {
  const counts = new Map<string, number>()
  for (const name of values) counts.set(name, (counts.get(name) ?? 0) + 1)
  return [...counts].map(([name, count]) => ({ name, count }))
}

const searchFields = new WeakMap<Prompt, string[]>()

function searchableFields(item: Prompt) {
  let fields = searchFields.get(item)
  if (!fields) {
    fields = [item.title, item.body, item.english, item.description, item.category, ...(item.tags ?? []),
      item.negative, item.parameters, item.author, item.authorUrl, formatPromptNumber(item.number),
      '#' + item.number, ...promptModels(item)]
      .filter((value): value is string => typeof value === 'string')
      .map(value => value.toLowerCase())
    searchFields.set(item, fields)
  }
  return fields
}

export function usePromptFilters(items: Ref<Prompt[]>) {
  const sort = ref('updated')
  const modelFilter = ref<string | null>(null)
  const tagFilter = ref<string | null>(null)
  const category = ref('')
  const favoritesOnly = ref(false)
  const query = ref('')
  const results = computed(() => {
    const keyword = query.value.trim().toLowerCase()
    return items.value.filter((item) =>
      (!favoritesOnly.value || item.favorite) &&
      (!category.value || item.category === category.value) &&
      (!modelFilter.value || promptModels(item).includes(modelFilter.value)) &&
      (!tagFilter.value || item.tags?.includes(tagFilter.value)) &&
      (!keyword || searchableFields(item).some(value => value.includes(keyword)))
    ).sort((a, b) => {
      if (sort.value === 'title') return a.title.localeCompare(b.title, 'zh-CN')
      if (sort.value === 'created') return b.createdAt.localeCompare(a.createdAt)
      if (sort.value === 'oldest') return a.createdAt.localeCompare(b.createdAt)
      return b.updatedAt.localeCompare(a.updatedAt)
    })
  })
  return { sort, modelFilter, tagFilter, category, favoritesOnly, query, results }
}
