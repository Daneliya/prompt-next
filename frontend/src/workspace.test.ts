import { effectScope, nextTick, ref } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { translateBackendError } from './backendErrors'
import { searchDelay, useDebouncedQuery } from './composables/useDebouncedQuery'
import { useStorageStats } from './composables/useStorageStats'
import { promptSourceIcon } from './promptSourceIcons'

const { storageSnapshot } = vi.hoisted(() => ({ storageSnapshot: vi.fn() }))
vi.mock('../bindings/github.com/Daneliya/prompt-next/internal/desktop', () => ({
  PromptService: { StorageSnapshot: storageSnapshot },
}))

afterEach(() => { vi.useRealTimers(); vi.resetAllMocks() })

describe('live search', () => {
  it('debounces typing, applies explicit search, and clears immediately', () => {
    vi.useFakeTimers()
    const scope = effectScope()
    const input = ref('')
    const query = ref('')
    const commit = scope.run(() => useDebouncedQuery(input, query))!
    input.value = ' first '
    vi.advanceTimersByTime(searchDelay - 1)
    input.value = ' second '
    vi.advanceTimersByTime(searchDelay - 1)
    expect(query.value).toBe('')
    vi.advanceTimersByTime(1)
    expect(query.value).toBe('second')
    input.value = ' third '
    commit()
    expect(query.value).toBe('third')
    input.value = ' '
    expect(query.value).toBe('')
    vi.runAllTimers()
    expect(query.value).toBe('')
    scope.stop()
  })
})

describe('backend errors and source icons', () => {
  it('translates common and wrapped errors while preserving unknown details', () => {
    expect(translateBackendError('时间格式无效')).toBe('Invalid date or time format.')
    expect(translateBackendError('无法打开所选数据目录：请选择有效的文件夹'))
      .toBe('Could not open the selected data folder: Select a valid folder.')
    expect(translateBackendError('I/O error')).toBe('I/O error')
  })

  it('matches only the configured site and its subdomains', () => {
    expect(promptSourceIcon('www.github.com')).toBe('/icons/sites/github.png')
    expect(promptSourceIcon('github.com.example.org')).toBe('')
  })
})

describe('storage statistics', () => {
  it('defers refresh while hidden and reuses cached media totals after non-media writes', async () => {
    storageSnapshot.mockResolvedValueOnce({ dbBytes: 1, imageBytes: 100, imageFiles: 1 })
      .mockResolvedValueOnce({ dbBytes: 2, imageBytes: 0, imageFiles: 0 })
      .mockResolvedValueOnce({ dbBytes: 3, imageBytes: 500, imageFiles: 2 })
    const scope = effectScope()
    const visible = ref(true)
    const state = scope.run(() => useStorageStats(visible, error => { throw error }))!
    await state.refresh()
    expect(storageSnapshot).toHaveBeenLastCalledWith(true)
    visible.value = false
    await nextTick()
    state.invalidate()
    expect(storageSnapshot).toHaveBeenCalledTimes(1)
    visible.value = true
    await nextTick()
    await state.refresh()
    expect(storageSnapshot).toHaveBeenLastCalledWith(false)
    expect(state.stats.value).toMatchObject({ dbBytes: 2, imageBytes: 100, imageFiles: 1 })
    state.invalidate(true)
    await state.refresh()
    expect(storageSnapshot).toHaveBeenLastCalledWith(true)
    expect(state.stats.value).toMatchObject({ dbBytes: 3, imageBytes: 500, imageFiles: 2 })
    scope.stop()
  })
})
