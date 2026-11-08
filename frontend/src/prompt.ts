import type { PromptService } from '../bindings/github.com/Daneliya/prompt-next/internal/desktop'

export type Prompt = NonNullable<Awaited<ReturnType<typeof PromptService.ListPrompts>>>[number]
export type Kind = 'text' | 'art' | 'video' | 'persona'

export function formatPromptNumber(number: number) {
  return '#' + String(number).padStart(6, '0')
}

export function promptCharacterCount(body: string) {
  return Array.from(body.replace(/\s/gu, '')).length
}
