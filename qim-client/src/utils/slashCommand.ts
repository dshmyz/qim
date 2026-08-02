/**
 * 斜杠命令框架：统一的"斜杠触发 → 搜索面板 → 插入 #X-<id>"补全链路。
 *
 * 设计原则：
 *   - 纯函数 + 注册中心，不依赖 Vue 运行时
 *   - token 检测/替换逻辑泛化，任意 trigger 复用
 *   - 命令的具体 UI（候选项渲染）由调用方提供组件
 *
 * 与 utils/mentions.ts 的 @ 提及同层但独立：
 *   - @ 提及有 span 跟踪、序列化等特殊逻辑，不纳入本框架
 *   - 模式切换类命令（如 /code）形状不同，也不纳入
 */

import type { Component } from 'vue'

/** 候选项基类：命令实现可扩展更多字段。 */
export interface SlashCommandItem {
  id: string | number
}

/** 光标所在的、可用于补全的斜杠 token。 */
export interface ActiveSlashToken {
  /** trigger 的起始位置（/ 的位置） */
  start: number
  /** 光标位置 */
  end: number
  /** trigger 之后的搜索词（已去除前导空白） */
  query: string
}

/** 命令触发时的上下文，用于 available 判断和拉取候选数据。 */
export interface SlashCommandContext {
  conversationId: string
  conversationType: string
}

/** 选中候选项后的动作结果：返回 true 表示已处理（如直接发消息），框架不再插入文本。 */
export type SlashCommandAction = 'send' | 'insert'

/** 选中候选项后的处理结果。 */
export interface SlashCommandSelectResult {
  /** 动作类型：send=直接发送消息，insert=插入文本到输入框 */
  action: SlashCommandAction
  /** action='send' 时：消息内容（如 share 消息的 JSON 字符串） */
  messageContent?: string
  /** action='send' 时：消息类型（如 'share'） */
  messageType?: string
  /** action='insert' 时：要插入的文本 */
  insertText?: string
}

/** 斜杠命令定义。I 为候选项类型。 */
export interface SlashCommand<I extends SlashCommandItem = SlashCommandItem> {
  /** 触发词，如 '/task' */
  trigger: string
  /** 面板标题，如 '选择任务' */
  title: string
  /** 命令图标（FontAwesome 类名），用于命令列表展示。可选。 */
  icon?: string
  /** 命令描述，用于命令列表展示。可选。 */
  description?: string
  /** 是否在当前上下文可用。未提供时视为始终可用。 */
  available?: (ctx: SlashCommandContext) => boolean
  /**
   * 是否使用后端搜索。
   * - true：每次 query 变化都重新调用 fetchItems(ctx, query)（由调用方 debounce），
   *   并跳过前端 filter（直接展示 fetchItems 返回的结果）。
   * - false/未设置：只在进入搜索模式时调用一次 fetchItems(ctx)，之后用 filter 做前端过滤。
   */
  backendSearch?: boolean
  /**
   * 拉取候选项（通常带缓存）。
   * query 仅在 backendSearch=true 时由调用方传入，用于后端搜索；
   * 其他命令可忽略 query 参数。
   */
  fetchItems: (ctx: SlashCommandContext, query?: string) => Promise<I[]>
  /** 按 query 过滤候选项（前端同步过滤）。backendSearch=true 时不会被调用。 */
  filter: (items: I[], query: string) => I[]
  /** 选中后插入到输入框的文本，如 '#T-123 ' */
  getInsertText: (item: I) => string
  /**
   * 选中后的自定义处理（可选）。返回 'send' 时框架直接发送消息，不再插入文本；
   * 返回 'insert' 或未提供时，回退到 getInsertText 插入文本。
   * 用于 /note 这类"选中即发送分享消息"的场景。
   */
  onSelect?: (item: I, ctx: SlashCommandContext) => SlashCommandSelectResult
  /** 渲染单条候选项的组件，props: { item: I, active: boolean } */
  itemComponent: Component
}

/**
 * 查找光标所在的某个 trigger 的 token。
 *
 * trigger 必须位于文本开头或空白字符之后；trigger 之后要么为空（光标紧接），
 * 要么以空白分隔再接搜索词。跨行或 trigger 后紧跟非空白字符时不触发。
 */
export function findActiveSlashToken(
  text: string,
  cursor: number,
  trigger: string
): ActiveSlashToken | null {
  if (cursor < 0 || cursor > text.length) return null
  if (!trigger) return null

  const before = text.slice(0, cursor)
  const triggerIdx = before.lastIndexOf(trigger)
  if (triggerIdx === -1) return null

  // trigger 前必须是文本开头或空白（避免 URL 路径里的 / 误触发）
  if (triggerIdx > 0 && !/\s/.test(text[triggerIdx - 1])) return null

  const afterTrigger = text.slice(triggerIdx + trigger.length, cursor)

  // trigger 后紧跟非空白字符（如 /taskhello）不触发
  if (afterTrigger.length > 0 && !/^\s/.test(afterTrigger)) return null

  // 跨行不触发
  if (/[\n\r]/.test(afterTrigger)) return null

  return {
    start: triggerIdx,
    end: cursor,
    query: afterTrigger.replace(/^\s+/, ''),
  }
}

/**
 * 将斜杠 token 整体替换为插入文本。
 */
export function replaceSlashToken(
  text: string,
  token: Pick<ActiveSlashToken, 'start' | 'end'>,
  insertText: string
): string {
  return text.slice(0, token.start) + insertText + text.slice(token.end)
}

/** findActive 的返回结构：命中的命令 + token。 */
export interface ActiveSlashCommand<I extends SlashCommandItem = SlashCommandItem> {
  command: SlashCommand<I>
  token: ActiveSlashToken
}

/**
 * 斜杠命令注册中心。
 * 统一管理已注册命令，提供按文本+光标位置查找活跃命令的能力。
 */
export class SlashCommandRegistry {
  private commands = new Map<string, SlashCommand>()

  /** 注册一个命令。同 trigger 后注册的覆盖前者。 */
  register<I extends SlashCommandItem>(command: SlashCommand<I>): void {
    this.commands.set(command.trigger, command as unknown as SlashCommand)
  }

  /** 按 trigger 查找命令。 */
  getByTrigger(trigger: string): SlashCommand | undefined {
    return this.commands.get(trigger)
  }

  /** 所有已注册命令。 */
  list(): SlashCommand[] {
    return Array.from(this.commands.values())
  }

  /**
   * 在文本中查找光标所在的活跃命令。
   *
   * 遍历所有可用命令，命中 token 检测的返回。
   * 多命令同时匹配时，取最近一个 trigger 命中的命令（各命令 trigger 不同，
   * 实际只会有一个命令的 trigger 出现在光标前）。
   */
  findActive<I extends SlashCommandItem>(
    text: string,
    cursor: number,
    ctx: SlashCommandContext
  ): ActiveSlashCommand<I> | null {
    for (const command of this.commands.values()) {
      // 先按 available 过滤，避免对不可用命令做 token 检测
      if (command.available && !command.available(ctx)) continue

      const token = findActiveSlashToken(text, cursor, command.trigger)
      if (token) {
        return { command: command as unknown as SlashCommand<I>, token }
      }
    }
    return null
  }

  /**
   * 返回在当前上下文可用的所有命令（供命令列表展示）。
   */
  listAvailable(ctx: SlashCommandContext): SlashCommand[] {
    return this.list().filter(cmd => !cmd.available || cmd.available(ctx))
  }
}

/** findCommandList 的返回结构：命令列表 token + 可用命令列表。 */
export interface ActiveCommandList {
  /** / 的位置和光标位置，以及 / 后的 query（用于过滤命令列表） */
  token: ActiveSlashToken
  /** 当前上下文可用的命令列表（已按 available 过滤） */
  commands: SlashCommand[]
}

/**
 * 查找光标所在的"命令列表触发点"。
 *
 * 触发条件：光标前是孤立的 / （位于文本开头或空白字符之后），且 / 后到光标
 * 之间要么为空，要么以空白分隔再接字符（用于过滤命令列表）。
 *
 * 与 findActiveSlashToken 的区别：
 *   - findActiveSlashToken 需要完整 trigger（如 /task）才触发，进入具体命令搜索
 *   - findCommandList 只需 / 即触发，展示所有可用命令供选择
 *
 * 典型流程：
 *   输入 /        → findCommandList 命中 → 展示命令列表
 *   输入 /t       → findCommandList 不命中（/ 后紧跟非空白字符），
 *                   findActiveSlashToken 也不命中 → 无面板
 *   输入 / task   → findCommandList 命中（query="task"）→ 命令列表按 query 过滤
 *   输入 /task    → findActiveSlashToken 命中 → 进入任务搜索面板
 */
export function findCommandList(
  text: string,
  cursor: number,
  ctx: SlashCommandContext,
  registry: SlashCommandRegistry
): ActiveCommandList | null {
  // 复用 findActiveSlashToken 的检测逻辑，trigger 固定为 '/'
  const token = findActiveSlashToken(text, cursor, '/')
  if (!token) return null

  const commands = registry.listAvailable(ctx)
  if (commands.length === 0) return null

  return { token, commands }
}
