import { watch, type Ref } from 'vue'

export const searchDelay = 160

export function useDebouncedQuery(input: Ref<string>, query: Ref<string>) {
  let timer: ReturnType<typeof setTimeout> | undefined
  function commit() {
    clearTimeout(timer)
    query.value = input.value.trim()
  }

  watch(input, (value, _, onCleanup) => {
    if (!value.trim()) {
      commit()
      return
    }
    timer = setTimeout(commit, searchDelay)
    onCleanup(() => clearTimeout(timer))
  }, { flush: 'sync' })

  return commit
}
