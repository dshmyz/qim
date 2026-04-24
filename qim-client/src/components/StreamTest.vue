<template>
  <div class="stream-test">
    <h2>流式消息测试</h2>
    <button @click="testStream">测试流式消息</button>
    <div class="stream-output">
      <StreamingMessage
        v-if="streamMessage"
        :content="streamMessage.content"
        :is-self="false"
        :is-streaming="streamMessage.isStreaming"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import StreamingMessage from './message/StreamingMessage.vue'

const streamMessage = ref(null)

const testStream = async () => {
  // 创建初始流式消息
  streamMessage.value = {
    content: '',
    isStreaming: true
  }
  
  // 模拟流式数据
  const response = "你好！我是AI助手，很高兴为你服务。我可以帮你解答问题、提供信息或者进行闲聊。请问有什么我可以帮助你的吗？"
  
  let accumulatedContent = ''
  for (const char of response) {
    accumulatedContent += char
    streamMessage.value.content = accumulatedContent
    await new Promise(resolve => setTimeout(resolve, 30))
  }
  
  // 流式结束
  streamMessage.value.isStreaming = false
}
</script>

<style scoped>
.stream-test {
  padding: 20px;
  border: 1px solid #ddd;
  border-radius: 8px;
  margin: 20px;
}

.stream-output {
  margin-top: 20px;
  min-height: 100px;
  border: 1px solid #eee;
  border-radius: 4px;
  padding: 10px;
}
</style>