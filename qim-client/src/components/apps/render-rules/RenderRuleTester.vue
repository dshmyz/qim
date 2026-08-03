<template>
  <div class="render-rule-tester">
    <div class="rr-tester-input">
      <label class="rr-label">样例文本</label>
      <textarea
        v-model="sampleText"
        class="rr-input rr-textarea"
        rows="3"
        placeholder="输入一段文本测试规则匹配，如：请查看 NI-30000 这个工单"
      ></textarea>
      <button
        type="button"
        class="rr-btn rr-btn-primary rr-tester-btn"
        :disabled="testing || !canTest"
        @click="runTest"
      >
        <i v-if="testing" class="fas fa-spinner fa-spin"></i>
        <i v-else class="fas fa-play"></i>
        {{ testing ? '测试中...' : '测试匹配' }}
      </button>
    </div>

    <div v-if="results.length > 0" class="rr-tester-results">
      <div class="rr-tester-results-title">
        匹配到 {{ results.length }} 个结果
      </div>
      <div v-for="(r, i) in results" :key="i" class="rr-tester-result">
        <div class="rr-tester-result-row">
          <span class="rr-tester-label">匹配文本</span>
          <code class="rr-tester-matched">{{ r.matched }}</code>
        </div>
        <div class="rr-tester-result-row">
          <span class="rr-tester-label">渲染标签</span>
          <span class="rr-tester-rendered">{{ r.label }}</span>
        </div>
        <div v-if="r.url" class="rr-tester-result-row">
          <span class="rr-tester-label">跳转链接</span>
          <a :href="r.url" target="_blank" rel="noopener" class="rr-tester-url">{{ r.url }}</a>
        </div>
      </div>
    </div>

    <div v-if="tested && results.length === 0" class="rr-tester-empty">
      <i class="fas fa-info-circle"></i>
      <span>未匹配到任何内容</span>
    </div>

    <div v-if="errorMsg" class="rr-tester-error">
      <i class="fas fa-exclamation-circle"></i>
      <span>{{ errorMsg }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import type { RenderRule } from '../../../stores/renderRules'
import { testRenderRule, type TestRuleResult } from '../../../api/renderRules'

const props = defineProps<{
  rule: RenderRule | null
}>()

const sampleText = ref('')
const testing = ref(false)
const tested = ref(false)
const results = ref<TestRuleResult[]>([])
const errorMsg = ref('')

const canTest = computed(() => {
  return props.rule && props.rule.match.pattern && sampleText.value.trim()
})

async function runTest() {
  if (!props.rule || !canTest.value) return
  testing.value = true
  tested.value = false
  errorMsg.value = ''
  results.value = []

  try {
    results.value = await testRenderRule(props.rule, sampleText.value)
    tested.value = true
  } catch (e) {
    errorMsg.value = e instanceof Error ? e.message : '测试请求失败'
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.render-rule-tester {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.rr-tester-input {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.rr-label {
  font-size: 13px;
  color: #606266;
  font-weight: 500;
}
.rr-textarea {
  resize: vertical;
  min-height: 60px;
  font-family: inherit;
}
.rr-input {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  font-size: 13px;
  color: #303133;
  background: #fff;
  box-sizing: border-box;
}
.rr-input:focus {
  outline: none;
  border-color: #409eff;
}
.rr-tester-btn {
  align-self: flex-start;
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  border: none;
  background: #409eff;
  color: #fff;
  display: flex;
  align-items: center;
  gap: 6px;
}
.rr-tester-btn:disabled {
  background: #a0cfff;
  cursor: not-allowed;
}
.rr-tester-results {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.rr-tester-results-title {
  font-size: 13px;
  color: #67c23a;
  font-weight: 600;
}
.rr-tester-result {
  background: #f0f9ff;
  border: 1px solid #d9ecff;
  border-radius: 6px;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.rr-tester-result-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}
.rr-tester-label {
  color: #909399;
  min-width: 70px;
  flex-shrink: 0;
}
.rr-tester-matched {
  background: #e6f7ff;
  padding: 1px 6px;
  border-radius: 3px;
  color: #1890ff;
  font-size: 12px;
}
.rr-tester-rendered {
  color: #303133;
  font-weight: 500;
}
.rr-tester-url {
  color: #409eff;
  text-decoration: none;
  font-size: 12px;
  word-break: break-all;
}
.rr-tester-url:hover {
  text-decoration: underline;
}
.rr-tester-empty {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  background: #f4f4f5;
  border-radius: 6px;
  color: #909399;
  font-size: 13px;
}
.rr-tester-error {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  background: #fef0f0;
  border: 1px solid #fde2e2;
  border-radius: 6px;
  color: #f56c6c;
  font-size: 13px;
}
</style>
