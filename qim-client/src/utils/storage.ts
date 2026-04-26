/**
 * 本地存储模块 - 基于 IndexedDB
 * 用于缓存会话和消息数据，减少服务端请求
 */

const DB_NAME = 'QIMChatDB'
const DB_VERSION = 1

const CONVERSATIONS_STORE = 'conversations'
const MESSAGES_STORE = 'messages'

let db: IDBDatabase | null = null

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    if (db) {
      resolve(db)
      return
    }

    const request = indexedDB.open(DB_NAME, DB_VERSION)

    request.onupgradeneeded = (event) => {
      const database = (event.target as IDBOpenDBRequest).result

      if (!database.objectStoreNames.contains(CONVERSATIONS_STORE)) {
        database.createObjectStore(CONVERSATIONS_STORE, { keyPath: 'id' })
      }

      if (!database.objectStoreNames.contains(MESSAGES_STORE)) {
        const messageStore = database.createObjectStore(MESSAGES_STORE, { keyPath: 'id' })
        messageStore.createIndex('conversationId', 'conversationId', { unique: false })
        messageStore.createIndex('timestamp', 'timestamp', { unique: false })
      }
    }

    request.onsuccess = (event) => {
      db = (event.target as IDBOpenDBRequest).result
      resolve(db)
    }

    request.onerror = (event) => {
      reject(new Error('IndexedDB 打开失败: ' + (event.target as IDBOpenDBRequest).error?.message))
    }
  })
}

// ============ 会话存储 ============

function toRawClone<T>(obj: T): T {
  return JSON.parse(JSON.stringify(obj))
}

export async function saveConversations(conversations: any[]): Promise<void> {
  try {
    const database = await openDB()
    const tx = database.transaction(CONVERSATIONS_STORE, 'readwrite')
    const store = tx.objectStore(CONVERSATIONS_STORE)

    for (const conv of conversations) {
      store.put(toRawClone(conv))
    }

    return new Promise((resolve, reject) => {
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error)
    })
  } catch (error) {
    console.error('保存会话失败:', error)
  }
}

export async function loadConversations(): Promise<any[]> {
  try {
    const database = await openDB()
    const tx = database.transaction(CONVERSATIONS_STORE, 'readonly')
    const store = tx.objectStore(CONVERSATIONS_STORE)

    return new Promise((resolve, reject) => {
      const request = store.getAll()
      request.onsuccess = () => resolve(request.result)
      request.onerror = () => reject(request.error)
    })
  } catch (error) {
    console.error('加载会话失败:', error)
    return []
  }
}

export async function saveConversation(conversation: any): Promise<void> {
  try {
    const database = await openDB()
    const tx = database.transaction(CONVERSATIONS_STORE, 'readwrite')
    const store = tx.objectStore(CONVERSATIONS_STORE)
    store.put(toRawClone(conversation))

    return new Promise((resolve, reject) => {
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error)
    })
  } catch (error) {
    console.error('保存单个会话失败:', error)
  }
}

export async function deleteConversation(id: string): Promise<void> {
  try {
    const database = await openDB()
    const tx = database.transaction(CONVERSATIONS_STORE, 'readwrite')
    const store = tx.objectStore(CONVERSATIONS_STORE)
    store.delete(id)

    return new Promise((resolve, reject) => {
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error)
    })
  } catch (error) {
    console.error('删除会话失败:', error)
  }
}

// ============ 消息存储 ============

export async function saveMessages(conversationId: string, messages: any[]): Promise<void> {
  try {
    const database = await openDB()
    const tx = database.transaction(MESSAGES_STORE, 'readwrite')
    const store = tx.objectStore(MESSAGES_STORE)
    const index = store.index('conversationId')

    // 先删除该会话的所有旧消息
    const deleteRequest = index.openCursor(IDBKeyRange.only(conversationId))
    const deletePromises: Promise<void>[] = []

    deleteRequest.onsuccess = (event) => {
      const cursor = (event.target as IDBRequest<IDBCursorWithValue>).result
      if (cursor) {
        cursor.delete()
        cursor.continue()
      }
    }

    // 等待删除完成后再插入新消息
    return new Promise((resolve, reject) => {
      tx.oncomplete = async () => {
        try {
          const tx2 = database.transaction(MESSAGES_STORE, 'readwrite')
          const store2 = tx2.objectStore(MESSAGES_STORE)

          for (const msg of messages) {
            store2.put(toRawClone({ ...msg, conversationId }))
          }

          const tx2Complete = new Promise<void>((res, rej) => {
            tx2.oncomplete = () => res()
            tx2.onerror = () => rej(tx2.error)
          })
          await tx2Complete
          resolve()
        } catch (err) {
          reject(err)
        }
      }
      tx.onerror = () => reject(tx.error)
    })
  } catch (error) {
    console.error('保存消息失败:', error)
  }
}

export async function loadMessages(conversationId: string): Promise<any[]> {
  try {
    const database = await openDB()
    const tx = database.transaction(MESSAGES_STORE, 'readonly')
    const store = tx.objectStore(MESSAGES_STORE)
    const index = store.index('conversationId')

    return new Promise((resolve, reject) => {
      const request = index.getAll(IDBKeyRange.only(conversationId))
      request.onsuccess = () => {
        const results = request.result.sort((a, b) => a.timestamp - b.timestamp)
        resolve(results)
      }
      request.onerror = () => reject(request.error)
    })
  } catch (error) {
    console.error('加载消息失败:', error)
    return []
  }
}

export async function addMessage(message: any): Promise<void> {
  try {
    const database = await openDB()
    const tx = database.transaction(MESSAGES_STORE, 'readwrite')
    const store = tx.objectStore(MESSAGES_STORE)
    store.put(toRawClone(message))

    return new Promise((resolve, reject) => {
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error)
    })
  } catch (error) {
    console.error('添加单条消息失败:', error)
  }
}

export async function deleteMessages(conversationId: string): Promise<void> {
  try {
    const database = await openDB()
    const tx = database.transaction(MESSAGES_STORE, 'readwrite')
    const store = tx.objectStore(MESSAGES_STORE)
    const index = store.index('conversationId')

    const request = index.openCursor(IDBKeyRange.only(conversationId))
    request.onsuccess = (event) => {
      const cursor = (event.target as IDBRequest<IDBCursorWithValue>).result
      if (cursor) {
        cursor.delete()
        cursor.continue()
      }
    }

    return new Promise((resolve, reject) => {
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error)
    })
  } catch (error) {
    console.error('删除消息失败:', error)
  }
}

// ============ 工具函数 ============

export async function clearAll(): Promise<void> {
  try {
    const database = await openDB()
    const tx = database.transaction([CONVERSATIONS_STORE, MESSAGES_STORE], 'readwrite')
    tx.objectStore(CONVERSATIONS_STORE).clear()
    tx.objectStore(MESSAGES_STORE).clear()

    return new Promise((resolve, reject) => {
      tx.oncomplete = () => resolve()
      tx.onerror = () => reject(tx.error)
    })
  } catch (error) {
    console.error('清空存储失败:', error)
  }
}

export async function getDBSize(): Promise<{ conversations: number; messages: number }> {
  try {
    const database = await openDB()

    const convCount = await new Promise<number>((resolve, reject) => {
      const tx = database.transaction(CONVERSATIONS_STORE, 'readonly')
      const request = tx.objectStore(CONVERSATIONS_STORE).count()
      request.onsuccess = () => resolve(request.result)
      request.onerror = () => reject(request.error)
    })

    const msgCount = await new Promise<number>((resolve, reject) => {
      const tx = database.transaction(MESSAGES_STORE, 'readonly')
      const request = tx.objectStore(MESSAGES_STORE).count()
      request.onsuccess = () => resolve(request.result)
      request.onerror = () => reject(request.error)
    })

    return { conversations: convCount, messages: msgCount }
  } catch (error) {
    console.error('获取存储统计失败:', error)
    return { conversations: 0, messages: 0 }
  }
}
