// 简化版本地存储模块 - 基于 localStorage

export async function saveConversations(conversations) {
  try {
    localStorage.setItem('qim_conversations', JSON.stringify(conversations));
  } catch (error) {
    console.warn('保存会话失败:', error);
  }
}

export async function loadConversations() {
  try {
    const data = localStorage.getItem('qim_conversations');
    return data ? JSON.parse(data) : [];
  } catch (error) {
    console.warn('加载会话失败:', error);
    return [];
  }
}

export async function saveConversation(conversation) {
  try {
    const conversations = await loadConversations();
    const index = conversations.findIndex(c => c.id === conversation.id);
    if (index !== -1) {
      conversations[index] = conversation;
    } else {
      conversations.push(conversation);
    }
    localStorage.setItem('qim_conversations', JSON.stringify(conversations));
  } catch (error) {
    console.warn('保存单个会话失败:', error);
  }
}

export async function deleteConversation(id) {
  try {
    const conversations = await loadConversations();
    const filtered = conversations.filter(c => c.id !== id);
    localStorage.setItem('qim_conversations', JSON.stringify(filtered));
  } catch (error) {
    console.warn('删除会话失败:', error);
  }
}

export async function saveSyncState(conversationId, syncState) {
  try {
    const syncStates = await loadSyncStates();
    syncStates[conversationId] = syncState;
    localStorage.setItem('qim_sync_states', JSON.stringify(syncStates));
  } catch (error) {
    console.warn('保存同步状态失败:', error);
  }
}

export async function loadSyncStates() {
  try {
    const data = localStorage.getItem('qim_sync_states');
    return data ? JSON.parse(data) : {};
  } catch (error) {
    console.warn('加载同步状态失败:', error);
    return {};
  }
}

export async function getSyncState(conversationId) {
  try {
    const syncStates = await loadSyncStates();
    const state = syncStates[conversationId];
    return state ? { conversationId, ...state } : null;
  } catch (error) {
    console.warn('获取同步状态失败:', error);
    return null;
  }
}

export async function deleteSyncState(conversationId) {
  try {
    const syncStates = await loadSyncStates();
    delete syncStates[conversationId];
    localStorage.setItem('qim_sync_states', JSON.stringify(syncStates));
  } catch (error) {
    console.warn('删除同步状态失败:', error);
  }
}

export async function clearSyncStates() {
  try {
    localStorage.removeItem('qim_sync_states');
  } catch (error) {
    console.warn('清空同步状态失败:', error);
  }
}

export async function clearAll() {
  try {
    localStorage.removeItem('qim_conversations');
    localStorage.removeItem('qim_sync_states');
  } catch (error) {
    console.warn('清空存储失败:', error);
  }
}
