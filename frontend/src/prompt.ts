import type { PromptService } from '../bindings/github.com/Daneliya/prompt-next/internal/desktop'

export type Prompt = NonNullable<Awaited<ReturnType<typeof PromptService.ListPrompts>>>[number]
export type Kind = 'text' | 'art' | 'video' | 'persona'

export function formatPromptNumber(number: number) {
  return '#' + String(number).padStart(6, '0')
}

export function promptCharacterCount(body: string) {
  return Array.from(body.replace(/\s/gu, '')).length
}

export const libraries = [
  { kind: 'text', shortLabel: '文本', label: '文本提示词库', icon: 'text', description: '写作、翻译、学习与日常任务' },
  { kind: 'art', shortLabel: '绘画', label: '绘画提示词库', icon: 'art', description: '画面描述、风格与生成参数' },
  { kind: 'video', shortLabel: '视频', label: '视频提示词库', icon: 'video', description: '镜头、动作、分镜与视频生成提示词' },
  { kind: 'persona', shortLabel: '人设', label: '大肥鱼人设库', icon: 'fish', description: '角色设定与助手指令' },
] as const

export function videoAsset(number: number, name: string) {
  return '/prompt-videos/' + number + '/' + encodeURIComponent(name)
}
