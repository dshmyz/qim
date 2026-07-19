type SelectableMessage = {
  type?: string
  isRecalled?: boolean
}

/** Whether a message can participate in merged forwarding selection. */
export const isMessageSelectionEligible = (message: SelectableMessage): boolean =>
  !message.isRecalled && message.type !== 'system'
