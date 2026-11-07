import type { PromptService } from '../bindings/github.com/Daneliya/prompt-next/internal/desktop'

export type Prompt = NonNullable<Awaited<ReturnType<typeof PromptService.ListPrompts>>>[number]
export type Kind = 'persona' | 'art'

export function formatPromptNumber(number: number) {
  return '#' + String(number).padStart(4, '0')
}
