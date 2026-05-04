# 实时通信重构 - 迁移计划

## 当前状态

✅ 阶段 1-4 已完成：
- 类型定义和文档
- 基础设施层 composables
- 能力层和会话层
- 业务层 composables

⏳ 阶段 5 待完成：
- 服务器端消息格式更新
- 客户端组件迁移
- 测试和验证

## 迁移策略

### 原则

1. **渐进式迁移** - 新旧代码可以共存，逐步替换
2. **保持功能** - 迁移过程中不影响现有功能
3. **充分测试** - 每次迁移后都要完整测试
4. **可回退** - 随时可以回退到旧版本

### 步骤

#### 步骤 1：更新服务器端消息处理

**目标：** 支持新的消息格式，同时兼容旧格式

**文件：** `qim-server/ws/ws.go`

**修改：**

```go
// 在 handleWebRTCSignal 函数中
// 检查并转发 media_type 字段
if mediaType, ok := msgData["media_type"]; ok {
    forwardData["media_type"] = mediaType
}

// 兼容旧的 share_type 和 call_type 字段
if shareType, ok := msgData["share_type"]; ok {
    forwardData["share_type"] = shareType
    // 同时设置 media_type
    forwardData["media_type"] = shareType
}
if callType, ok := msgData["call_type"]; ok {
    forwardData["call_type"] = callType
    // 同时设置 media_type
    forwardData["media_type"] = callType
}
```

**测试：**
- 发送新格式消息，验证服务器正确转发
- 发送旧格式消息，验证服务器兼容处理

**提交：** `git commit -m "feat: 服务器端支持新的消息格式"`

---

#### 步骤 2：创建新的消息路由

**目标：** 在客户端添加新的消息路由逻辑

**文件：** `qim-client/src/composables/useRealtimeMessaging.ts` (新建)

**内容：**

```typescript
export function useRealtimeMessaging() {
  const screenShare = useScreenShareNew()
  const videoCall = useVideoCallNew()
  
  const handleWebRTCOffer = (data: any) => {
    const mediaType = data.media_type || data.share_type || data.call_type
    
    switch (mediaType) {
      case 'screen':
        screenShare.acceptShare(data.signal, data.from_user_id)
        break
      case 'video':
      case 'audio':
        videoCall.acceptCall(data.signal, data.from_user_id)
        break
    }
  }
  
  const handleWebRTCAnswer = (data: any) => {
    const mediaType = data.media_type || data.share_type || data.call_type
    
    switch (mediaType) {
      case 'screen':
        screenShare.handleAnswer(data.signal)
        break
      case 'video':
      case 'audio':
        videoCall.handleAnswer(data.signal)
        break
    }
  }
  
  const handleWebRTCIceCandidate = (data: any) => {
    const mediaType = data.media_type || data.share_type || data.call_type
    
    switch (mediaType) {
      case 'screen':
        screenShare.handleIceCandidate(data.candidate)
        break
      case 'video':
      case 'audio':
        videoCall.handleIceCandidate(data.candidate)
        break
    }
  }
  
  return {
    handleWebRTCOffer,
    handleWebRTCAnswer,
    handleWebRTCIceCandidate
  }
}
```

**测试：**
- 验证消息路由正确
- 验证新旧格式兼容

**提交：** `git commit -m "feat: 添加新的消息路由逻辑"`

---

#### 步骤 3：迁移屏幕共享组件

**目标：** 使用新的 composable 重构屏幕共享组件

**文件：** `qim-client/src/components/shared/ScreenShare.vue`

**策略：**
1. 创建新组件 `ScreenShareNew.vue`
2. 使用 `useScreenShareNew`
3. 简化状态管理
4. 完整测试后替换旧组件

**测试清单：**
- [ ] 发送方：开始共享
- [ ] 接收方：收到共享请求
- [ ] 接收方：接受共享
- [ ] 接收方：拒绝共享
- [ ] 双方：视频流正常显示
- [ ] 发送方：暂停/恢复共享
- [ ] 发送方：停止共享
- [ ] 接收方：关闭窗口
- [ ] 最小化/展开功能
- [ ] 拖拽功能

**提交：** `git commit -m "refactor: 迁移屏幕共享组件到新架构"`

---

#### 步骤 4：迁移视频通话组件

**目标：** 使用新的 composable 重构视频通话组件

**文件：** `qim-client/src/components/realtime/VideoCall.vue`

**策略：**
1. 创建新组件 `VideoCallNew.vue`
2. 使用 `useVideoCallNew`
3. 简化状态管理
4. 完整测试后替换旧组件

**测试清单：**
- [ ] 发起方：开始通话
- [ ] 接收方：收到通话请求
- [ ] 接收方：接受通话
- [ ] 接收方：拒绝通话
- [ ] 双方：视频流正常显示
- [ ] 双方：音频正常
- [ ] 切换摄像头
- [ ] 切换麦克风
- [ ] 结束通话

**提交：** `git commit -m "refactor: 迁移视频通话组件到新架构"`

---

#### 步骤 5：清理旧代码

**目标：** 移除不再使用的旧代码

**文件：**
- `qim-client/src/composables/useScreenShare.ts` (保留，标记为 deprecated)
- `qim-client/src/composables/useVideoCall.ts` (保留，标记为 deprecated)
- `qim-client/src/utils/webrtc.js` (保留，但标记部分为 deprecated)

**策略：**
1. 添加 `@deprecated` 注释
2. 保留一段时间供参考
3. 后续版本完全移除

**提交：** `git commit -m "chore: 标记旧代码为 deprecated"`

---

## 时间估算

| 步骤 | 预计时间 | 优先级 |
|------|---------|--------|
| 步骤 1：服务器端更新 | 1-2 小时 | 高 |
| 步骤 2：消息路由 | 1-2 小时 | 高 |
| 步骤 3：屏幕共享迁移 | 4-6 小时 | 高 |
| 步骤 4：视频通话迁移 | 4-6 小时 | 中 |
| 步骤 5：清理旧代码 | 1-2 小时 | 低 |
| **总计** | **11-18 小时** | |

## 风险和缓解

### 风险 1：消息格式不兼容

**影响：** 新旧客户端无法互通

**缓解：**
- 服务器端同时支持新旧格式
- 客户端同时识别新旧格式
- 充分测试兼容性

### 风险 2：功能缺失

**影响：** 迁移后功能不完整

**缓解：**
- 完整的测试清单
- 逐个功能验证
- 保留旧代码作为参考

### 风险 3：性能问题

**影响：** 新架构性能不如旧代码

**缓解：**
- 性能测试
- 优化关键路径
- 必要时回退

## 回退计划

如果迁移过程中遇到严重问题：

1. **立即回退：**
   ```bash
   git checkout main
   git branch -D refactor/realtime-communication
   ```

2. **部分回退：**
   - 保留服务器端更新（向后兼容）
   - 回退客户端组件到旧版本
   - 继续使用旧的 composables

3. **修复后继续：**
   - 修复问题
   - 重新测试
   - 继续迁移

## 成功标准

迁移成功的标准：

1. ✅ 所有功能正常工作
2. ✅ 性能无明显下降
3. ✅ 代码更简洁易维护
4. ✅ 测试覆盖完整
5. ✅ 文档完善

## 下一步行动

1. **立即执行：** 步骤 1 - 更新服务器端消息处理
2. **后续执行：** 步骤 2-5，根据优先级逐步推进
3. **持续改进：** 收集反馈，优化实现

---

**注意：** 迁移过程中，新旧代码可以共存。建议先在测试环境充分验证后再部署到生产环境。
