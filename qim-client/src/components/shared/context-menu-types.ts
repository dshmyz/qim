export interface ContextMenuItem {
  label?: string
  icon?: string
  iconColor?: string
  action?: () => void
  visible?: boolean
  divider?: boolean
  danger?: boolean
}
