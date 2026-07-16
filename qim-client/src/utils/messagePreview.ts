import { resolveMessageDisplay, type MessageDisplayKind } from './messageDisplay'

export type MessagePreview = {
  kind: MessageDisplayKind
  label: string
}

type MessagePreviewInput = {
  type: string
  content: string
}

export const getMessagePreview = (input: MessagePreviewInput): MessagePreview => {
  const { kind, label } = resolveMessageDisplay(input)
  return { kind, label }
}
