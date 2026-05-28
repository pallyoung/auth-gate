import type { TFunction } from 'i18next'

export type LocalizedText = {
  translationKey?: string
  message?: string
}

export type LocalizedTextState = LocalizedText | null

export class LocalizedError extends Error {
  translationKey?: string

  constructor({ translationKey, message }: LocalizedText) {
    super(message ?? translationKey ?? 'LocalizedError')
    this.name = 'LocalizedError'
    this.translationKey = translationKey
  }
}

export function getLocalizedTextState(error: unknown): Exclude<LocalizedTextState, null> {
  if (error instanceof LocalizedError) {
    return error.translationKey
      ? { translationKey: error.translationKey }
      : { message: error.message }
  }

  if (error instanceof Error) {
    return { message: error.message }
  }

  return { message: String(error) }
}

export function resolveLocalizedText(t: TFunction, state: LocalizedTextState): string {
  if (!state) {
    return ''
  }

  return state.translationKey ? t(state.translationKey as any) : state.message || ''
}
