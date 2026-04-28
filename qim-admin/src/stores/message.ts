import { defineStore } from 'pinia'
import { ref } from 'vue'
import { searchMessages, getMessageDetail } from '@/api/messages'
import type { Message, MessageSearchParams } from '@/types/message'

export const useMessageStore = defineStore('message', () => {
  const messages = ref<Message[]>([])
  const total = ref(0)
  const loading = ref(false)
  const currentMessage = ref<Message | null>(null)

  async function search(params: MessageSearchParams) {
    loading.value = true
    try {
      const { data } = await searchMessages(params)
      messages.value = data.list
      total.value = data.total
    } finally {
      loading.value = false
    }
  }

  async function getDetail(id: number) {
    loading.value = true
    try {
      const { data } = await getMessageDetail(id)
      currentMessage.value = data
    } finally {
      loading.value = false
    }
  }

  return {
    messages,
    total,
    loading,
    currentMessage,
    search,
    getDetail,
  }
})
