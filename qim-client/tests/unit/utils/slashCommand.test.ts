import { describe, it, expect } from 'vitest'
import {
  findActiveSlashToken,
  findCommandList,
  replaceSlashToken,
  SlashCommandRegistry,
  type SlashCommand,
  type SlashCommandItem
} from '../../../src/utils/slashCommand'

// ============ findActiveSlashToken ============
describe('findActiveSlashToken', () => {
  const trigger = '/task'

  it('文本开头的 /task 触发，query 为空', () => {
    const text = '/task '
    const token = findActiveSlashToken(text, text.length, trigger)
    expect(token).not.toBeNull()
    expect(token!.start).toBe(0)
    expect(token!.query).toBe('')
  })

  it('空白字符后的 /task 触发', () => {
    const text = 'hello /task '
    const token = findActiveSlashToken(text, text.length, trigger)
    expect(token).not.toBeNull()
    expect(token!.start).toBe(6)
  })

  it('/task 后接搜索词触发，query 去除前导空白', () => {
    const text = '/task 登录'
    const token = findActiveSlashToken(text, text.length, trigger)
    expect(token).not.toBeNull()
    expect(token!.query).toBe('登录')
  })

  it('/task 后紧跟非空白字符不触发（如 /taskhello）', () => {
    const text = '/taskhello'
    const token = findActiveSlashToken(text, text.length, trigger)
    expect(token).toBeNull()
  })

  it('跨行不触发', () => {
    const text = '/task\nhello'
    const token = findActiveSlashToken(text, text.length, trigger)
    expect(token).toBeNull()
  })

  it('URL 路径里的 / 不误触发（前面非空白）', () => {
    const text = 'http://example.com/task '
    const token = findActiveSlashToken(text, text.length, trigger)
    expect(token).toBeNull()
  })

  it('光标位于 /task 中间时不触发（trigger 未完整输入）', () => {
    const text = '/task'
    // 光标在 /t 处
    const token = findActiveSlashToken(text, 2, trigger)
    expect(token).toBeNull()
  })

  it('支持任意 trigger（如 /channel）', () => {
    const text = '/channel 通知'
    const token = findActiveSlashToken(text, text.length, '/channel')
    expect(token).not.toBeNull()
    expect(token!.query).toBe('通知')
  })

  it('光标越界返回 null', () => {
    expect(findActiveSlashToken('/task ', 100, trigger)).toBeNull()
    expect(findActiveSlashToken('/task ', -1, trigger)).toBeNull()
  })

  it('最近一个 trigger 优先（光标前有多个 /task）', () => {
    const text = '/task foo /task '
    const token = findActiveSlashToken(text, text.length, trigger)
    expect(token).not.toBeNull()
    expect(token!.start).toBe(10)
  })
})

// ============ replaceSlashToken ============
describe('replaceSlashToken', () => {
  const trigger = '/task'

  it('把 token 替换为插入文本', () => {
    const text = '/task foo'
    const token = findActiveSlashToken(text, text.length, trigger)!
    const result = replaceSlashToken(text, token, '#T-123 ')
    expect(result).toBe('#T-123 ')
  })

  it('保留 token 前后的文本', () => {
    // 'hello /task end'，光标在 ' end' 的 e 前（即 /task 后的空格处）
    const text = 'hello /task end'
    const token = findActiveSlashToken(text, 12, trigger)!
    const result = replaceSlashToken(text, token, '#T-5 ')
    expect(result).toBe('hello #T-5 end')
  })
})

// ============ SlashCommandRegistry ============
interface TaskItem extends SlashCommandItem {
  id: number
  title: string
}

describe('SlashCommandRegistry', () => {
  function makeRegistry() {
    const registry = new SlashCommandRegistry()
    const taskCommand: SlashCommand<TaskItem> = {
      trigger: '/task',
      title: '选择任务',
      fetchItems: async () => [
        { id: 1, title: '登录' },
        { id: 2, title: '注册' }
      ],
      filter: (items, query) => {
        const q = query.trim().toLowerCase()
        if (!q) return items
        return items.filter(i => i.title.toLowerCase().includes(q))
      },
      getInsertText: item => `#T-${item.id} `
    }
    const channelCommand: SlashCommand<SlashCommandItem> = {
      trigger: '/channel',
      title: '选择频道',
      fetchItems: async () => [],
      filter: items => items,
      getInsertText: item => `#C-${item.id} `
    }
    registry.register(taskCommand)
    registry.register(channelCommand)
    return registry
  }

  it('register + getByTrigger 查找命令', () => {
    const registry = makeRegistry()
    expect(registry.getByTrigger('/task')).toBeDefined()
    expect(registry.getByTrigger('/channel')).toBeDefined()
    expect(registry.getByTrigger('/unknown')).toBeUndefined()
  })

  it('list 返回所有已注册命令', () => {
    const registry = makeRegistry()
    expect(registry.list().map(c => c.trigger).sort()).toEqual(['/channel', '/task'])
  })

  it('findActive 在文本中识别活跃命令', () => {
    const registry = makeRegistry()
    const ctx = { conversationId: '1', conversationType: 'group' }
    const result = registry.findActive('/task 登录', 8, ctx)
    expect(result).not.toBeNull()
    expect(result!.command.trigger).toBe('/task')
    expect(result!.token.query).toBe('登录')
  })

  it('findActive 识别不同命令', () => {
    const registry = makeRegistry()
    const ctx = { conversationId: '1', conversationType: 'group' }
    const result = registry.findActive('/channel 通知', 11, ctx)
    expect(result).not.toBeNull()
    expect(result!.command.trigger).toBe('/channel')
  })

  it('findActive 无匹配返回 null', () => {
    const registry = makeRegistry()
    const ctx = { conversationId: '1', conversationType: 'group' }
    expect(registry.findActive('hello world', 11, ctx)).toBeNull()
  })

  it('available 返回 false 的命令不会被 findActive 匹配', () => {
    const registry = new SlashCommandRegistry()
    registry.register({
      trigger: '/task',
      title: '选择任务',
      available: ctx => ctx.conversationType === 'group',
      fetchItems: async () => [],
      filter: items => items,
      getInsertText: item => `#T-${item.id} `
    })
    // 单聊不触发
    expect(registry.findActive('/task ', 6, { conversationId: '1', conversationType: 'single' })).toBeNull()
    // 群聊触发
    expect(registry.findActive('/task ', 6, { conversationId: '1', conversationType: 'group' })).not.toBeNull()
  })
})

// ============ findCommandList ============
describe('findCommandList', () => {
  function makeRegistry() {
    const registry = new SlashCommandRegistry()
    registry.register({
      trigger: '/task',
      title: '选择任务',
      fetchItems: async () => [],
      filter: items => items,
      getInsertText: item => `#T-${item.id} `
    })
    registry.register({
      trigger: '/channel',
      title: '选择频道',
      fetchItems: async () => [],
      filter: items => items,
      getInsertText: item => `#C-${item.id} `
    })
    return registry
  }

  it('文本开头的 / 触发命令列表', () => {
    const registry = makeRegistry()
    const ctx = { conversationId: '1', conversationType: 'group' }
    const result = findCommandList('/', 1, ctx, registry)
    expect(result).not.toBeNull()
    expect(result!.token.start).toBe(0)
    expect(result!.token.query).toBe('')
    expect(result!.commands.length).toBe(2)
  })

  it('空白字符后的 / 触发', () => {
    const registry = makeRegistry()
    const ctx = { conversationId: '1', conversationType: 'group' }
    // 'hello /' 长度 7，/ 在 index 6，光标在末尾（/ 之后）= 7
    const result = findCommandList('hello /', 7, ctx, registry)
    expect(result).not.toBeNull()
    expect(result!.token.start).toBe(6)
  })

  it('/ 后接字符不触发命令列表（如 /t），交给 findActive 处理', () => {
    const registry = makeRegistry()
    const ctx = { conversationId: '1', conversationType: 'group' }
    // /t 不是一个完整命令，也不应作为命令列表触发
    expect(findCommandList('/t', 2, ctx, registry)).toBeNull()
  })

  it('/ 后接空白再接字符触发（query 为空白后的内容，用于过滤命令列表）', () => {
    const registry = makeRegistry()
    const ctx = { conversationId: '1', conversationType: 'group' }
    const result = findCommandList('/ t', 3, ctx, registry)
    expect(result).not.toBeNull()
    expect(result!.token.query).toBe('t')
  })

  it('URL 路径里的 / 不误触发（前面非空白）', () => {
    const registry = makeRegistry()
    const ctx = { conversationId: '1', conversationType: 'group' }
    expect(findCommandList('http://example.com/', 19, ctx, registry)).toBeNull()
  })

  it('跨行不触发', () => {
    const registry = makeRegistry()
    const ctx = { conversationId: '1', conversationType: 'group' }
    expect(findCommandList('\n/', 2, ctx, registry)).not.toBeNull() // \n 后的 / 前是空白，触发
    expect(findCommandList('/\nhello', 7, ctx, registry)).toBeNull() // / 后跨行不触发
  })

  it('available 为 false 的命令不出现在列表中', () => {
    const registry = new SlashCommandRegistry()
    registry.register({
      trigger: '/task',
      title: '选择任务',
      available: ctx => ctx.conversationType === 'group',
      fetchItems: async () => [],
      filter: items => items,
      getInsertText: item => `#T-${item.id} `
    })
    // 单聊：列表为空，返回 null（无可选命令）
    expect(findCommandList('/', 1, { conversationId: '1', conversationType: 'single' }, registry)).toBeNull()
    // 群聊：命令可用
    const groupResult = findCommandList('/', 1, { conversationId: '1', conversationType: 'group' }, registry)
    expect(groupResult).not.toBeNull()
    expect(groupResult!.commands.length).toBe(1)
  })

  it('光标越界返回 null', () => {
    const registry = makeRegistry()
    const ctx = { conversationId: '1', conversationType: 'group' }
    expect(findCommandList('/', 100, ctx, registry)).toBeNull()
    expect(findCommandList('/', -1, ctx, registry)).toBeNull()
  })

  it('命令列表按 query 过滤（前端展示过滤由调用方做，此处只返回 token+全量可用命令）', () => {
    const registry = makeRegistry()
    const ctx = { conversationId: '1', conversationType: 'group' }
    const result = findCommandList('/ t', 3, ctx, registry)
    expect(result).not.toBeNull()
    // findCommandList 返回全量可用命令，过滤由 UI 层用 trigger.includes(query) 做
    expect(result!.commands.length).toBe(2)
  })
})

// ============ onSelect（直接发消息动作） ============
describe('onSelect', () => {
  it('提供 onSelect 的命令可返回 send 动作，框架据此直接发消息而非插入文本', () => {
    const registry = new SlashCommandRegistry()
    const cmd: SlashCommand<{ id: number; title: string }> = {
      trigger: '/note',
      title: '选择笔记',
      fetchItems: async () => [{ id: 1, title: '测试笔记' }],
      filter: items => items,
      getInsertText: () => '',
      onSelect: (item) => ({
        action: 'send',
        messageType: 'share',
        messageContent: JSON.stringify({ type: 'note', id: item.id, name: item.title }),
      }),
    }
    registry.register(cmd)

    const active = registry.findActive('/note ', 6, { conversationId: '1', conversationType: 'single' })
    expect(active).not.toBeNull()

    const result = active!.command.onSelect!({ id: 1, title: '测试笔记' }, { conversationId: '1', conversationType: 'single' })
    expect(result.action).toBe('send')
    expect(result.messageType).toBe('share')
    expect(result.messageContent).toContain('"type":"note"')
    expect(result.messageContent).toContain('"id":1')
  })

  it('onSelect 返回 insert 时回退到插入文本', () => {
    const registry = new SlashCommandRegistry()
    const cmd: SlashCommand<{ id: number }> = {
      trigger: '/foo',
      title: 'foo',
      fetchItems: async () => [{ id: 1 }],
      filter: items => items,
      getInsertText: item => `#F-${item.id} `,
      onSelect: () => ({ action: 'insert', insertText: '#F-1 ' }),
    }
    registry.register(cmd)

    const active = registry.findActive('/foo ', 5, { conversationId: '1', conversationType: 'group' })
    expect(active).not.toBeNull()

    const result = active!.command.onSelect!({ id: 1 }, { conversationId: '1', conversationType: 'group' })
    expect(result.action).toBe('insert')
    expect(result.insertText).toBe('#F-1 ')
  })

  it('未提供 onSelect 时框架回退到 getInsertText', () => {
    const registry = new SlashCommandRegistry()
    const cmd: SlashCommand<{ id: number }> = {
      trigger: '/task',
      title: '选择任务',
      fetchItems: async () => [{ id: 1 }],
      filter: items => items,
      getInsertText: item => `#T-${item.id} `,
    }
    registry.register(cmd)

    const active = registry.findActive('/task ', 6, { conversationId: '1', conversationType: 'group' })
    expect(active).not.toBeNull()
    expect(active!.command.onSelect).toBeUndefined()
    // 框架在 onSelect 缺失时走 getInsertText
    expect(active!.command.getInsertText({ id: 1 })).toBe('#T-1 ')
  })
})
