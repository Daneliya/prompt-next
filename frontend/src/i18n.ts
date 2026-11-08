import { ref, watchEffect } from 'vue'
import { english } from './english'

export const language = ref<'zh-CN' | 'en-US'>('zh-CN')

watchEffect(() => { document.documentElement.lang = language.value })

export function t(key?: string, ...values: unknown[]): string {
  if (!key) return ''
  const text = language.value === 'en-US' && Object.hasOwn(english, key) ? english[key]! : key
  return text.replace(/\{(\d+)\}/g, (_, index: string) => String(values[Number(index)] ?? ''))
}
