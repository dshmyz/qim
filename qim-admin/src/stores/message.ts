import { defineStore } from 'pinia'
import { ref } from 'vue'
import { searchMessages } from '@/api/messages'
import type { Message, MessageSearchParams } from '@/types/message'

export const useMessageStore = defineStore('message', () => {
  const messages = ref<Message[]>([])
  const total = ref(0)
  const loading = ref(false)

  async function search(params: MessageSearchParams) {
    loading.value = true
    try {
      const { data } = await searchMessages(params)
      messages.value = data.data.list
      total.value = data.data.total
    } finally {
      loading.value = false
    }
  }

  return {
    messages,
    total,
    loading,
    search,
  }
})
