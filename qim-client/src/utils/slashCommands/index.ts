/**
 * 斜杠命令注册中心单例。
 *
 * 应用启动时注册所有内置命令，UI 层通过此单例查找活跃命令。
 * 新增命令只需在此处 register，无需改动输入框骨架。
 */

import { SlashCommandRegistry, type SlashCommand } from '../slashCommand'
import { taskCommand } from './taskCommand'
import { noteCommand } from './noteCommand'
import { fileCommand } from './fileCommand'
import { quickCommand } from './quickCommand'

const registry = new SlashCommandRegistry()

/** 注册内置命令。幂等，重复注册同 trigger 会覆盖。 */
export function registerBuiltinCommands(): void {
  registry.register(taskCommand)
  registry.register(noteCommand)
  registry.register(fileCommand)
  registry.register(quickCommand)
}

/** 获取全局注册中心单例。 */
export function getSlashCommandRegistry(): SlashCommandRegistry {
  return registry
}

/** 注册自定义命令（供扩展使用）。 */
export function registerSlashCommand<I extends import('../slashCommand').SlashCommandItem>(
  command: SlashCommand<I>
): void {
  registry.register(command)
}

// 模块加载时注册内置命令
registerBuiltinCommands()
