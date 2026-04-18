<template>
  <div class="chat-window">
    <div class="chat-header">
      <div class="header-info">
        <img :src="(conversation?.avatar && conversation.avatar.startsWith('http')) ? conversation.avatar : (conversation?.avatar ? serverUrl + conversation.avatar : generateAvatar(conversation?.name || '用户'))" :alt="conversation?.name || '未知'" class="header-avatar" />
        <div class="header-text">
          <div class="header-name" @dblclick="editGroupInfo">{{ conversation?.name || '未知会话' }}</div>
          <div class="header-status">
            {{ conversation?.type === 'group' ? '群聊' : '在线' }}
            <span v-if="conversation?.type === 'group' && conversation?.members" class="member-count">
              ({{ conversation.members.length }}人)
            </span>
            <span v-if="conversation?.type === 'single' && conversation?.ip" class="ip-info">
              {{ conversation.ip }}
            </span>
          </div>
        </div>
      </div>
      <div class="header-actions">
        <span v-if="conversation?.type === 'group'" class="header-icon" title="邀请成员" @click="handleInviteMembers"><i class="fas fa-user-plus"></i></span>
        <span class="header-icon" @click="toggleHeaderMenu">
          <i class="fas fa-ellipsis-v"></i>
          <!-- 头部下拉菜单 -->
          <div v-if="showHeaderMenu" class="header-menu" @click.stop>
            <div v-if="conversation?.type === 'group' || conversation?.type === 'discussion'" class="menu-item" @click="editGroupInfo">
              <i class="fas fa-edit"></i> 修改群名称
            </div>
            <div v-if="conversation?.type === 'group'" class="menu-item" @click="editGroupAnnouncement">
              <i class="fas fa-bullhorn"></i> 编辑群公告
            </div>
          </div>
        </span>
      </div>
    </div>

    <div class="chat-main">
      <div ref="messageListRef" class="message-list">
        <!-- 搜索状态 -->
        <div v-if="isSearching" class="search-status">
          <div class="search-loading">搜索中...</div>
        </div>
        
        <!-- 搜索结果 -->
        <div v-else-if="searchResults.length > 0" class="search-results">
          <div class="search-results-header">
            找到 {{ searchResults.length }} 条相关消息
            <button class="clear-search-btn" @click="clearSearch">清除搜索</button>
          </div>
          <MessageItem
            v-for="message in searchResults"
            :key="message.id"
            :message="message"
            :is-self="message.isSelf"
            :is-recalled="message.isRecalled"
            :conversation-type="conversation?.type || 'single'"
            :read-users-map="readUsersMap"
            :server-url="serverUrl"
            @contextmenu="showMessageContextMenu"
            @show-user-profile="showUserProfile"
            @preview-image="previewImage"
            @download-file="downloadFile"
            @save-as="saveFileAs"
            @view-shared-content="viewSharedContent"
            @retry-send-message="retrySendMessage"
            @show-read-users="showReadUsers"
          />
        </div>
        
        <!-- 无搜索结果 -->
        <div v-else-if="searchQuery && !isSearching" class="search-status">
          <div class="search-no-results">没有找到相关消息</div>
        </div>
        
        <!-- 正常消息列表 -->
        <div v-else>
          <!-- 没有更多消息提示 -->
          <div v-if="!hasMoreMessages" class="no-more-messages">
            <span>没有更多消息了</span>
          </div>
          
          <div v-for="(message, index) in messages" :key="message.id">
            <!-- 显示时间分隔线 -->
            <div v-if="shouldShowTimeDivider(index, message)" class="time-divider">
              <span class="time-divider-text">{{ formatTime(message.timestamp) }}</span>
            </div>
            
            <MessageItem
              :message="message"
              :is-self="message.isSelf"
              :is-recalled="message.isRecalled"
              :conversation-type="conversation?.type || 'single'"
              :read-users-map="readUsersMap"
              :server-url="serverUrl"
              @contextmenu="showMessageContextMenu"
              @show-user-profile="showUserProfile"
              @scroll-to-quoted-message="scrollToQuotedMessage"
              @preview-image="previewImage"
              @download-file="downloadFile"
              @save-as="saveFileAs"
              @view-shared-content="viewSharedContent"
              @open-mini-app="openMiniApp"
              @open-news-link="openNewsLink"
              @retry-send-message="retrySendMessage"
              @show-read-users="showReadUsers"
            />
          </div>
        </div>
      </div>

      <!-- 群成员侧边栏 -->
      <div v-if="conversation?.type === 'group' && conversation?.members" class="members-sidebar" :class="{ 'collapsed': !isMembersSidebarExpanded }">
        <div class="sidebar-header-container">
          <div v-if="isMembersSidebarExpanded" class="members-header">
            <div class="header-content">
              <button class="toggle-sidebar-btn" @click="toggleMembersSidebar">
                <i class="fas fa-chevron-left"></i>
              </button>
              <h3>群成员 ({{ conversation.members.length }})</h3>
            </div>
            <div class="header-actions">
              <button class="search-toggle-btn" @click="toggleMemberSearch">
                <i class="fas fa-search"></i>
              </button>
            </div>
          </div>
          <button v-else class="collapsed-toggle-btn" @click="toggleMembersSidebar">
            <i class="fas fa-user"></i>
          </button>
        </div>
        <div v-if="showMemberSearch && isMembersSidebarExpanded" class="members-search">
          <input
            v-model="memberSearchQuery"
            type="text"
            placeholder="搜索群成员..."
            class="member-search-input"
            @focus="showMemberSearch = true"
          />
        </div>
        <div v-if="isMembersSidebarExpanded" class="members-content">
          <div v-for="member in filteredMembers" :key="member.id" class="member-item" @contextmenu.prevent="showMemberContextMenu($event, member)" @dblclick="startPrivateChat(member)">
            <img :src="member.avatar" :alt="member.name || '未知用户'" class="member-avatar" />
            <div class="member-info">
                <span class="member-name">{{ member.name || '未知用户' }}</span><span v-if="member.role === 'owner'" class="member-role owner" title="群主"><i class="fas fa-crown"></i></span><span v-else-if="member.role === 'admin'" class="member-role admin" title="管理员"><i class="fas fa-user-shield"></i></span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="chat-input-area">
      <div class="input-toolbar">
        <button class="toolbar-btn" @click="startVoiceCall"><i class="fas fa-phone-alt"></i></button>
        <button class="toolbar-btn" @click="startVideoCall"><i class="fas fa-video"></i></button>
        <button class="toolbar-btn" @click="startScreenShare"><i class="fas fa-desktop"></i></button>
        <button class="toolbar-btn" @click="toggleEmojiPanel"><i class="fas fa-smile"></i></button>
        <button class="toolbar-btn" @click="selectFile"><i class="fas fa-paperclip"></i></button>
        <button class="toolbar-btn" @click="selectImage"><i class="fas fa-image"></i></button>
        <button v-if="isElectron" class="toolbar-btn" @click="takeScreenshot"><i class="fas fa-scissors"></i></button>
        <button class="toolbar-btn" @click="openMessageManager"><i class="fas fa-history"></i></button>
        <button class="toolbar-btn" @click="openMiniAppList"><i class="fas fa-th-large"></i></button>

      </div>
      
      <!-- 表情面板 -->
      <div v-if="showEmojiPanel" class="emoji-panel-container">
        <div class="emoji-panel-backdrop" @click="closeEmojiPanel"></div>
        <div class="emoji-panel">
        <div class="emoji-category">
          <div class="emoji-category-title">常用表情</div>
          <div class="emoji-grid">
            <div v-for="emoji in commonEmojis" :key="emoji" class="emoji-item" @click="insertEmoji(emoji)">
              {{ emoji }}
            </div>
          </div>
        </div>
        <div class="emoji-category">
          <div class="emoji-category-title">表情符号</div>
          <div class="emoji-grid">
            <div v-for="emoji in faceEmojis" :key="emoji" class="emoji-item" @click="insertEmoji(emoji)">
              {{ emoji }}
            </div>
          </div>
        </div>
        <div class="emoji-category">
          <div class="emoji-category-title">动物与自然</div>
          <div class="emoji-grid">
            <div v-for="emoji in animalEmojis" :key="emoji" class="emoji-item" @click="insertEmoji(emoji)">
              {{ emoji }}
            </div>
          </div>
        </div>
        </div>
      </div>
      
      <!-- @成员面板 -->
      <div v-if="showAtMembersPanel && conversation?.type === 'group'" class="at-members-panel-container">
        <div class="at-members-panel-backdrop" @click="closeAtMembersPanel"></div>
        <div class="at-members-panel">
          <div class="at-members-header">
            <h4>选择成员</h4>
          </div>
          <div class="at-members-search">
            <input
              v-model="atMembersSearchQuery"
              type="text"
              placeholder="搜索成员..."
              class="at-members-search-input"
            />
          </div>
          <div class="at-members-list">
            <div v-for="member in filteredAtMembers" :key="member.id" class="at-member-item" @click="selectAtMember(member)">
              <img :src="member.avatar" :alt="member.name || '未知用户'" class="at-member-avatar" />
              <span class="at-member-name">{{ member.name || '未知用户' }}</span>
            </div>
            <div v-if="filteredAtMembers.length === 0" class="empty-at-members">
              <p>没有找到匹配的成员</p>
            </div>
          </div>
        </div>
      </div>
      
      <!-- 小程序列表面板 -->
      <MiniAppManager 
        v-model:showMiniAppList="showMiniAppList"
        @send-mini-app-message="handleSendMiniAppMessage"
      />
      
      <input type="file" ref="fileInput" style="display: none" @change="handleFileSelect" multiple />

      <!-- 搜索框 -->
      <div v-if="showSearch" class="search-container">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="搜索历史消息..."
          class="search-input"
          @keyup.enter="performSearch"
        />
        <button class="search-btn" @click="performSearch">搜索</button>
        <button class="close-search-btn" @click="showSearch = false">×</button>
      </div>
      <!-- 引用消息 -->
      <div v-if="quotedMessage" class="quoted-message">
        <div class="quoted-message-header">
          <span class="quoted-message-sender">{{ quotedMessage.sender?.name || quotedMessage.name || '未知用户' }}</span>
          <button class="quoted-message-remove" @click="quotedMessage = null">×</button>
        </div>
        <div class="quoted-message-content">
          <template v-if="quotedMessage.type === 'text'">
            {{ quotedMessage.content || '无内容' }}
          </template>
          <template v-else-if="quotedMessage.type === 'image'">
            [图片]
          </template>
          <template v-else-if="quotedMessage.type === 'file'">
            [文件]
          </template>
          <template v-else-if="quotedMessage.type === 'mini-app' || quotedMessage.type === 'miniApp'">
            [小程序]
          </template>
          <template v-else-if="quotedMessage.type === 'share'">
            [分享]
          </template>
          <template v-else>
            {{ quotedMessage.content || '无内容' }}
          </template>
        </div>
      </div>
      <textarea
        ref="messageInputRef"
        v-model="inputMessage"
        class="message-input"
        placeholder="输入消息..."
        rows="4"
        @keydown.enter="handleKeydown"
        @input="handleInputAndResize"
        @paste="handlePaste"
      />
      <div class="input-actions">
        <span class="input-tip">按 Enter 发送，Shift+Enter 换行</span>
        <button class="send-btn" :disabled="!inputMessage.trim()" @click="handleSend">
          发送
        </button>
      </div>
    </div>
  </div>
  
  <!-- 成员上下文菜单 -->
  <div v-if="showMemberContextMenuFlag" class="context-menu" :style="{ left: memberContextMenuPosition.x + 'px', top: memberContextMenuPosition.y + 'px' }">
    <div v-if="canRemoveMember" class="context-menu-item" @click="removeMemberFromGroup">
      <span class="context-menu-icon"><i class="fas fa-trash"></i></span>
      <span>移除群聊</span>
    </div>
    <div class="context-menu-item" @click="viewMemberInfo">
      <span class="context-menu-icon"><i class="fas fa-user"></i></span>
      <span>查看资料</span>
    </div>
    <div v-if="canSetAdmin" class="context-menu-item" @click="setAsAdmin">
      <span class="context-menu-icon"><i class="fas fa-star"></i></span>
      <span>{{ isSelectedMemberAdmin ? '取消管理员' : '设为管理员' }}</span>
    </div>
    <div v-if="canTransferOwner" class="context-menu-item" @click="transferOwner">
      <span class="context-menu-icon"><i class="fas fa-crown"></i></span>
      <span>转让群主</span>
    </div>
    <div class="context-menu-item" @click="sendPrivateMessage">
      <span class="context-menu-icon"><i class="fas fa-comment"></i></span>
      <span>发起私聊</span>
    </div>
  </div>
  
  <!-- 用户资料弹窗 -->
  <UserProfile 
    :visible="showUserProfileFlag" 
    :user="selectedUser" 
    @close="closeUserProfile"
    @send-private-message="handleSendPrivateMessage"
  />

  <!-- 已读用户列表弹窗 -->
  <div v-if="showReadUsersModal" class="read-users-modal" @click="showReadUsersModal = false">
    <div class="read-users-content" @click.stop>
      <div class="read-users-header">
        <h3>已读用户 ({{ currentReadUsers.read_users?.length || 0 }}/{{ Math.max(0, (currentReadUsers.total_members || 0) - 1) }})</h3>
        <button class="close-btn" @click="showReadUsersModal = false">×</button>
      </div>
      <div class="read-users-body">
        <div v-if="currentReadUsers.read_users?.length === 0" class="empty-read">
          暂无已读用户
        </div>
        <div v-else class="read-users-list">
          <div v-for="user in currentReadUsers.read_users" :key="user.id" class="read-user-item">
            <img :src="(user.avatar && user.avatar.startsWith('http')) ? user.avatar : (user.avatar ? serverUrl + user.avatar : 'https://api.dicebear.com/7.x/avataaars/svg?seed=' + user.id)" :alt="user.name" class="read-user-avatar" />
            <div class="read-user-info">
              <span class="read-user-name">{{ user.name || user.username }}</span>
            </div>
            <i class="fas fa-check read-icon"></i>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- 消息上下文菜单 -->
  <div v-if="showMessageContextMenuFlag" class="context-menu" :style="{ left: messageContextMenuPosition.x + 'px', top: messageContextMenuPosition.y + 'px' }">
    <!-- 图片消息选项 -->
    <div v-if="selectedMessage && selectedMessage.type === 'image'" class="context-menu-item" @click="previewImage(selectedMessage.content)">
      <span class="context-menu-icon"><i class="fas fa-eye"></i></span>
      <span>预览</span>
    </div>
    <div v-if="selectedMessage && selectedMessage.type === 'image'" class="context-menu-item" @click="saveFileAs(selectedMessage.content)">
      <span class="context-menu-icon"><i class="fas fa-save"></i></span>
      <span>保存图片</span>
    </div>
    <!-- 文件消息选项 -->
    <div v-if="selectedMessage && selectedMessage.type === 'file'" class="context-menu-item" @click="downloadFile(selectedMessage.content)">
      <span class="context-menu-icon"><i class="fas fa-download"></i></span>
      <span>下载</span>
    </div>
    <div v-if="selectedMessage && selectedMessage.type === 'file'" class="context-menu-item" @click="saveFileAs(selectedMessage.content)">
      <span class="context-menu-icon"><i class="fas fa-save"></i></span>
      <span>另存为</span>
    </div>
    <!-- 分隔线 -->
    <div v-if="selectedMessage && (selectedMessage.type === 'image' || selectedMessage.type === 'file')" class="context-menu-divider"></div>
    <!-- 通用选项 -->
    <div v-if="selectedMessage && (selectedMessage.type === 'text' || selectedMessage.type === 'image')" class="context-menu-item" @click="copyMessage">
      <span class="context-menu-icon"><i class="fas fa-copy"></i></span>
      <span>复制</span>
    </div>
    <div class="context-menu-item" @click="forwardMessage">
      <span class="context-menu-icon"><i class="fas fa-share-alt"></i></span>
      <span>转发</span>
    </div>
    <div class="context-menu-item" @click="quoteMessage">
      <span class="context-menu-icon"><i class="fas fa-quote-right"></i></span>
      <span>引用</span>
    </div>
    <div v-if="selectedMessage.type === 'text'" class="context-menu-item" @click="addToNote">
      <span class="context-menu-icon"><i class="fas fa-sticky-note"></i></span>
      <span>添加到便签</span>
    </div>
    <!-- <div class="context-menu-item" @click="deleteMessage">
      <span class="context-menu-icon"><i class="fas fa-trash"></i></span>
      <span>删除</span>
    </div> -->
    <div v-if="selectedMessage.isSelf" class="context-menu-item" @click="recallMessage">
      <span class="context-menu-icon"><i class="fas fa-undo"></i></span>
      <span>撤回</span>
    </div>
    <div v-if="selectedMessage.isSelf && canSendReminder(selectedMessage)" class="context-menu-item" @click="sendMessageReminder">
      <span class="context-menu-icon"><i class="fas fa-bell"></i></span>
      <span>发送提醒</span>
    </div>
  </div>
  

  
  <!-- 消息管理器 -->
  <MessageManager 
    :visible="showMessageManager" 
    :conversation-id="conversation?.id" 
    @close="closeMessageManager"
    @scroll-to-message="scrollToMessage"
  />
  
  <!-- 确认对话框 -->
  <div v-if="showConfirmDialog" class="confirm-dialog-modal" @click="closeConfirmDialog">
    <div class="confirm-dialog-content" @click.stop>
      <div class="confirm-dialog-header">
        <h3>{{ confirmDialogTitle }}</h3>
        <button class="close-btn" @click="closeConfirmDialog">×</button>
      </div>
      <div class="confirm-dialog-body">
        <p>{{ confirmDialogMessage }}</p>
      </div>
      <div class="confirm-dialog-footer">
        <button class="cancel" @click="closeConfirmDialog">取消</button>
        <button class="confirm" @click="handleConfirmAction">确定</button>
      </div>
    </div>
  </div>
  
  <!-- 截图预览对话框 -->
  <div v-if="showScreenshotPreview" class="screenshot-preview-modal" @click="cancelScreenshot">
    <div class="screenshot-preview-content" @click.stop>
      <div class="screenshot-preview-header">
        <h3>截图预览</h3>
        <button class="close-btn" @click="cancelScreenshot">×</button>
      </div>
      <div class="screenshot-preview-body">
        <div class="screenshot-image-container">
          <img :src="screenshotImageData" class="screenshot-image" alt="截图" />
        </div>
      </div>
      <div class="screenshot-preview-footer">
        <button class="screenshot-btn retake-btn" @click="retakeScreenshot">重新截图</button>
        <button class="screenshot-btn cancel-btn" @click="cancelScreenshot">取消</button>
        <button class="screenshot-btn send-btn" @click="uploadScreenshot">发送</button>
      </div>
    </div>
  </div>
  
  <!-- 通话模态框 -->
  <div v-if="showCallModal" class="call-modal" @click="endCall">
    <div class="call-modal-content" @click.stop>
      <div class="call-modal-header">
        <h3>{{ callType === 'voice' ? '语音通话' : '视频通话' }}</h3>
      </div>
      <div class="call-modal-body">
        <div class="call-info">
          <div class="call-avatar">
            <img :src="props.conversation?.avatar || 'https://api.dicebear.com/7.x/avataaars/svg?seed=user'" :alt="props.conversation?.name || '未知'" />
          </div>
          <div class="call-name">{{ props.conversation?.name || '未知' }}</div>
          <div class="call-status">
            <span v-if="callStatus === 'ringing'" class="status-ringing">正在呼叫...</span>
            <span v-else-if="callStatus === 'answered'" class="status-answered">通话中</span>
            <span v-else-if="callStatus === 'ended'" class="status-ended">通话结束</span>
          </div>
        </div>
        
        <!-- 视频通话区域 -->
        <div v-if="callType === 'video' && callStatus === 'answered'" class="video-container">
          <div class="local-video">
            <div class="video-placeholder">
              <i class="fas fa-user"></i>
              <span>您</span>
            </div>
          </div>
          <div class="remote-video">
            <div class="video-placeholder">
              <i class="fas fa-user"></i>
              <span>{{ props.conversation?.name || '对方' }}</span>
            </div>
          </div>
        </div>
      </div>
      <div class="call-modal-footer">
        <button v-if="callStatus === 'ringing'" class="call-btn reject-btn" @click="rejectCall">
          <i class="fas fa-phone-slash"></i>
          <span>拒绝</span>
        </button>
        <button v-if="callStatus === 'ringing'" class="call-btn answer-btn" @click="answerCall">
          <i class="fas fa-phone"></i>
          <span>接听</span>
        </button>
        <button v-else class="call-btn end-btn" @click="endCall">
          <i class="fas fa-phone-slash"></i>
          <span>结束通话</span>
        </button>
        <button v-if="isScreenSharing" class="call-btn screen-share-btn" @click="stopScreenShare">
          <i class="fas fa-stop"></i>
          <span>停止屏幕共享</span>
        </button>
      </div>
    </div>
  </div>
  
  <!-- 屏幕共享模态框 -->
  <div v-if="isScreenSharing" class="call-modal" @click="stopScreenShare">
    <div class="call-modal-content" @click.stop>
      <div class="call-modal-header">
        <h3>屏幕共享</h3>
      </div>
      <div class="call-modal-body">
        <div class="call-info">
          <div class="call-avatar">
            <img :src="props.conversation?.avatar || 'https://api.dicebear.com/7.x/avataaars/svg?seed=user'" :alt="props.conversation?.name || '未知'" />
          </div>
          <div class="call-name">{{ props.conversation?.name || '未知' }}</div>
          <div class="call-status">
            <span class="status-answered">屏幕共享中</span>
          </div>
        </div>
        
        <!-- 屏幕共享区域 -->
        <div class="video-container">
          <div class="remote-video">
            <div class="video-placeholder">
              <i class="fas fa-desktop"></i>
              <span>屏幕共享中</span>
            </div>
          </div>
        </div>
      </div>
      <div class="call-modal-footer">
        <button class="call-btn end-btn" @click="stopScreenShare">
          <i class="fas fa-stop"></i>
          <span>停止屏幕共享</span>
        </button>
      </div>
    </div>
  </div>

  <!-- 图片预览弹窗 -->
  <div v-if="showImagePreview" class="image-preview-modal" @click="closeImagePreview">
    <div class="image-preview-content" @click.stop>
      <div class="image-preview-header">
        <button class="close-btn" @click="closeImagePreview">
          <i class="fas fa-times"></i>
        </button>
      </div>
      <div class="image-preview-body">
        <img :src="previewImageUrl" alt="预览图片" />
      </div>
    </div>
  </div>
  
  <!-- 分享内容预览弹窗 -->
  <div v-if="showSharePreview" class="share-preview-modal" @click="closeSharePreview">
    <div class="share-preview-content" @click.stop>
      <div class="share-preview-header">
        <h3>{{ sharePreviewData.type === 'file' ? '文件详情' : (sharePreviewData.type === 'note' ? '笔记详情' : '便签详情') }}</h3>
        <button class="close-btn" @click="closeSharePreview">
          <i class="fas fa-times"></i>
        </button>
      </div>
      <div class="share-preview-body">
        <!-- 文件类型 -->
        <div v-if="sharePreviewData.type === 'file'" class="share-file-content">
          <div class="share-file-icon">
            <i :class="getFileIcon(sharePreviewData.url || sharePreviewData.path)"></i>
          </div>
          <div class="share-file-info">
            <div class="share-preview-title">{{ sharePreviewData.name }}</div>
            <div class="share-file-size" v-if="sharePreviewData.size">{{ formatFileSize(sharePreviewData.size) }}</div>
          </div>
        </div>
        <!-- 笔记和便签类型 -->
        <div v-else>
          <div class="share-preview-title">{{ sharePreviewData.name }}</div>
          <div class="share-preview-content-text" v-if="sharePreviewData.content">{{ sharePreviewData.content }}</div>
        </div>
        <div class="share-preview-meta">
          <span class="share-preview-type">{{ sharePreviewData.type === 'file' ? '文件' : (sharePreviewData.type === 'note' ? '笔记' : '便签') }}</span>
          <span class="share-preview-time" v-if="sharePreviewData.created_at">{{ formatTime(new Date(sharePreviewData.created_at).getTime()) }}</span>
        </div>
      </div>
      <!-- 文件操作按钮 -->
      <div v-if="sharePreviewData.type === 'file'" class="share-preview-footer">
        <button class="share-file-action-btn" @click="downloadFile(sharePreviewData.url || sharePreviewData.path, sharePreviewData.name)">下载</button>
        <button class="share-file-action-btn" @click="saveFileAs(sharePreviewData.url || sharePreviewData.path, sharePreviewData.name)">另存为</button>
      </div>
    </div>
  </div>
  
  <!-- 编辑群信息模态框 -->
  <div v-if="showEditGroupInfoModal" class="modal-overlay" @click="showEditGroupInfoModal = false">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <h3>修改群名称</h3>
        <button class="close-btn" @click="showEditGroupInfoModal = false">×</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>群名称</label>
          <input type="text" v-model="editGroupName" class="form-input" placeholder="请输入新的群名称" />
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="showEditGroupInfoModal = false">取消</button>
        <button class="btn btn-primary" @click="saveGroupInfo">保存</button>
      </div>
    </div>
  </div>
  
  <!-- 编辑群公告模态框 -->
  <div v-if="showEditAnnouncementModal" class="modal-overlay" @click="showEditAnnouncementModal = false">
    <div class="modal-content" @click.stop>
      <div class="modal-header">
        <h3>编辑群公告</h3>
        <button class="close-btn" @click="showEditAnnouncementModal = false">×</button>
      </div>
      <div class="modal-body">
        <div class="form-group">
          <label>群公告内容</label>
          <textarea v-model="editAnnouncementContent" class="form-textarea" placeholder="输入群公告内容..." rows="5"></textarea>
          <p class="form-tip">群公告将对所有群成员可见</p>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn btn-secondary" @click="showEditAnnouncementModal = false">取消</button>
        <button class="btn btn-primary" @click="saveAnnouncement">保存</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed, onMounted, onUnmounted } from 'vue'
import type { Conversation, Message } from '../types'
import { ElMessage } from 'element-plus'
import UserProfile from '../common/UserProfile.vue'
import MiniAppManager from './apps/MiniAppManager.vue'
import MessageItem from './message/MessageItem.vue'
import MessageManager from './MessageManager.vue'
import { openMiniApp } from '../utils/miniAppUtils'
import { API_BASE_URL } from '../config'
import { generateAvatar } from '../utils/avatar'
import '../styles/mini-app.css'

// 服务器地址
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 获取token
const getToken = () => {
  return localStorage.getItem('token')
}

// 格式化日期为YYYY-MM-DD格式（本地时间）
const formatDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

// 通用请求方法
const request = async (url: string, options?: RequestInit) => {
  const token = getToken()
  const headers = {
    'Content-Type': 'application/json',
    ...(token ? { 'Authorization': `Bearer ${token}` } : {})
  }
  
  const fullUrl = `${serverUrl.value}${url}`
  const requestHeaders = {
    ...headers,
    ...options?.headers
  }
  console.log('发送请求:', fullUrl, options)
  console.log('请求头:', requestHeaders)
  console.log('Token:', token)
  
  try {
    const response = await fetch(fullUrl, {
      ...options,
      headers: requestHeaders
    })
    
    console.log('响应状态:', response.status, response.statusText)
    
    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      console.error('请求失败:', errorData)
      throw new Error(errorData.message || '请求失败')
    }
    
    const data = await response.json()
    console.log('响应数据:', data)
    return data
  } catch (error) {
    console.error('网络错误:', error)
    throw error
  }
}

interface Props {
  conversation: Conversation
  messages: Message[]
  getReadUsers?: (messageId: string) => Promise<{ read_users: any[], total_members: number }>
  currentUser: any
  hasMoreMessages: boolean
  updateConversation?: (conversation: Conversation) => void
}

const props = defineProps<Props>()
const emit = defineEmits<{
  send: [content: string]
  recall: [messageId: number]
  inviteMembers: [conversationId: string]
  'read-receipt': [conversationId: string]
  'switch-app': [app: string]
  'loadMore': [messages: any[]]
  'switchConversation': [conversationId: string]
  'retry-send': [message: any]
}>()
const inputMessage = ref('')
const messageListRef = ref<HTMLDivElement>()
const messageInputRef = ref<HTMLTextAreaElement>()
const showSearch = ref(false)
const searchQuery = ref('')
const searchResults = ref<Message[]>([])
const isSearching = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

// 成员上下文菜单
const showMemberContextMenuFlag = ref(false)
const memberContextMenuPosition = ref({ x: 0, y: 0 })
const selectedMember = ref(null)

// 用户资料弹窗
const showUserProfileFlag = ref(false)
const selectedUser = ref({})

// 消息上下文菜单
const showMessageContextMenuFlag = ref(false)
const messageContextMenuPosition = ref({ x: 0, y: 0 })
const selectedMessage = ref(null)
const quotedMessage = ref(null)

// 头部下拉菜单状态
const showHeaderMenu = ref(false)

// 编辑群信息状态
const showEditGroupInfoModal = ref(false)
const editGroupName = ref('')

// 编辑群公告状态
const showEditAnnouncementModal = ref(false)
const editAnnouncementContent = ref('')

// 已读用户列表
const readUsersMap = ref<Record<string, { read_users: any[], total_members: number }>>({})
const showReadUsersModal = ref(false)
const currentReadUsers = ref<{ read_users: any[], total_members: number }>({ read_users: [], total_members: 0 })

// 获取消息已读用户列表
const fetchReadUsers = async (messageId: string) => {
  if (!isMounted.value) return { read_users: [], total_members: 0 }
  
  // 强制重新获取已读用户列表，不使用缓存
  if (props.getReadUsers) {
    try {
      const data = await props.getReadUsers(messageId)
      console.log('获取已读用户列表成功:', messageId, data)
      if (isMounted.value) {
        // 对已读用户列表进行去重处理，确保一个用户只出现一次
        const uniqueReadUsers = []
        const seenUserIds = new Set()
        
        if (data.read_users) {
          for (const user of data.read_users) {
            if (user.id && !seenUserIds.has(user.id)) {
              seenUserIds.add(user.id)
              uniqueReadUsers.push(user)
            }
          }
        }
        
        // 更新去重后的已读用户列表
        readUsersMap.value[messageId] = {
          ...data,
          read_users: uniqueReadUsers
        }
      }
      return data
    } catch (error) {
      console.error('获取已读用户列表失败:', error)
      return { read_users: [], total_members: 0 }
    }
  }
  return { read_users: [], total_members: 0 }
}

// 显示已读用户列表弹窗
const showReadUsers = async (message: Message) => {
  if (!message.isSelf || !isMounted.value) return
  const data = await fetchReadUsers(message.id)
  if (isMounted.value) {
    currentReadUsers.value = data
    showReadUsersModal.value = true
  }
}

// 消息管理器
const showMessageManager = ref(false)

// 表情面板相关
const showEmojiPanel = ref(false)
const commonEmojis = ['😊', '😂', '❤️', '👍', '🎉', '🔥', '🤔', '😢', '😡', '👏']
const faceEmojis = ['😀', '😃', '😄', '😁', '😆', '😅', '😂', '🤣', '😊', '😇', '🙂', '🙃', '😉', '😌', '😍', '🥰', '😘', '😗', '😙', '😚', '😋', '😛', '😝', '😜', '🤪', '🤨', '🧐', '🤓', '😎', '🤩', '🥳', '😏', '😒', '😞', '😔', '😟', '😕', '🙁', '☹️', '😣', '😖', '😫', '😩', '🥺', '😢', '😭', '😤', '😠', '😡', '🤬', '🤯', '😳', '🥵', '🥶', '😱', '😨', '😰', '😥', '😓', '🤗', '🤔', '🤭', '🤫', '🤥', '😶', '😐', '😑', '😬', '🙄', '😯', '😦', '😧', '😮', '😲', '🥱', '😴', '🤤', '😪', '😵', '🤐', '🥴', '🤢', '🤮', '🤧', '🥵', '🤒', '🤕', '🤠']
const animalEmojis = ['🐶', '🐱', '🐭', '🐹', '🐰', '🦊', '🐻', '🐼', '🐨', '🐯', '🦁', '🐮', '🐷', '🐸', '🐵', '🐔', '🐧', '🐦', '🐤', '🐣', '🐥', '🦆', '🦅', '🦉', '🦇', '🐺', '🐗', '🐴', '🦄', '🐝', '🐛', '🦋', '🐌', '🐞', '🐜', '🕷️', '🦂', '🐢', '🐍', '🦎', '🦖', '🦕', '🐙', '🦑', '🦐', '🦞', '🦀', '🐡', '🐠', '🐟', '🐬', '🐳', '🐋', '🐊', '🐅', '🐆', '🐈', '🐩']

// @成员功能相关
const showAtMembersPanel = ref(false)
const atMembersSearchQuery = ref('')
const filteredAtMembers = computed(() => {
  if (!props.conversation) {
    return []
  }
  if (!atMembersSearchQuery.value) {
    return props.conversation.members || []
  }
  const query = atMembersSearchQuery.value.toLowerCase()
  return (props.conversation.members || []).filter(member => 
    member.name.toLowerCase().includes(query)
  )
})
const inputCursorPosition = ref(0)

// 打开消息管理器
const openMessageManager = () => {
  showMessageManager.value = true
}

// 关闭消息管理器
const closeMessageManager = () => {
  showMessageManager.value = false
}

// 打开资讯链接
const openNewsLink = (url: string) => {
  if (!url) return
  console.log('打开资讯链接:', url)
  // 这里可以实现打开链接的逻辑
  window.open(url, '_blank')
}

// 打开小程序列表
const openMiniAppList = () => {
  showMiniAppList.value = true
}

// 关闭小程序列表
const closeMiniAppList = () => {
  showMiniAppList.value = false
}



// 处理发送小程序消息
const handleSendMiniAppMessage = (miniApp: any) => {
  console.log('发送小程序消息:', miniApp)
  // 这里可以实现发送小程序消息的逻辑
  emit('send', JSON.stringify({ type: 'miniApp', data: miniApp }))
}

// 打开小程序 - 使用MiniAppHandler组件中的函数
// 函数已移至MiniAppHandler.vue组件中

// 确认对话框
const showConfirmDialog = ref(false)
const confirmDialogTitle = ref('确认操作')
const confirmDialogMessage = ref('')
const confirmDialogCallback = ref<(() => void) | null>(null)

// 打开确认对话框
const openConfirmDialog = (title: string, message: string, callback: () => void) => {
  confirmDialogTitle.value = title
  confirmDialogMessage.value = message
  confirmDialogCallback.value = callback
  showConfirmDialog.value = true
}

// 关闭确认对话框
const closeConfirmDialog = () => {
  showConfirmDialog.value = false
  confirmDialogCallback.value = null
}

// 处理确认操作
const handleConfirmAction = () => {
  if (confirmDialogCallback.value) {
    confirmDialogCallback.value()
  }
  closeConfirmDialog()
}

// 显示消息提示
const showMessage = (options: { message: string, type?: 'success' | 'warning' | 'error' | 'info', duration?: number }) => {
  const { message, type = 'info', duration = 3000 } = options
  console.log('显示消息:', message, type)
  
  // 创建消息容器
  const messageElement = document.createElement('div')
  
  // 根据类型设置样式
  const typeStyles = {
    success: {
      background: '#f0f9eb',
      color: '#67c23a',
      border: '1px solid #e1f3d8'
    },
    warning: {
      background: '#fdf6ec',
      color: '#e6a23c',
      border: '1px solid #faecd8'
    },
    error: {
      background: '#fef0f0',
      color: '#f56c6c',
      border: '1px solid #fbc4c4'
    },
    info: {
      background: '#f4f4f5',
      color: '#909399',
      border: '1px solid #ebeef5'
    }
  }
  
  const style = typeStyles[type]
  
  // 设置样式
  messageElement.style.position = 'fixed'
  messageElement.style.top = '20px'
  messageElement.style.left = '50%'
  messageElement.style.transform = 'translateX(-50%)'
  messageElement.style.background = style.background
  messageElement.style.color = style.color
  messageElement.style.border = style.border
  messageElement.style.borderRadius = '4px'
  messageElement.style.padding = '12px 20px'
  messageElement.style.boxShadow = '0 2px 12px 0 rgba(0, 0, 0, 0.1)'
  messageElement.style.fontSize = '14px'
  messageElement.style.zIndex = '9999'
  messageElement.style.animation = 'messageFadeIn 0.3s ease'
  messageElement.style.pointerEvents = 'none'
  messageElement.style.minWidth = '300px'
  messageElement.style.maxWidth = '500px'
  messageElement.style.textAlign = 'center'
  
  // 添加图标
  const icon = document.createElement('span')
  icon.style.marginRight = '8px'
  
  switch (type) {
    case 'success':
      icon.innerHTML = '✓'
      icon.style.fontWeight = 'bold'
      break
    case 'warning':
      icon.innerHTML = '⚠️'
      break
    case 'error':
      icon.innerHTML = '✗'
      icon.style.fontWeight = 'bold'
      break
    case 'info':
      icon.innerHTML = 'ℹ️'
      break
  }
  
  messageElement.appendChild(icon)
  
  // 添加消息文本
  const text = document.createElement('span')
  text.textContent = message
  messageElement.appendChild(text)
  
  // 添加到DOM
  document.body.appendChild(messageElement)
  console.log('消息已添加到DOM', messageElement)
  
  // 添加动画样式
  const animationStyle = document.createElement('style')
  animationStyle.textContent = `
    @keyframes messageFadeIn {
      from {
        opacity: 0;
        transform: translateX(-50%) translateY(-10px);
      }
      to {
        opacity: 1;
        transform: translateX(-50%) translateY(0);
      }
    }
  `
  document.head.appendChild(animationStyle)
  
  // 自动移除
  setTimeout(() => {
    messageElement.style.animation = 'messageFadeOut 0.3s ease'
    
    // 添加淡出动画
    const fadeOutStyle = document.createElement('style')
    fadeOutStyle.textContent = `
      @keyframes messageFadeOut {
        from {
          opacity: 1;
          transform: translateX(-50%) translateY(0);
        }
        to {
          opacity: 0;
          transform: translateX(-50%) translateY(-10px);
        }
      }
    `
    document.head.appendChild(fadeOutStyle)
    
    // 动画结束后移除元素
    setTimeout(() => {
      messageElement.remove()
      animationStyle.remove()
      fadeOutStyle.remove()
      console.log('消息已移除')
    }, 300)
  }, duration)
}

// 便捷方法
const $message = {
  success: (message: string, duration?: number) => showMessage({ message, type: 'success', duration }),
  warning: (message: string, duration?: number) => showMessage({ message, type: 'warning', duration }),
  error: (message: string, duration?: number) => showMessage({ message, type: 'error', duration }),
  info: (message: string, duration?: number) => showMessage({ message, type: 'info', duration })
}

// 切换表情面板
const toggleEmojiPanel = () => {
  showEmojiPanel.value = !showEmojiPanel.value
  // 如果显示表情面板，关闭搜索框
  if (showEmojiPanel.value) {
    showSearch.value = false
  }
}

// 插入表情
const insertEmoji = (emoji: string) => {
  inputMessage.value += emoji
  // 关闭表情面板
  showEmojiPanel.value = false
  // 聚焦到输入框
  nextTick(() => {
    if (messageInputRef.value) {
      messageInputRef.value.focus()
    }
  })
  // 自动调整输入框高度
  autoResizeTextarea()
}

// 关闭表情面板
const closeEmojiPanel = () => {
  showEmojiPanel.value = false
}

// 监听输入框输入事件，处理 @ 功能
const handleInput = (event: Event) => {
  const textarea = event.target as HTMLTextAreaElement
  const value = textarea.value
  const cursorPos = textarea.selectionStart
  inputCursorPosition.value = cursorPos
  
  // 检查是否输入了 @ 符号
  if (value.charAt(cursorPos - 1) === '@') {
    // 显示 @ 成员面板
    showAtMembersPanel.value = true
    atMembersSearchQuery.value = ''
  }
}

// 处理输入事件并自动调整文本框高度
const handleInputAndResize = (event: Event) => {
  handleInput(event)
  autoResizeTextarea()
}

// 处理粘贴事件
const handlePaste = async (event: ClipboardEvent) => {
  const items = event.clipboardData?.items
  if (!items) return
  
  for (let i = 0; i < items.length; i++) {
    const item = items[i]
    if (item.kind === 'file') {
      event.preventDefault()
      const file = item.getAsFile()
      if (file) {
        await uploadAndSendFile(file)
      }
    }
  }
}

// 上传文件并发送
const uploadAndSendFile = async (file: File) => {
  try {
    const formData = new FormData()
    formData.append('file', file)
    
    const token = getToken()
    const response = await fetch(`${serverUrl.value}/api/v1/upload`, {
      method: 'POST',
      headers: {
        ...(token ? { 'Authorization': `Bearer ${token}` } : {})
      },
      body: formData
    })
    
    if (response.ok) {
      const data = await response.json()
      if (data.code === 0) {
        const fileUrl = data.data.url
        const fileName = data.data.name
        const fileSize = data.data.size
        
        // 构建消息对象，包含引用信息
        const messageData = {
          content: fileUrl,
          type: file.type.startsWith('image/') ? 'image' : 'file',
          file_name: fileName,
          file_size: fileSize,
          quotedMessage: quotedMessage.value
        }
        
        // 发送消息
        emit('send', messageData)
        
        // 清空引用消息
        quotedMessage.value = null
      }
    }
  } catch (error) {
    console.error('上传文件失败:', error)
    $message.error('上传文件失败')
  }
}

// 选择 @ 成员
const selectAtMember = (member: { id: string; name: string; avatar: string }) => {
  const textarea = messageInputRef.value
  if (!textarea) return
  
  const cursorPos = textarea.selectionStart
  const value = inputMessage.value
  
  // 找到 @ 符号的位置
  let atPosition = cursorPos - 1
  while (atPosition >= 0 && value.charAt(atPosition) !== '@') {
    atPosition--
  }
  
  if (atPosition >= 0) {
    // 替换 @ 符号及其后的内容为 @成员名
    const newText = value.substring(0, atPosition) + `@${member.name} ` + value.substring(cursorPos)
    inputMessage.value = newText
    
    // 自动调整输入框高度
    autoResizeTextarea()
    
    // 关闭 @ 成员面板
    showAtMembersPanel.value = false
    
    // 设置光标位置到 @成员名 后面
    nextTick(() => {
      if (textarea) {
        textarea.selectionStart = textarea.selectionEnd = atPosition + member.name.length + 2 // +2 是 @ 和空格的长度
        textarea.focus()
      }
    })
  }
}

// 关闭 @ 成员面板
const closeAtMembersPanel = () => {
  showAtMembersPanel.value = false
}

// 群成员搜索
const memberSearchQuery = ref('')
const showMemberSearch = ref(false)
const toggleMemberSearch = () => {
  showMemberSearch.value = !showMemberSearch.value
  // 如果显示搜索框，清空搜索内容并聚焦
  if (showMemberSearch.value) {
    memberSearchQuery.value = ''
    // 在下一个DOM更新周期聚焦输入框
    nextTick(() => {
      const searchInput = document.querySelector('.member-search-input') as HTMLInputElement
      if (searchInput) {
        searchInput.focus()
      }
    })
  }
}

// 群成员侧边栏展开/收缩状态
const isMembersSidebarExpanded = ref(true)
const toggleMembersSidebar = () => {
  isMembersSidebarExpanded.value = !isMembersSidebarExpanded.value
}
const filteredMembers = computed(() => {
  if (!props.conversation) {
    return []
  }
  let members = props.conversation.members || []
  
  // 排序：群主 > 管理员 > 普通成员
  members = members.sort((a, b) => {
    const rolePriority = { owner: 3, admin: 2, member: 1 }
    const aPriority = rolePriority[a.role] || 1
    const bPriority = rolePriority[b.role] || 1
    
    // 按角色优先级排序
    if (aPriority !== bPriority) {
      return bPriority - aPriority
    }
    
    // 角色相同时按名称排序
    return (a.name || '').localeCompare(b.name || '')
  })
  
  // 搜索过滤
  if (memberSearchQuery.value) {
    const query = memberSearchQuery.value.toLowerCase()
    members = members.filter(member => 
      member.name.toLowerCase().includes(query)
    )
  }
  
  return members
})

const toggleSearch = () => {
  showSearch.value = !showSearch.value
  if (!showSearch.value) {
    searchQuery.value = ''
    searchResults.value = []
    isSearching.value = false
  }
}

const performSearch = () => {
  const query = searchQuery.value.trim()
  if (!query) return
  
  isSearching.value = true
  
  // 模拟搜索延迟
  setTimeout(() => {
    searchResults.value = props.messages.filter(message => 
      message.content.toLowerCase().includes(query.toLowerCase())
    )
    isSearching.value = false
  }, 300)
}

const clearSearch = () => {
  searchQuery.value = ''
  searchResults.value = []
  isSearching.value = false
}

const handleSend = () => {
  const content = inputMessage.value.trim()
  if (!content) return
  
  // 构建消息对象，包含引用信息
  const messageData = {
    content: content,
    type: 'text',
    quotedMessage: quotedMessage.value
  }
  
  // 检测消息中是否包含@用户
  const atUsers = content.match(/@([\u4e00-\u9fa5\w]+)/g)
  if (atUsers && props.conversation?.members) {
    // 提取@的用户名
    const atUsernames = atUsers.map(atUser => atUser.substring(1))
    // 查找对应的用户
    const mentionedUsers = props.conversation.members.filter(member => 
      atUsernames.includes(member.name)
    )
    // 为@提到的用户发送通知
    mentionedUsers.forEach(user => {
      console.log('发送通知给用户:', user.name)
      // 这里可以实现通知逻辑，例如调用API发送通知
    })
  }
  
  emit('send', messageData)
  inputMessage.value = ''
  quotedMessage.value = null
}

const handleKeydown = (event) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    handleSend()
  }
  // Shift+Enter 会默认换行，不需要额外处理
}

const scrollToBottom = () => {
  nextTick(() => {
    if (messageListRef.value) {
      messageListRef.value.scrollTop = messageListRef.value.scrollHeight
      // 滚动到底部时标记消息为已读
      markMessagesAsRead()
    }
  })
}

// 标记消息为已读
const lastMarkReadTime = ref(0)
const markMessagesAsRead = async () => {
  if (!props.conversation) return
  
  // 限制调用频率，避免短时间内重复调用
  const now = Date.now()
  if (now - lastMarkReadTime.value < 3000) return
  lastMarkReadTime.value = now
  
  try {
    const response = await request(`/api/v1/conversations/${props.conversation.id}/read`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      }
    })
    
    if (response.code === 0) {
      console.log('消息已标记为已读')
      // 重新获取已读用户列表
      await loadReadUsersForMessages()
      // 这里可以触发父组件更新消息状态
    }
  } catch (error) {
    console.error('标记消息已读失败:', error)
  }
}

// 节流函数
const throttle = (func: Function, delay: number) => {
  let timeoutId: number | null = null
  return function(this: any, ...args: any[]) {
    if (timeoutId === null) {
      timeoutId = window.setTimeout(() => {
        func.apply(this, args)
        timeoutId = null
      }, delay)
    }
  }
}

// 监听消息列表滚动，当滚动到底部时标记消息为已读，当滚动到顶部时加载更多消息
const handleScroll = throttle(() => {
  if (!messageListRef.value) return
  
  const { scrollTop, scrollHeight, clientHeight } = messageListRef.value
  // 当滚动到距离底部50px以内时，标记消息为已读
  if (scrollHeight - scrollTop - clientHeight < 50) {
    markMessagesAsRead()
  }
  
  // 当滚动到顶部时，加载更多消息
  if (scrollTop < 50 && !isLoadingMore.value) {
    loadMoreMessages()
  }
}, 100)

// 加载更多消息的状态
const isLoadingMore = ref(false)
const hasMoreMessages = ref(true)

// 加载更多消息
const loadMoreMessages = async () => {
  if (!props.conversation || !hasMoreMessages.value) return
  
  isLoadingMore.value = true
  try {
    // 通知父组件加载更多消息，使用分页逻辑
    emit('loadMore', props.conversation.id)
  } catch (error) {
    console.error('加载更多消息失败:', error)
  } finally {
    isLoadingMore.value = false
  }
}

// 组件是否挂载
const isMounted = ref(true)

// 加载消息后获取已读用户列表，使用 Promise.all 并行请求
const loadReadUsersForMessages = async () => {
  if (!isMounted.value || !props.conversation || props.conversation.type !== 'group') return
  
  const promises = props.messages
    .filter(message => message.isSelf)
    .map(message => fetchReadUsers(message.id))
  
  await Promise.all(promises)
}

// 监听组件挂载和消息变化
watch(() => props.messages, async () => {
  if (!isMounted.value) return
  // 新消息到达时，滚动到底部并标记为已读
  scrollToBottom()
  // 获取已读用户列表
  await loadReadUsersForMessages()
}, { deep: true })

// 监听消息的 isRead 状态变化，重新获取已读用户列表
watch(
  () => props.messages,
  async (newMessages, oldMessages) => {
    if (!isMounted.value) return
    // 检查是否有消息的 isRead 状态发生了变化
    const hasReadStatusChanged = oldMessages && newMessages.some((newMsg, index) => {
      const oldMsg = oldMessages[index]
      return oldMsg && newMsg.isRead !== oldMsg.isRead
    })
    if (hasReadStatusChanged) {
      console.log('消息已读状态发生变化，重新获取已读用户列表')
      // 当消息的已读状态变化时，重新获取已读用户列表
      await loadReadUsersForMessages()
    }
  },
  { deep: true }
)

// 组件挂载时标记消息为已读
onMounted(async () => {
  isMounted.value = true
  // 添加滚动事件监听器
  if (messageListRef.value) {
    messageListRef.value.addEventListener('scroll', handleScroll)
  }
  // 初始标记消息为已读
  markMessagesAsRead()
  // 初始加载已读用户列表
  await loadReadUsersForMessages()
})

// 监听转发笔记事件
const handleForwardNote = (event: CustomEvent) => {
  const { content } = event.detail
  inputMessage.value = content
  autoResizeTextarea()
  console.log('收到转发笔记:', content)
}

// 监听分享文件事件
const handleShareFile = (event: CustomEvent) => {
  const { file } = event.detail
  // 直接发送文件消息
  emit('send', file)
  console.log('收到分享文件:', file)
}

// 处理全局键盘事件
const handleGlobalKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape') {
    if (showImagePreview.value) {
      closeImagePreview()
    } else if (showSharePreview.value) {
      closeSharePreview()
    }
  }
}

// 组件挂载时添加事件监听器
onMounted(async () => {
  isMounted.value = true
  // 添加滚动事件监听器
  if (messageListRef.value) {
    messageListRef.value.addEventListener('scroll', handleScroll)
  }
  // 初始标记消息为已读
  markMessagesAsRead()
  // 初始加载已读用户列表
  await loadReadUsersForMessages()
  // 加载小程序列表
  loadMiniApps()
  // 添加转发笔记事件监听器
  window.addEventListener('forwardNoteToChat', handleForwardNote as EventListener)
  // 添加分享文件事件监听器
  window.addEventListener('shareFileToChat', handleShareFile as EventListener)
  // 添加键盘事件监听器
  window.addEventListener('keydown', handleGlobalKeydown)
})

// 组件卸载时移除事件监听器
onUnmounted(() => {
  isMounted.value = false
  if (messageListRef.value) {
    messageListRef.value.removeEventListener('scroll', handleScroll)
  }
  // 移除转发笔记事件监听器
  window.removeEventListener('forwardNoteToChat', handleForwardNote as EventListener)
  // 移除分享文件事件监听器
  window.removeEventListener('shareFileToChat', handleShareFile as EventListener)
  // 移除键盘事件监听器
  window.removeEventListener('keydown', handleGlobalKeydown)
})

// 加载小程序列表
const loadMiniApps = () => {
  console.log('加载小程序列表')
  // 这里可以从服务器获取小程序列表，现在使用硬编码的列表
  // 实际项目中应该调用API获取
  const miniAppsList = [
    {
      id: 'calculator',
      name: '计算器',
      icon: 'https://api.dicebear.com/7.x/avataaars/svg?seed=calculator',
      description: '基本的加减乘除运算',
      path: '/calculator'
    },
    {
      id: 'notepad',
      name: '记事本',
      icon: 'https://api.dicebear.com/7.x/avataaars/svg?seed=notepad',
      description: '文本编辑和保存',
      path: '/notepad'
    },
    {
      id: 'todo',
      name: '待办事项',
      icon: 'https://api.dicebear.com/7.x/avataaars/svg?seed=todo',
      description: '任务管理',
      path: '/todo'
    },
    {
      id: 'password-generator',
      name: '密码生成器',
      icon: 'https://api.dicebear.com/7.x/avataaars/svg?seed=password',
      description: '生成强密码',
      path: '/password-generator'
    }
  ]
  console.log('小程序列表:', miniAppsList)
  // 这里可以将小程序列表存储到某个状态中，以便在需要时使用
}

// 滚动到引用的消息位置
const scrollToQuotedMessage = (quotedMessageId) => {
  console.log('滚动到引用消息:', quotedMessageId)
  nextTick(() => {
    const messageElement = document.querySelector(`.message-item[data-message-id="${quotedMessageId}"]`)
    if (messageElement) {
      messageElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
      // 给消息添加高亮效果
      messageElement.classList.add('highlighted-message')
      setTimeout(() => {
        messageElement.classList.remove('highlighted-message')
      }, 2000)
    }
  })
}

// 滚动到指定消息位置
const scrollToMessage = (messageId) => {
  console.log('滚动到消息:', messageId)
  nextTick(() => {
    const messageElement = document.querySelector(`.message-item[data-message-id="${messageId}"]`)
    if (messageElement) {
      messageElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
      // 给消息添加高亮效果
      messageElement.classList.add('highlighted-message')
      setTimeout(() => {
        messageElement.classList.remove('highlighted-message')
      }, 2000)
    }
  })
  // 关闭消息管理器
  closeMessageManager()
}

const autoResizeTextarea = () => {
  const textarea = messageInputRef.value
  if (textarea) {
    // 重置高度以获取正确的scrollHeight
    textarea.style.height = 'auto'
    // 设置最大高度为200px，超过则显示滚动条
    const maxHeight = 200
    const scrollHeight = textarea.scrollHeight
    // 确保高度不超过最大值
    textarea.style.height = `${Math.min(scrollHeight, maxHeight)}px`
    // 当内容超过最大高度时显示滚动条
    textarea.style.overflowY = scrollHeight > maxHeight ? 'auto' : 'hidden'
  }
}



function formatTime(timestamp: number | string | null | undefined): string {
  // 检查 timestamp 是否有效
  if (!timestamp || (typeof timestamp !== 'number' && typeof timestamp !== 'string')) {
    return '未知时间'
  }
  
  const date = new Date(timestamp)
  
  // 检查日期是否有效
  if (isNaN(date.getTime())) {
    return '未知时间'
  }
  
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const messageDate = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  const diffDays = Math.floor((today.getTime() - messageDate.getTime()) / (24 * 60 * 60 * 1000))
  
  if (diffDays === 0) {
    // 今天的消息，显示具体时间
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } else if (diffDays === 1) {
    // 昨天的消息，显示"昨天 时间"
    return `昨天 ${date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
  } else if (diffDays < 7) {
    // 本周的消息，显示星期几和时间
    const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
    const weekday = weekdays[date.getDay()]
    return `${weekday} ${date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`
  } else {
    // 更早的消息，显示具体日期和时间
    return date.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  }
}

// 判断是否应该显示时间分隔线
function shouldShowTimeDivider(index: number, currentMessage: Message): boolean {
  // 第一条消息总是显示时间
  if (index === 0) {
    return true
  }
  
  // 获取前一条消息
  const previousMessage = props.messages[index - 1]
  if (!previousMessage) {
    return true
  }
  
  // 计算时间差（毫秒）
  const timeDiff = currentMessage.timestamp - previousMessage.timestamp
  
  // 如果时间差超过5分钟，显示时间分隔线
  if (timeDiff > 5 * 60 * 1000) {
    return true
  }
  
  // 如果是不同的日期，显示时间分隔线
  const currentDate = new Date(currentMessage.timestamp)
  const previousDate = new Date(previousMessage.timestamp)
  
  if (
    currentDate.getFullYear() !== previousDate.getFullYear() ||
    currentDate.getMonth() !== previousDate.getMonth() ||
    currentDate.getDate() !== previousDate.getDate()
  ) {
    return true
  }
  
  return false
}

const showMemberContextMenu = (event: MouseEvent, member: any) => {
  event.stopPropagation()
  
  // 计算菜单位置，确保在屏幕内显示
  const menuWidth = 180 // 菜单宽度
  const menuHeight = 80 // 菜单高度
  const windowWidth = window.innerWidth
  const windowHeight = window.innerHeight
  
  let x = event.clientX
  let y = event.clientY
  
  // 调整x坐标，确保菜单不超出屏幕右侧
  if (x + menuWidth > windowWidth) {
    x = windowWidth - menuWidth - 10
  }
  
  // 调整y坐标，确保菜单不超出屏幕底部
  if (y + menuHeight > windowHeight) {
    y = windowHeight - menuHeight - 10
  }
  
  memberContextMenuPosition.value = {
    x,
    y
  }
  selectedMember.value = member
  showMemberContextMenuFlag.value = true
  
  // 点击其他地方关闭菜单
  setTimeout(() => {
    document.addEventListener('click', closeMemberContextMenu)
  }, 0)
}

const closeMemberContextMenu = () => {
  showMemberContextMenuFlag.value = false
  selectedMember.value = null
  document.removeEventListener('click', closeMemberContextMenu)
}

// 开始与成员的私聊
const startPrivateChat = (member: any) => {
  if (member && member.id) {
    emit('switchConversation', member.id)
  }
}

// 计算当前用户在群中的角色
const currentUserRole = computed(() => {
  if (!props.conversation?.members || !props.currentUser) return 'member'
  const member = props.conversation.members.find((m: any) => m.id === props.currentUser.id)
  return member?.role || 'member'
})

// 检查是否可以移除成员
const canRemoveMember = computed(() => {
  if (!selectedMember.value || currentUserRole.value === 'member') return false
  if (selectedMember.value.role === 'owner') return false
  if (currentUserRole.value === 'admin' && selectedMember.value.role === 'admin') return false
  return true
})

// 检查是否可以设置管理员
const canSetAdmin = computed(() => {
  if (!selectedMember.value || (currentUserRole.value !== 'owner' && currentUserRole.value !== 'admin')) return false
  if (selectedMember.value.role === 'owner') return false
  if (currentUserRole.value === 'admin' && selectedMember.value.role === 'admin') return false
  return true
})

// 检查是否可以转让群主
const canTransferOwner = computed(() => {
  if (!selectedMember.value || currentUserRole.value !== 'owner') return false
  if (selectedMember.value.role === 'owner') return false
  return true
})

// 检查选中的成员是否是管理员
const isSelectedMemberAdmin = computed(() => {
  return selectedMember.value?.role === 'admin'
})

const removeMemberFromGroup = async () => {
  if (!selectedMember.value) {
    closeMemberContextMenu()
    return
  }
  
  openConfirmDialog('确认移除', `确定要移除成员 ${selectedMember.value.name} 吗？`, async () => {
    try {
      const response = await request(`/api/v1/conversations/${props.conversation.id}/members/${selectedMember.value.id}`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json'
        }
      })
      
      if (response.code === 0) {
        ElMessage.success('移除成员成功')
        emit('switchConversation', props.conversation.id)
      } else {
        ElMessage.error('移除成员失败: ' + response.message)
      }
    } catch (error: any) {
      console.error('移除成员失败:', error)
      ElMessage.error('移除成员失败: ' + error.message)
    }
  })
  closeMemberContextMenu()
}

const viewMemberInfo = () => {
  if (selectedMember.value) {
    showUserProfile(selectedMember.value)
    console.log('查看资料:', selectedMember.value)
  }
  closeMemberContextMenu()
}

const setAsAdmin = async () => {
  if (!selectedMember.value) {
    closeMemberContextMenu()
    return
  }
  
  const newRole = isSelectedMemberAdmin.value ? 'member' : 'admin'
  const action = isSelectedMemberAdmin.value ? '取消管理员' : '设为管理员'
  
  openConfirmDialog('确认操作', `确定要${action}成员 ${selectedMember.value.name} 吗？`, async () => {
    try {
      const response = await request(`/api/v1/conversations/${props.conversation.id}/members/${selectedMember.value.id}/role`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ role: newRole })
      })
      
      if (response.code === 0) {
        ElMessage.success(`${action}成功`)
        emit('switchConversation', props.conversation.id)
      } else {
        ElMessage.error(`${action}失败: ` + response.message)
      }
    } catch (error: any) {
      console.error(`${action}失败:`, error)
      ElMessage.error(`${action}失败: ` + error.message)
    }
  })
  closeMemberContextMenu()
}

const transferOwner = async () => {
  if (!selectedMember.value) {
    closeMemberContextMenu()
    return
  }
  
  openConfirmDialog('确认转让群主', `确定要将群主转让给 ${selectedMember.value.name} 吗？转让后您将成为管理员。`, async () => {
    try {
      const response = await request(`/api/v1/conversations/${props.conversation.id}/members/${selectedMember.value.id}/transfer-owner`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        }
      })
      
      if (response.code === 0) {
        ElMessage.success('群主转让成功')
        emit('switchConversation', props.conversation.id)
      } else {
        ElMessage.error('群主转让失败: ' + response.message)
      }
    } catch (error: any) {
      console.error('群主转让失败:', error)
      ElMessage.error('群主转让失败: ' + error.message)
    }
  })
  closeMemberContextMenu()
}

const sendPrivateMessage = () => {
  if (selectedMember.value) {
    sendPrivateMessageToUser()
    console.log('发起私聊:', selectedMember.value)
  }
  closeMemberContextMenu()
}

const showUserProfile = (user: any) => {
  selectedUser.value = user
  showUserProfileFlag.value = true
}

const closeUserProfile = () => {
  showUserProfileFlag.value = false
  selectedUser.value = {}
}

// 重新发送失败的消息
const retrySendMessage = (message: any) => {
  console.log('重新发送消息:', message)
  emit('retry-send', message)
}

interface User {
  id: string | number
  name: string
  avatar?: string
}

const handleSendPrivateMessage = async (user: User | string | number) => {
  try {
    // 检查参数类型
    let processedUserId: string | number
    if (typeof user === 'object' && user !== null) {
      processedUserId = user.id
    } else {
      processedUserId = user
    }
    
    // 确保userId是数字类型
    if (typeof processedUserId === 'string') {
      // 如果是字符串格式（如 'emp1'），尝试提取数字部分
      if (processedUserId.startsWith('emp')) {
        processedUserId = processedUserId.replace('emp', '')
      }
      // 转换为数字
      processedUserId = parseInt(processedUserId)
    }
    
    const response = await request('/api/v1/conversations/single', {
      method: 'POST',
      body: JSON.stringify({
        user_id: processedUserId
      })
    })
    
    if (response.code === 0) {
      // 通知父组件切换到新会话
      emit('switchConversation', response.data.id.toString())
    }
  } catch (error) {
    console.error('创建私聊失败:', error)
    // 模拟创建会话（当API调用失败时）
    const mockConversationId = `conv_${Date.now()}`
    // 通知父组件切换到新会话
    emit('switchConversation', mockConversationId)
  }
  closeUserProfile()
}

const sendPrivateMessageToUser = () => {
  if (selectedUser.value) {
    handleSendPrivateMessage(selectedUser.value.id)
  }
  closeUserProfile()
}

const closeMessageContextMenu = () => {
  showMessageContextMenuFlag.value = false
  selectedMessage.value = null
  document.removeEventListener('click', closeMessageContextMenu)
}

const copyMessage = () => {
  if (selectedMessage.value && selectedMessage.value.content) {
    navigator.clipboard.writeText(selectedMessage.value.content)
      .then(() => {
        console.log('消息已复制:', selectedMessage.value.content)
      })
      .catch(err => {
        console.error('复制失败:', err)
      })
  }
  closeMessageContextMenu()
}

const forwardMessage = () => {
  if (selectedMessage.value) {
    // 触发全局事件，打开分享弹窗并传递消息数据
    window.dispatchEvent(new CustomEvent('forwardMessage', {
      detail: {
        message: selectedMessage.value
      }
    }))
  }
  closeMessageContextMenu()
}

const deleteMessage = () => {
  if (selectedMessage.value) {
    ElMessage.info('暂时不支持删除消息，因为目前没计划删除。')
  }
  closeMessageContextMenu()
}

// 消息撤回
const recallMessage = async () => {
  if (!selectedMessage.value) {
    closeMessageContextMenu()
    return
  }
  
  const messageToRecall = selectedMessage.value
  
  if (messageToRecall.isSelf) {
    openConfirmDialog('确认撤回', '确定要撤回这条消息吗？', async () => {
      try {
        const response = await request(`/api/v1/messages/${messageToRecall.id}/recall`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          }
        })
        
        if (response.code === 0) {
          // 通知父组件更新消息状态
          emit('recall', messageToRecall.id)
          closeMessageContextMenu()
          $message.success('消息撤回成功')
        } else {
          $message.error('消息撤回失败: ' + response.message)
        }
      } catch (error) {
        console.error('消息撤回失败:', error)
        $message.error('消息撤回失败: ' + error.message)
      }
    })
  } else {
    closeMessageContextMenu()
  }
}

// 判断是否可以发送提醒
const canSendReminder = (message: any): boolean => {
  if (!message.timestamp || message.isRead) return false
  
  // 群聊不支持提醒
  if (props.conversation.type === 'group') return false
  
  // 机器人消息不支持提醒
  if (message.sender && message.sender.isBot) return false
  
  const now = Date.now()
  const messageTime = new Date(message.timestamp).getTime()
  const oneHour = 60 * 60 * 1000
  
  return now - messageTime > oneHour
}

// 发送消息提醒
const sendMessageReminder = async () => {
  if (!selectedMessage.value) {
    closeMessageContextMenu()
    return
  }
  
  const message = selectedMessage.value
  
  try {
    const response = await request(`/api/v1/messages/${message.id}/remind`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      }
    })
    
    if (response.code === 0) {
      $message.success('提醒已发送')
    } else {
      $message.error('发送提醒失败: ' + response.message)
    }
  } catch (error) {
    console.error('发送提醒失败:', error)
    $message.error('发送提醒失败: ' + error.message)
  }
  
  closeMessageContextMenu()
}

// 消息引用
const quoteMessage = () => {
  if (selectedMessage.value) {
    // 设置引用消息
    console.log('设置引用消息:', selectedMessage.value)
    quotedMessage.value = selectedMessage.value
    // 聚焦到输入框
    const input = document.querySelector('.message-input') as HTMLTextAreaElement
    if (input) {
      input.focus()
    }
  }
  closeMessageContextMenu()
}

// 将消息添加到便签
const addToNote = () => {
  if (selectedMessage.value) {
    const message = selectedMessage.value
    
    // 检查消息类型，仅支持文本类型
    if (message.type !== 'text') {
      $message.warning('仅支持文本类型的消息添加到便签')
      closeMessageContextMenu()
      return
    }
    
    // 构建便签内容
    const noteContent = `【聊天记录】
发送者：${message.sender.name}
时间：${formatTime(message.timestamp)}
内容：${message.content}`
    
    // 触发全局事件，通知便签应用接收内容
    window.dispatchEvent(new CustomEvent('addToNote', {
      detail: { 
        title: `聊天记录 ${formatTime(message.timestamp)}`,
        content: noteContent 
      }
    }))
    
    $message.success('消息已添加到便签')
    console.log('添加消息到便签:', message)
  }
  closeMessageContextMenu()
}

// 截图相关状态
const showScreenshotPreview = ref(false)
const screenshotImageData = ref('')

// 检测是否在Electron环境中
const isElectron = computed(() => {
  // 开发环境中也返回true，让用户能看到截图按钮
  // 实际运行时会自动使用模拟截图功能
  return true || (window.electron && window.electron.ipcRenderer && typeof window.electron.ipcRenderer.once === 'function')
})

const takeScreenshot = () => {
  // 检查是否在Electron环境中，并且ipcRenderer有once方法
  if (window.electron && window.electron.ipcRenderer && typeof window.electron.ipcRenderer.once === 'function') {
    // 发送截图请求到主进程
    window.electron.ipcRenderer.send('take-screenshot')
    
    // 监听截图结果
    window.electron.ipcRenderer.once('screenshot-taken', async (event, imageData) => {
      // 处理截图结果
      console.log('截图成功:', imageData)
      if (imageData) {
        // 显示截图预览
        screenshotImageData.value = imageData
        showScreenshotPreview.value = true
      }
    })
  } else {
    // 非Electron环境或ipcRenderer不完整的模拟实现
    console.log('截图功能仅在完整的Electron环境中可用')
    // 模拟截图功能
    simulateScreenshot()
  }
}

// 模拟截图功能（用于非Electron环境）
const simulateScreenshot = () => {
  // 生成一个模拟的截图数据
  const canvas = document.createElement('canvas')
  canvas.width = 800
  canvas.height = 600
  const ctx = canvas.getContext('2d')
  if (ctx) {
    // 绘制一个简单的模拟截图
    ctx.fillStyle = '#f0f0f0'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
    ctx.fillStyle = '#333'
    ctx.font = '24px Arial'
    ctx.fillText('模拟截图', 300, 300)
    ctx.fillStyle = '#666'
    ctx.font = '16px Arial'
    ctx.fillText('这是一个模拟的截图效果', 280, 330)
    
    // 将canvas转换为base64
    const imageData = canvas.toDataURL('image/png')
    screenshotImageData.value = imageData
    showScreenshotPreview.value = true
  }
}

// 上传截图到服务器
const uploadScreenshot = async () => {
  if (!screenshotImageData.value) return
  
  try {
    // 将base64转换为Blob
    const response = await fetch(screenshotImageData.value)
    const blob = await response.blob()
    
    // 创建FormData
    const formData = new FormData()
    formData.append('file', blob, 'screenshot.png')
    
    // 上传到服务器
    const uploadResponse = await fetch(`${serverUrl.value}/api/v1/upload`, {
      method: 'POST',
      headers: {
        ...(getToken() ? { 'Authorization': `Bearer ${getToken()}` } : {})
      },
      body: formData
    })
    
    if (uploadResponse.ok) {
      const data = await uploadResponse.json()
      if (data.code === 0) {
        // 上传成功，获取文件URL
        const fileUrl = data.data.url
        console.log('截图上传成功:', fileUrl)
        
        // 构建图片消息
        const messageData = {
          content: fileUrl,
          type: 'image'
        }
        
        // 发送消息
        emit('send', messageData)
        
        // 关闭预览
        showScreenshotPreview.value = false
        screenshotImageData.value = ''
        
        $message.success('截图发送成功')
      } else {
        $message.error('截图上传失败: ' + data.message)
      }
    } else {
      $message.error('截图上传失败: 服务器错误')
    }
  } catch (error) {
    console.error('截图上传失败:', error)
    $message.error('截图上传失败: 网络错误')
  }
}

// 取消截图
const cancelScreenshot = () => {
  showScreenshotPreview.value = false
  screenshotImageData.value = ''
}

// 重新截图
const retakeScreenshot = () => {
  showScreenshotPreview.value = false
  screenshotImageData.value = ''
  takeScreenshot()
}

// 通话相关状态
const isInCall = ref(false)
const callType = ref('') // 'voice' 或 'video'
const callStatus = ref('') // 'ringing', 'answered', 'ended'
const showCallModal = ref(false)
const isScreenSharing = ref(false) // 是否正在共享屏幕

// 小程序列表
const showMiniAppList = ref(false)

const screenStream = ref(null) // 屏幕共享流

// 开始语音通话
const startVoiceCall = () => {
  if (!props.conversation) return
  
  // 检查是否在通话中
  if (isInCall.value) {
    $message.warning('您已经在通话中')
    return
  }
  
  // 开始语音通话
  callType.value = 'voice'
  callStatus.value = 'ringing'
  isInCall.value = true
  showCallModal.value = true
  
  // 模拟通话请求
  simulateCallRequest('voice')
}

// 开始视频通话
const startVideoCall = () => {
  if (!props.conversation) return
  
  // 检查是否在通话中
  if (isInCall.value) {
    $message.warning('您已经在通话中')
    return
  }
  
  // 开始视频通话
  callType.value = 'video'
  callStatus.value = 'ringing'
  isInCall.value = true
  showCallModal.value = true
  
  // 模拟通话请求
  simulateCallRequest('video')
}

// 模拟通话请求
const simulateCallRequest = (type) => {
  // 模拟对方接听
  setTimeout(() => {
    callStatus.value = 'answered'
    $message.success('对方已接听')
  }, 3000)
}

// 结束通话
const endCall = () => {
  callStatus.value = 'ended'
  isInCall.value = false
  showCallModal.value = false
  
  // 停止屏幕共享
  if (isScreenSharing.value) {
    stopScreenShare()
  }
  
  // 模拟通话结束
  setTimeout(() => {
    callType.value = ''
    callStatus.value = ''
  }, 1000)
  
  $message.info('通话已结束')
}

// 拒绝通话
const rejectCall = () => {
  callStatus.value = 'ended'
  isInCall.value = false
  showCallModal.value = false
  
  // 模拟通话结束
  setTimeout(() => {
    callType.value = ''
    callStatus.value = ''
  }, 1000)
  
  $message.info('已拒绝通话')
}

// 接听通话
const answerCall = () => {
  callStatus.value = 'answered'
  $message.success('已接听通话')
}

// 开始屏幕共享
const startScreenShare = async () => {
	if (!props.conversation) return
	
	try {
		// 检查浏览器是否支持屏幕共享
		if (!navigator.mediaDevices || !navigator.mediaDevices.getDisplayMedia) {
			$message.error('您的浏览器不支持屏幕共享功能，请使用Chrome、Firefox或Edge浏览器')
			return
		}
		
		// 请求屏幕共享权限
		const stream = await navigator.mediaDevices.getDisplayMedia({
			video: {
				cursor: 'always'
			},
			audio: false
		})
		
		// 保存屏幕共享流
		screenStream.value = stream
		isScreenSharing.value = true
		
		// 显示屏幕共享开始消息
		$message.success('屏幕共享已开始')
		
		// 监听屏幕共享结束
		stream.getVideoTracks()[0].onended = () => {
			stopScreenShare()
		}
	} catch (error) {
		console.error('屏幕共享失败:', error)
		if (error.name === 'NotSupportedError') {
			$message.error('屏幕共享功能在当前环境中不支持，请使用支持屏幕共享的浏览器')
		} else if (error.name === 'NotAllowedError') {
			$message.error('屏幕共享权限被拒绝，请允许浏览器访问您的屏幕')
		} else if (error.name === 'AbortError') {
			// 用户取消了屏幕共享选择
			$message.info('屏幕共享已取消')
		} else {
			$message.error('屏幕共享失败，请稍后重试')
		}
	}
}

// 停止屏幕共享
const stopScreenShare = () => {
  if (screenStream.value) {
    screenStream.value.getTracks().forEach(track => track.stop())
    screenStream.value = null
  }
  isScreenSharing.value = false
  $message.info('屏幕共享已结束')
}

const selectFile = () => {
  // 触发文件选择对话框
  fileInput.value?.click()
}

const selectImage = () => {
  // 创建一个临时的文件输入元素
  const imageInput = document.createElement('input')
  imageInput.type = 'file'
  imageInput.accept = 'image/*'
  imageInput.multiple = true
  
  // 监听文件选择事件
  imageInput.addEventListener('change', async (event) => {
    const target = event.target as HTMLInputElement
    const files = target.files
    
    if (files && files.length > 0) {
      // 处理选中的图片文件
      for (let i = 0; i < files.length; i++) {
        const file = files[i]
        console.log('选中的图片:', file.name, file.size)
        
        try {
          // 上传文件
          const formData = new FormData()
          formData.append('file', file)
          
          const response = await fetch(`${serverUrl.value}/api/v1/upload`, {
            method: 'POST',
            headers: {
              ...(getToken() ? { 'Authorization': `Bearer ${getToken()}` } : {})
            },
            body: formData
          })
          
          if (response.ok) {
            const data = await response.json()
            if (data.code === 0) {
              // 上传成功，获取文件URL
              const fileUrl = data.data.url
              // 构建完整的文件URL
              const fullFileUrl = fileUrl.startsWith('http') ? fileUrl : `${serverUrl.value}${fileUrl}`
              console.log('图片上传成功:', fullFileUrl)
              
              // 构建图片消息并直接发送
              const messageData = {
                content: fullFileUrl,
                type: 'image',
                fileSize: file.size,
                fileName: file.name
              }
              
              // 发送消息
              emit('send', messageData)
            } else {
              $message.error('图片上传失败: ' + data.message)
            }
          } else {
            $message.error('图片上传失败: 服务器错误')
          }
        } catch (error) {
          console.error('图片上传失败:', error)
          $message.error('图片上传失败: 网络错误')
        }
      }
    }
  })
  
  // 触发图片选择对话框
  imageInput.click()
}

const handleFileSelect = async (event: Event) => {
  const target = event.target as HTMLInputElement
  const files = target.files
  
  if (files && files.length > 0) {
    // 处理选中的文件
    for (let i = 0; i < files.length; i++) {
      const file = files[i]
      console.log('选中的文件:', file.name, file.size)
      
      try {
        // 上传文件
        const formData = new FormData()
        formData.append('file', file)
        
        const response = await fetch(`${serverUrl.value}/api/v1/upload`, {
          method: 'POST',
          headers: {
            ...(getToken() ? { 'Authorization': `Bearer ${getToken()}` } : {})
          },
          body: formData
        })
        
        if (response.ok) {
          const data = await response.json()
          if (data.code === 0) {
            // 上传成功，获取文件URL
            const fileUrl = data.data.url
            // 构建完整的文件URL
            const fullFileUrl = fileUrl.startsWith('http') ? fileUrl : `${serverUrl.value}${fileUrl}`
            console.log('文件上传成功:', fullFileUrl)
            
            // 判断文件类型
            const isImage = file.type.startsWith('image/')
            const messageType = isImage ? 'image' : 'file'
            
            // 构建消息并直接发送
            const messageData = {
              content: fullFileUrl,
              type: messageType,
              fileSize: file.size,
              fileName: file.name
            }
            
            // 发送消息
            emit('send', messageData)
          } else {
            $message.error('文件上传失败: ' + data.message)
          }
        } else {
          $message.error('文件上传失败: 服务器错误')
        }
      } catch (error) {
        console.error('文件上传失败:', error)
        $message.error('文件上传失败: 网络错误')
      }
    }
    
    // 清空文件输入，以便可以重复选择同一个文件
    if (fileInput.value) {
      fileInput.value.value = ''
    }
  }
}

const saveFileAs = async (fileContent: string, fileName?: string) => {
  // 优先使用传入的文件名，否则从fileContent中提取
  const finalFileName = fileName || fileContent.split('/').pop() || fileContent
  console.log('另存为文件:', finalFileName)
  
  try {
    // 构建完整的文件下载URL
    const fileUrl = fileContent.startsWith('http') ? fileContent : `${serverUrl.value}${fileContent}`
    
    // 发起下载请求
    const response = await fetch(fileUrl, {
      method: 'GET',
      headers: {
        ...(getToken() ? { 'Authorization': `Bearer ${getToken()}` } : {})
      }
    })
    
    if (response.ok) {
      // 创建Blob对象
      const blob = await response.blob()
      
      // 创建下载链接
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = finalFileName
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      window.URL.revokeObjectURL(url)
      
      $message.success(`文件 ${finalFileName} 保存成功`)
    } else {
      $message.error('文件保存失败: 服务器错误')
    }
  } catch (error) {
    console.error('文件保存失败:', error)
    $message.error('文件保存失败: 网络错误')
  }
}

// 查看分享的内容
// 分享内容预览相关
const showSharePreview = ref(false)
const sharePreviewData = ref<any>(null)

const viewSharedContent = (content: string) => {
  if (!content || content === '[消息已撤回]') return
  
  try {
    const shareData = JSON.parse(content)
    // 根据分享的类型和ID，跳转到对应的应用或页面
    if (shareData.type === 'file' || shareData.type === 'note' || shareData.type === 'sticky') {
      // 对于文件、笔记和便签，在当前页展示
      sharePreviewData.value = shareData
      showSharePreview.value = true
    } else {
      $message.info(`查看分享内容: ${shareData.name}`)
    }
  } catch (e) {
    console.error('解析分享数据失败:', e)
    $message.error('查看分享内容失败')
  }
}

const closeSharePreview = () => {
  showSharePreview.value = false
  sharePreviewData.value = null
}

const showImagePreview = ref(false)
const previewImageUrl = ref('')

const previewImage = (imageUrl: string) => {
  console.log('预览图片:', imageUrl)
  previewImageUrl.value = imageUrl
  showImagePreview.value = true
}

const closeImagePreview = () => {
  showImagePreview.value = false
  previewImageUrl.value = ''
}

const downloadFile = async (fileContent: string, fileName?: string) => {
  // 优先使用传入的文件名，否则从fileContent中提取
  const finalFileName = fileName || fileContent.split('/').pop() || fileContent
  console.log('下载文件:', finalFileName)
  
  try {
    // 构建完整的文件下载URL
    const fileUrl = fileContent.startsWith('http') ? fileContent : `${serverUrl.value}${fileContent}`
    
    // 发起下载请求
    const response = await fetch(fileUrl, {
      method: 'GET',
      headers: {
        ...(getToken() ? { 'Authorization': `Bearer ${getToken()}` } : {})
      }
    })
    
    if (response.ok) {
      // 创建Blob对象
      const blob = await response.blob()
      
      // 创建下载链接
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = finalFileName
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      window.URL.revokeObjectURL(url)
      
      $message.success(`文件 ${finalFileName} 下载成功`)
    } else {
      $message.error('文件下载失败: 服务器错误')
    }
  } catch (error) {
    console.error('文件下载失败:', error)
    $message.error('文件下载失败: 网络错误')
  }
}

// 根据文件扩展名获取对应的Font Awesome图标
const getFileIcon = (fileUrl: string): string => {
  const fileName = fileUrl.split('/').pop() || fileUrl
  const extension = fileName.split('.').pop()?.toLowerCase() || ''
  
  switch (extension) {
    // 文档类
    case 'doc':
    case 'docx':
      return 'fas fa-file-word'
    case 'xls':
    case 'xlsx':
      return 'fas fa-file-excel'
    case 'ppt':
    case 'pptx':
      return 'fas fa-file-powerpoint'
    case 'pdf':
      return 'fas fa-file-pdf'
    case 'txt':
      return 'fas fa-file-alt'
    case 'md':
      return 'fas fa-file-markdown'
    
    // 图片类
    case 'jpg':
    case 'jpeg':
    case 'png':
    case 'gif':
    case 'webp':
    case 'bmp':
      return 'fas fa-file-image'
    
    // 音频类
    case 'mp3':
    case 'wav':
    case 'ogg':
    case 'flac':
      return 'fas fa-file-audio'
    
    // 视频类
    case 'mp4':
    case 'avi':
    case 'mov':
    case 'wmv':
    case 'flv':
      return 'fas fa-file-video'
    
    // 压缩包类
    case 'zip':
    case 'rar':
    case '7z':
    case 'tar':
    case 'gz':
      return 'fas fa-file-archive'
    
    // 代码类
    case 'js':
    case 'ts':
    case 'jsx':
    case 'tsx':
    case 'html':
    case 'css':
    case 'scss':
    case 'less':
    case 'json':
    case 'xml':
    case 'yaml':
    case 'yml':
    case 'py':
    case 'java':
    case 'c':
    case 'cpp':
    case 'cs':
    case 'go':
    case 'php':
    case 'rb':
    case 'swift':
    case 'kt':
      return 'fas fa-file-code'
    
    // 其他
    default:
      return 'fas fa-file'
  }
}

// 格式化文件大小
const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// 将文本中的URL转换为可点击的超链接，并为@提到的用户添加高亮显示
const convertUrlsToLinks = (text: string): string => {
  // 正则表达式匹配URL
  const urlRegex = /(https?:\/\/[\w\-._~:/?#[\]@!$&'()*+,;=.]+)/g
  // 正则表达式匹配@用户
  const atRegex = /@([\u4e00-\u9fa5\w]+)/g
  
  let result = text
  
  // 先处理URL
  result = result.replace(urlRegex, (url) => {
    return `<a href="${url}" target="_blank" rel="noopener noreferrer" class="message-link">${url}</a>`
  })
  
  // 再处理@用户
  result = result.replace(atRegex, (match, username) => {
    return `<span class="at-user">@${username}</span>`
  })
  
  return result
}

// 消息右键菜单添加文件相关选项
const showMessageContextMenu = (event: MouseEvent, message: Message) => {
  event.stopPropagation()
  
  // 已撤回的消息不显示右键菜单
  if (message.isRecalled) {
    return
  }
  
  // 计算菜单位置，确保在屏幕内显示
  const menuWidth = 180 // 菜单宽度
  const menuHeight = 120 // 菜单高度
  const windowWidth = window.innerWidth
  const windowHeight = window.innerHeight
  
  let x = event.clientX
  let y = event.clientY
  
  // 调整x坐标，确保菜单不超出屏幕右侧
  if (x + menuWidth > windowWidth) {
    x = windowWidth - menuWidth - 10
  }
  
  // 调整y坐标，确保菜单不超出屏幕底部
  if (y + menuHeight > windowHeight) {
    y = windowHeight - menuHeight - 10
  }
  
  messageContextMenuPosition.value = { x, y }
  showMessageContextMenuFlag.value = true
  selectedMessage.value = message
  
  // 检查消息类型
  if (message.type === 'file' || message.type === 'image') {
    // 可以在这里添加文件或图片特定的菜单选项
    console.log('显示文件/图片消息的右键菜单')
  }
  
  // 点击其他地方关闭菜单
  setTimeout(() => {
    document.addEventListener('click', closeMessageContextMenu)
  }, 0)
}

// 处理邀请成员
const handleInviteMembers = () => {
  if (props.conversation?.id) {
    emit('inviteMembers', props.conversation.id)
  }
}

// 切换头部下拉菜单
const toggleHeaderMenu = () => {
  showHeaderMenu.value = !showHeaderMenu.value
  // 点击其他地方关闭菜单
  if (showHeaderMenu.value) {
    setTimeout(() => {
      document.addEventListener('click', closeHeaderMenu)
    }, 0)
  }
}

// 关闭头部下拉菜单
const closeHeaderMenu = () => {
  showHeaderMenu.value = false
  document.removeEventListener('click', closeHeaderMenu)
}

// 检查是否有权限修改群名称
const canEditGroupName = computed(() => {
  if (!props.conversation) return false
  
  // 讨论组全员可修改
  if (props.conversation.type === 'discussion') {
    return true
  }
  
  // 群只有管理员和群主能修改
  if (props.conversation.type === 'group') {
    const userRole = currentUserRole.value
    return userRole === 'owner' || userRole === 'admin'
  }
  
  return false
})

// 编辑群信息
const editGroupInfo = () => {
  if (props.conversation && canEditGroupName.value) {
    editGroupName.value = props.conversation.name || ''
    showEditGroupInfoModal.value = true
  } else if (props.conversation && !canEditGroupName.value) {
    ElMessage.warning('只有管理员和群主可以修改群名称')
  }
  closeHeaderMenu()
}

// 保存群信息
const saveGroupInfo = async () => {
  if (!props.conversation) {
    showEditGroupInfoModal.value = false
    return
  }
  
  try {
    const response = await request(`/api/v1/conversations/${props.conversation.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ name: editGroupName.value })
    })
    
    if (response.code === 0) {
      ElMessage.success('群名称已成功更新')
      // 更新本地群聊数据
      if (props.updateConversation) {
        props.updateConversation({ ...props.conversation, name: editGroupName.value })
      }
      
      // 发送群名称修改系统消息
      const systemMessage = {
        id: `system_${Date.now()}`,
        type: 'system',
        content: `${props.currentUser.name} 修改群名称为 ${editGroupName.value}`,
        timestamp: Date.now(),
        sender: {
          id: 'system',
          name: '系统',
          avatar: ''
        },
        isSelf: false,
        isRead: true
      }
      
      // 这里可以通过emit将系统消息传递给父组件，或者直接添加到本地消息列表
      // 暂时直接添加到本地消息列表
      if (Array.isArray(props.messages)) {
        props.messages.push(systemMessage)
      }
    } else {
      ElMessage.error(response.message || '更新群名称失败')
    }
  } catch (error) {
    console.error('更新群名称失败:', error)
    ElMessage.error('网络错误，更新群名称失败')
  }
  showEditGroupInfoModal.value = false
}

// 编辑群公告
const editGroupAnnouncement = () => {
  if (props.conversation) {
    editAnnouncementContent.value = props.conversation.announcement || ''
    showEditAnnouncementModal.value = true
  }
  closeHeaderMenu()
}

// 保存群公告
const saveAnnouncement = async () => {
  if (!props.conversation) {
    showEditAnnouncementModal.value = false
    return
  }
  
  try {
    const response = await request(`/api/v1/conversations/${props.conversation.id}/announcement`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ announcement: editAnnouncementContent.value })
    })
    
    if (response.code === 0) {
      ElMessage.success('群公告已成功更新')
      // 更新本地群聊数据
      if (props.updateConversation) {
        props.updateConversation({ ...props.conversation, announcement: editAnnouncementContent.value })
      }
    } else {
      ElMessage.error(response.message || '更新群公告失败')
    }
  } catch (error) {
    console.error('更新群公告失败:', error)
    ElMessage.error('网络错误，更新群公告失败')
  }
  showEditAnnouncementModal.value = false
}
</script>

<style scoped>
.chat-window {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--sidebar-bg);
  box-shadow: 0 0 10px rgba(0, 0, 0, 0.05);
  border-radius: 0;
  margin: 0;
  overflow: hidden;
}

.chat-main {
  flex: 1;
  display: flex;
  overflow: hidden;
  background: var(--content-bg);
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.03);
}

.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: var(--sidebar-bg);
  height: 72px;
  box-sizing: border-box;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  margin: 0;
  border-radius: 0;
  border-bottom: 1px solid var(--border-color);
}

.header-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  object-fit: cover;
}

.header-text {
  display: flex;
  flex-direction: column;
}

.header-name {
  font-weight: 500;
  font-size: 16px;
  color: var(--text-color);
}

.header-status {
  font-size: 12px;
  color: #4caf50;
  display: flex;
  align-items: center;
  gap: 8px;
}

.ip-info {
  color: var(--text-color);
  opacity: 0.7;
  font-size: 11px;
  margin-left: 8px;
  padding: 2px 6px;
  background: var(--hover-color);
  border-radius: 3px;
}

.header-actions {
  display: flex;
  gap: 8px;
  position: relative;
}

.header-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--text-color);
  opacity: 0.7;
  font-size: 14px;
  border-radius: 6px;
  transition: background 0.2s;
  position: relative;
}

.header-icon:hover {
  background: var(--hover-color);
  opacity: 1;
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  background: var(--content-bg);
  box-shadow: 2px 0 10px rgba(0, 0, 0, 0.03);
}



/* 消息管理器样式 */
.message-manager-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(5px);
}

.message-manager-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
  width: 90%;
  max-width: 900px;
  max-height: 85vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.message-manager-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: var(--sidebar-bg);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.message-manager-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
  display: flex;
  align-items: center;
  gap: 10px;
}

.message-manager-header h3::before {
  /* content: '📋'; */
  font-size: 20px;
}

.message-manager-body {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
  background: var(--sidebar-bg);
}

.message-manager-search {
  margin-bottom: 24px;
  position: relative;
}

.message-manager-search .search-input {
  width: 100%;
  padding: 12px 16px;
  border-radius: 8px;
  font-size: 14px;
  transition: all 0.2s ease;
  background: var(--sidebar-bg);
  color: var(--text-color);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.message-manager-search .search-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(25, 118, 210, 0.1);
}

.message-manager-filters {
  display: flex;
  gap: 24px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}

.filter-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
  min-width: 150px;
}

.filter-group label {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  opacity: 0.8;
}

.filter-select {
  padding: 10px 16px;
  border-radius: 8px;
  font-size: 14px;
  background: var(--sidebar-bg);
  color: var(--text-color);
  transition: all 0.2s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.filter-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(25, 118, 210, 0.1);
}

/* 日期范围选择器样式 */
.date-range-group {
  flex: 2;
  min-width: 300px;
}

.date-range-inputs {
  display: flex;
  align-items: center;
  gap: 12px;
}

.date-input {
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 14px;
  background: var(--sidebar-bg);
  color: var(--text-color);
  transition: all 0.2s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  flex: 1;
}

.date-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(25, 118, 210, 0.1);
}

.date-range-separator {
  font-size: 14px;
  color: var(--text-color);
  opacity: 0.7;
  white-space: nowrap;
}

/* 分页控件样式 */
.message-manager-pagination {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 24px;
  padding-top: 16px;
  background: var(--sidebar-bg);
  border-radius: 0 0 12px 12px;
  padding: 16px 20px;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.05);
}

/* 悬浮分页效果 */
.sticky-pagination {
  position: sticky;
  bottom: 0;
  z-index: 10;
  margin-top: 0;
  border-radius: 0;
}

.pagination-info {
  font-size: 14px;
  color: var(--text-color);
  opacity: 0.7;
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 12px;
}

.pagination-btn {
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 14px;
  background: var(--sidebar-bg);
  color: var(--text-color);
  cursor: pointer;
  transition: all 0.2s ease;
}

.pagination-btn:hover:not(:disabled) {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.pagination-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  background: var(--sidebar-bg);
  color: var(--text-color);
}

.pagination-current {
  font-size: 14px;
  color: var(--text-color);
  min-width: 80px;
  text-align: center;
}

.page-jump {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: 16px;
}

.page-input {
  width: 60px;
  padding: 6px 8px;
  border-radius: 6px;
  font-size: 14px;
  border: 1px solid var(--border-color);
  background: var(--sidebar-bg);
  color: var(--text-color);
  text-align: center;
  transition: all 0.2s ease;
}

.page-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(25, 118, 210, 0.1);
}

.jump-btn {
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 14px;
  background: var(--primary-color);
  color: #fff;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
}

.jump-btn:hover {
  background: var(--primary-color);
  opacity: 0.9;
  transform: translateY(-1px);
}

.jump-btn:active {
  transform: translateY(0);
}

.jump-btn:disabled {
  background: var(--border-color);
  cursor: not-allowed;
  opacity: 0.6;
  transform: none;
}

.message-manager-list {
  max-height: 450px;
  overflow-y: auto;
  background: var(--sidebar-bg);
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.message-manager-item {
  padding: 10px 16px;
  transition: all 0.2s ease;
  cursor: pointer;
  border-bottom: 1px solid var(--border-color);
}

.message-manager-item:hover {
  background: var(--hover-color);
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.message-manager-item:last-child {
  border-bottom: none;
}

.message-manager-item-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
  flex-wrap: wrap;
}

.message-sender {
  font-weight: 600;
  color: var(--text-color);
  font-size: 13px;
  flex: 1;
  min-width: 80px;
}

.message-time {
  font-size: 11px;
  color: var(--text-color);
  opacity: 0.6;
  transition: opacity 0.3s ease;
  flex: 0 0 auto;
}

.message-item:hover .message-time {
  opacity: 1;
}

/* 消息管理器中的消息时间始终显示 */
.message-manager-item .message-time {
  opacity: 0.7;
}

.message-manager-item:hover .message-time {
  opacity: 1;
}

.message-type {
  font-size: 11px;
  font-weight: 500;
  color: var(--primary-color);
  background: var(--hover-color);
  padding: 2px 8px;
  border-radius: 10px;
  flex: 0 0 auto;
}

.message-manager-item-content {
  font-size: 13px;
  color: var(--text-color);
  line-height: 1.4;
  padding-left: 0;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 60px;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
}



.message-content-file {
  display: flex;
  align-items: center;
}

.message-mini-app,
.message-share,
.message-news {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
}

.message-mini-app > span,
.message-share > span,
.message-news > span {
  display: flex;
  align-items: center;
  gap: 8px;
}

.message-mini-app i,
.message-share i,
.message-news i {
  color: var(--primary-color);
}

.mini-app-description,
.share-type,
.news-summary {
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.3;
  /* margin-left: 24px; */
}

.message-file-link {
  display: flex;
  align-items: center;
  color: #64b5f6;
  text-decoration: none;
  transition: all 0.3s ease;
}

.message-file-link:hover {
  color: #42a5f5;
  text-decoration: underline;
}

.message-file-link i {
  margin-right: 8px;
}

.message-type {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 12px;
  font-size: 12px;
  font-weight: 500;
}

.message-type-text {
  background-color: #e3f2fd;
  color: #1976d2;
}

.message-type-image {
  background-color: #e8f5e8;
  color: #388e3c;
}

.message-type-file {
  background-color: #fff3e0;
  color: #f57c00;
}

/* 分享消息样式 */
.share-message {
  background: var(--sidebar-bg);
  border-radius: 12px;
  padding: 14px;
  max-width: 400px;
  min-width: 250px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
}

.share-message:hover {
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12);
}

.share-info {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.share-icon-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.share-icon {
  font-size: 24px;
  margin-top: 2px;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--list-bg);
  border-radius: 6px;
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.share-details {
  flex: 1;
  min-width: 0;
}

.share-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 8px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
}

.share-type {
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.2;
  white-space: nowrap;
  text-align: center;
  margin-bottom: 4px;
}

.share-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.share-action-btn {
  padding: 6px 16px;
  font-size: 12px;
  border-radius: 8px;
  border: none;
  background-color: var(--primary-light);
  color: var(--primary-color);
  /* border: 1px solid var(--primary-color); */
  cursor: pointer;
  transition: all 0.3s ease;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}

.share-action-btn:hover {
  background-color: var(--primary-color);
  color: #fff;
  border-color: var(--primary-color);
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.3);
  transform: translateY(-1px);
}

.share-action-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.3);
}

/* 自己发送的分享消息 */
.message-item.self .share-message {
  background: var(--primary-color);
}

.message-item.self .share-name {
  color: #fff;
}

.message-item.self .share-type {
  color: rgba(255, 255, 255, 0.8);
}

.message-item.self .share-icon {
  background-color: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.3);
  color: #fff;
}

.message-item.self .share-action-btn {
  background-color: rgba(255, 255, 255, 0.2);
  border: none;
  border-color: rgba(255, 255, 255, 0.3);
  color: #fff;
}

.message-item.self .share-action-btn:hover {
  background-color: rgba(255, 255, 255, 0.3);
  border-color: rgba(255, 255, 255, 0.4);
  box-shadow: 0 4px 12px rgba(255, 255, 255, 0.3);
}

.message-item.self .share-action-btn:active {
  box-shadow: 0 2px 8px rgba(255, 255, 255, 0.3);
}

.loading-message {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
  font-size: 14px;
  color: var(--text-color);
  opacity: 0.6;
  background: var(--sidebar-bg);
  border-radius: 8px;
  margin: 16px 0;
}

.message-manager-header h3 i {
  margin-right: 8px;
  color: #64b5f6;
}

.empty-message {
  text-align: center;
  padding: 60px 0;
  color: var(--text-color);
  opacity: 0.6;
  font-size: 14px;
  background: var(--sidebar-bg);
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.empty-message::before {
  content: '📭';
  display: block;
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
}

/* 滚动条样式 */
.message-manager-list::-webkit-scrollbar,
.message-manager-body::-webkit-scrollbar {
  width: 6px;
}

.message-manager-list::-webkit-scrollbar-track,
.message-manager-body::-webkit-scrollbar-track {
  background: var(--sidebar-bg);
  border-radius: 3px;
}

.message-manager-list::-webkit-scrollbar-thumb,
.message-manager-body::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
}

.message-manager-list::-webkit-scrollbar-thumb:hover,
.message-manager-body::-webkit-scrollbar-thumb:hover {
  background: var(--text-color);
  opacity: 0.5;
}

.message-item {
  display: flex;
  align-items: flex-start;
  margin-bottom: 16px;
}

.message-item.self {
  flex-direction: row-reverse;
}

.message-item.self .message-sender {
  display: none;
}

.message-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.message-content {
  max-width: 60%;
  margin: 0 12px;
}

.message-link {
  color: #3b82f6;
  text-decoration: none;
  font-weight: 500;
  transition: all 0.3s ease;
}

.message-link:hover {
  color: #2563eb;
  text-decoration: underline;
  transform: translateY(-1px);
}

.at-user {
  color: #3b82f6;
  font-weight: 600;
  background-color: rgba(59, 130, 246, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
  transition: all 0.3s ease;
}

.at-user:hover {
  background-color: rgba(59, 130, 246, 0.2);
  transform: translateY(-1px);
}

.message-bubble {
  padding: 10px 14px;
  border-radius: 12px;
  background: var(--sidebar-bg);
  color: var(--text-color);
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;
  white-space: pre-wrap;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.message-item.self .message-bubble {
  background: var(--primary-color);
  color: white;
  border: none;
}

/* 为其他主题保留原来的样式 */
[data-theme="dark"] .message-item.self .message-bubble {
  background: var(--primary-color);
  color: #fff;
  border: none;
}

[data-theme="dark"] .message-item.self .file-message {
  background: var(--primary-color);
  color: var(--secondary-color);
}

[data-theme="dark"] .message-item.self .recalled-message {
  background: rgba(255, 255, 255, 0.1) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

[data-theme="netblue"] .message-item.self .message-bubble {
  background: var(--primary-color);
  color: #fff;
  border: none;
}

[data-theme="netblue"] .message-item.self .file-message {
  background: var(--primary-color);
  color: #fff;
}

[data-theme="netblue"] .message-item.self .recalled-message {
  background: rgba(66, 153, 225, 0.8) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

[data-theme="elegantpurple"] .message-item.self .message-bubble {
  background: var(--primary-color);
  color: #fff;
  border: none;
}

[data-theme="elegantpurple"] .message-item.self .file-message {
  background: var(--primary-color);
  color: #fff;
}

[data-theme="elegantpurple"] .message-item.self .recalled-message {
  background: rgba(139, 92, 246, 0.8) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

[data-theme="sacredyellow"] .message-item.self .message-bubble {
  background: var(--primary-color);
  color: #fff;
  border: none;
}

[data-theme="sacredyellow"] .message-item.self .file-message {
  background: var(--primary-color);
  color: #fff;
}

[data-theme="sacredyellow"] .message-item.self .recalled-message {
  background: rgba(217, 119, 6, 0.8) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

[data-theme="chinesered"] .message-item.self .message-bubble {
  background: var(--primary-color);
  color: #fff;
  border: none;
}

[data-theme="chinesered"] .message-item.self .file-message {
  background: var(--primary-color);
  color: #fff;
}

[data-theme="chinesered"] .message-item.self .recalled-message {
  background: rgba(220, 38, 38, 0.8) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

[data-theme="grassgreen"] .message-item.self .message-bubble {
  background: var(--primary-color);
  color: #fff;
  border: none;
}

[data-theme="grassgreen"] .message-item.self .file-message {
  background: var(--primary-color);
  color: #fff;
}

[data-theme="grassgreen"] .message-item.self .recalled-message {
  background: rgba(16, 185, 129, 0.8) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

/* 小程序消息样式 */
.mini-app-message {
  cursor: pointer;
  transition: all 0.2s ease;
  min-width: 250px;
  max-width: 400px;
}

.mini-app-message:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.mini-app-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.mini-app-icon-container {
  flex-shrink: 0;
}

.mini-app-icon {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  object-fit: cover;
}

.mini-app-details {
  flex: 1;
  min-width: 0;
}

.mini-app-name {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mini-app-description {
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 6px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 己方消息中的小程序消息样式 */
.message-item.self .mini-app-message .mini-app-name {
  color: white;
  font-weight: 600;
}

.message-item.self .mini-app-message .mini-app-description {
  color: rgba(255, 255, 255, 0.8);
}

.mini-app-tag {
  display: inline-block;
  font-size: 10px;
  padding: 2px 6px;
  background-color: var(--hover-color);
  border-radius: 4px;
  color: var(--text-secondary);
}

.mini-app-arrow {
  color: var(--text-secondary);
  font-size: 12px;
}

/* 资讯消息样式 */
.news-message {
  cursor: pointer;
  transition: all 0.2s ease;
}

.news-message:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.news-info {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.news-content {
  flex: 1;
  min-width: 0;
}

.news-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 6px;
  line-height: 1.3;
}

.news-summary {
  font-size: 12px;
  color: var(--text-secondary);
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.news-image-container {
  flex-shrink: 0;
  width: 80px;
  height: 60px;
}

.news-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 4px;
}

/* 撤回消息样式 */
.recalled-message {
  background: var(--sidebar-bg) !important;
  color: var(--text-secondary) !important;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: 12px;
}

.message-item.self .recalled-message {
  background: rgba(33, 150, 243, 0.8) !important;
  color: rgba(255, 255, 255, 0.8) !important;
}

/* 引用消息预览样式 */
.quoted-message-preview {
  background: var(--hover-color);
  border-left: 3px solid var(--primary-color);
  padding: 10px 14px;
  margin-bottom: 10px;
  border-radius: 6px;
  font-size: 13px;
  max-width: 100%;
  overflow: hidden;
  transition: all 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  cursor: pointer;
}

/* 被高亮的消息 */
.highlighted-message {
  animation: highlight 2s ease;
}

@keyframes highlight {
  0% {
    background-color: rgba(255, 255, 0, 0.2);
  }
  100% {
    background-color: transparent;
  }
}

.quoted-message-preview:hover {
  background: var(--hover-color);
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
  transform: translateY(-1px);
}

.quoted-message-preview-header {
  font-weight: 600;
  color: var(--text-color);
  margin-bottom: 6px;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
}

.quoted-message-preview-content {
  color: var(--text-color);
  opacity: 0.9;
  line-height: 1.4;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 13px;
  padding-left: 12px;
  position: relative;
}

.quoted-message-preview-content::before {
  content: '';
  position: absolute;
  left: 0;
  top: 2px;
  bottom: 2px;
  width: 2px;
  background: var(--primary-color);
  opacity: 0.3;
  border-radius: 1px;
}

/* 自己发送的消息的引用样式 */
.message-item.self .quoted-message-preview {
  background: rgba(59, 130, 246, 0.15);
  border-left-color: var(--primary-color);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.message-item.self .quoted-message-preview:hover {
  background: rgba(59, 130, 246, 0.2);
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.15);
  transform: translateY(-1px);
}

.message-item.self .quoted-message-preview-header,
.message-item.self .quoted-message-preview-content {
  color: var(--text-color);
}

.message-item.self .quoted-message-preview-content::before {
  background: var(--primary-color);
  opacity: 0.5;
}

.message-meta {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  margin-top: 4px;
}

.message-time {
  font-size: 11px;
  color: var(--text-color);
  opacity: 0;
  transition: opacity 0.3s ease;
}

.message-item:hover .message-time {
  opacity: 0.6;
}

.message-item.self .message-meta {
  justify-content: flex-end;
}

.message-read-status {
  font-size: 10px;
  color: #999;
  opacity: 0.8;
}

.message-read-status.clickable {
  cursor: pointer;
  transition: all 0.2s;
}

.message-read-status.clickable:hover {
  opacity: 0.8;
  transform: scale(1.05);
}

.message-read-status.read {
  color: #4caf50;
  opacity: 1;
}

.message-read-status.failed {
  color: #f56c6c;
  opacity: 1;
  display: flex;
  align-items: center;
  gap: 4px;
}

.message-read-status.failed .retry-btn {
  cursor: pointer;
  transition: all 0.2s;
}

.message-read-status.failed .retry-btn:hover {
  color: #f78989;
  transform: scale(1.1);
}

/* 发送失败的消息气泡样式 */
.message-item.self.failed .message-bubble {
  background-color: #f56c6c;
}

/* 撤回消息样式 */
.message-item.recalled .message-bubble {
  background: var(--sidebar-bg);
  color: #999;
  display: flex;
  align-items: center;
  gap: 6px;
}

.message-item.self.recalled .message-bubble {
  background: var(--sidebar-bg);
  color: #999;
}

.recalled-message span {
  font-size: 12px;
}

.chat-input-area {
  padding: 12px 20px;
  background: var(--sidebar-bg);
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 150px;
  position: relative;
  box-shadow: 0 -2px 10px rgba(0, 0, 0, 0.03);
  border-top: 1px solid var(--border-color);
}

.input-toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.toolbar-btn {
  width: 30px;
  height: 30px;
  border: none;
  background: transparent;
  color: var(--text-color);
  cursor: pointer;
  font-size: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background 0.2s;
}

.toolbar-btn:hover {
  background: var(--hover-color);
}

/* 表情面板容器样式 */
.emoji-panel-container {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 100;
}

.emoji-panel-backdrop {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: transparent;
}

/* 表情面板样式 */
.emoji-panel {
  position: absolute;
  bottom: 100%;
  left: 20px;
  right: 20px;
  margin-bottom: 8px;
  background: var(--sidebar-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 10px;
  max-height: 280px;
  overflow-y: auto;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  z-index: 101;
}

.emoji-category {
  margin-bottom: 12px;
}

.emoji-category-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-color);
  opacity: 0.7;
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.emoji-grid {
  display: grid;
  grid-template-columns: repeat(16, 1fr);
  gap: 2px;
}

.emoji-item {
  font-size: 20px;
  text-align: center;
  cursor: pointer;
  padding: 2px;
  border-radius: 4px;
  transition: background 0.2s ease;
  min-width: 24px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.emoji-item:hover {
  background: var(--hover-color);
}

/* 引用消息样式 */
.quoted-message {
  background: var(--hover-color);
  border-left: 4px solid var(--primary-color);
  padding: 10px;
  margin-bottom: 10px;
  border-radius: 4px;
}

.quoted-message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 5px;
}

.quoted-message-sender {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-color);
}

.quoted-message-remove {
  background: none;
  border: none;
  font-size: 16px;
  cursor: pointer;
  color: var(--text-secondary);
  padding: 0;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: background 0.2s;
}

.quoted-message-remove:hover {
  background: rgba(0, 0, 0, 0.1);
  color: var(--text-color);
}

.quoted-message-content {
  font-size: 14px;
  color: var(--text-color);
  line-height: 1.4;
}

.message-input {
  width: 100%;
  padding: 10px 12px;
  border-radius: 8px;
  font-size: 14px;
  resize: none;
  outline: none;
  font-family: inherit;
  min-height: 120px;
  max-height: 200px;
  overflow-y: hidden;
  box-sizing: border-box;
  background: var(--sidebar-bg);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

.message-input:focus {
  border-color: var(--primary-color);
}

.input-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
}

.input-tip {
  font-size: 12px;
  color: var(--text-color);
  opacity: 0.6;
}

.send-btn {
  padding: 8px 24px;
  background: var(--primary-color);
  color: #fff;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.2s;
}

.send-btn:hover:not(:disabled) {
  opacity: 0.9;
}

.send-btn:disabled {
  background: var(--border-color);
  cursor: not-allowed;
}

.member-count {
  color: var(--primary-color);
  cursor: pointer;
  font-size: 12px;
  margin-left: 4px;
}

.member-count:hover {
  text-decoration: underline;
}

.members-list {
  background: var(--sidebar-bg);
  max-height: 200px;
  overflow-y: auto;
}

.members-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: var(--sidebar-bg);
}

.members-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.close-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  color: var(--text-color);
  cursor: pointer;
  font-size: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  transition: background 0.2s;
}

.close-btn:hover {
  background: var(--hover-color);
}

.members-content {
  padding: 12px 20px;
}

.member-item {
  display: flex;
  align-items: center;
  padding: 8px 0;
  gap: 12px;
}

.member-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
}

.member-name {
  font-size: 14px;
  color: var(--text-color);
}

.members-sidebar {
  width: 180px;
  background: var(--sidebar-bg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: -2px 0 10px rgba(0, 0, 0, 0.05);
  border-left: 1px solid var(--border-color);
  transition: width 0.3s ease;
}

.members-sidebar.collapsed {
  width: 30px;
  border-left: none;
}

.sidebar-header-container {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 8px;
  background: var(--sidebar-bg);
  border-bottom: 1px solid var(--border-color);
}

.members-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.header-content {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-content .toggle-sidebar-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  color: var(--text-color);
  cursor: pointer;
  font-size: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background 0.2s;
}

.header-content .toggle-sidebar-btn:hover {
  background: var(--hover-color);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}



.collapsed-toggle-btn {
  width: 30px;
  height: 30px;
  border: none;
  background: var(--sidebar-bg);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  color: var(--text-color);
  transition: all 0.2s;
}

.collapsed-toggle-btn:hover {
  background: var(--hover-color);
  border-radius: 4px;
}

.members-sidebar .members-header {
  padding: 8px 12px;
  background: var(--sidebar-bg);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.members-sidebar .members-header h3 {
  margin: 0;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-color);
}

.search-toggle-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: var(--text-color);
  transition: all 0.2s;
}

.search-toggle-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
}

.members-search {
  padding: 6px 8px;
  background: var(--sidebar-bg);
}

.member-search-input {
  width: 100%;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  outline: none;
  background: var(--sidebar-bg);
  color: var(--text-color);
  border: 1px solid var(--border-color);
}

.member-search-input:focus {
  border-color: var(--primary-color);
}

.members-sidebar .members-content {
  flex: 1;
  overflow-y: auto;
  padding: 4px 8px;
}

.members-sidebar .member-item {
  display: flex;
  align-items: center;
  gap: 8px;
  border-radius: 6px;
  padding: 6px 10px;
  transition: all 0.2s ease;
  margin-bottom: 1px;
  cursor: pointer;
}

.members-sidebar .member-item:hover {
  background: var(--hover-color);
  transform: translateY(-1px);
}

.members-sidebar .member-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.members-sidebar .member-name {
  font-size: 13px;
}

/* @成员面板 */
.at-members-panel-container {
  position: relative;
  z-index: 1000;
  margin-top: 8px;
}

.at-members-panel-backdrop {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.1);
  z-index: -1;
}

.at-members-panel {
  background: var(--list-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 16px;
  max-height: 300px;
  overflow-y: auto;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
  min-width: 200px;
}

.at-members-header {
  margin-bottom: 12px;
}

.at-members-header h4 {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.at-members-search {
  margin-bottom: 12px;
}

.at-members-search-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--input-bg);
  color: var(--text-color);
  font-size: 14px;
}

.at-members-search-input:focus {
  outline: none;
  border-color: var(--primary-color);
}

.at-members-list {
  max-height: 200px;
  overflow-y: auto;
}

.at-member-item {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
  margin-bottom: 4px;
}

.at-member-item:hover {
  background: var(--hover-color);
}

.at-member-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  margin-right: 8px;
  object-fit: cover;
}

.at-member-name {
  font-size: 14px;
  color: var(--text-color);
}

.empty-at-members {
  padding: 20px;
  text-align: center;
  color: var(--text-secondary);
  font-size: 14px;
}

.members-sidebar .member-name {
  font-size: 13px;
  color: var(--text-color);
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 400;
}

.members-sidebar .member-info {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 4px;
  flex: 1;
  min-width: 0;
}

.members-sidebar .member-role {
  font-size: 14px;
  padding: 1px 4px;
  border-radius: 3px;
  font-weight: 500;
  white-space: nowrap;
}

.members-sidebar .member-role.owner {
  /* background: linear-gradient(135deg, #ffd700, #ffaa00); */
  color: #ffd700;
}

.members-sidebar .member-role.admin {
  /* background: linear-gradient(135deg, #4facfe, #00f2fe); */
  color: #4facfe;
}

/* 搜索相关样式 */
.search-container {
  display: flex;
  align-items: center;
  padding: 8px 0;
  margin-bottom: 12px;
  gap: 8px;
  background: var(--sidebar-bg);
  padding: 8px 12px;
  border-radius: 8px;
}

.search-input {
  flex: 1;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 14px;
  outline: none;
  background: var(--sidebar-bg);
  color: var(--text-color);
}

.search-input:focus {
  border-color: var(--primary-color);
}

.search-btn {
  padding: 6px 16px;
  background: var(--primary-color);
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.2s;
}

.search-btn:hover {
  opacity: 0.9;
}

.close-search-btn {
  width: 24px;
  height: 24px;
  border: none;
  background: transparent;
  color: var(--text-color);
  cursor: pointer;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  transition: background 0.2s;
}

.close-search-btn:hover {
  background: var(--hover-color);
}

.search-status {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: #666;
  font-size: 14px;
}

.search-loading {
  display: flex;
  align-items: center;
  gap: 8px;
}

.search-loading::before {
  content: '';
  width: 16px;
  height: 16px;
  border: 2px solid #e0e0e0;
  border-top: 2px solid #1976d2;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.search-results {
  padding: 16px 20px;
}

.search-results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 8px;
  font-size: 14px;
  color: #333;
}

.clear-search-btn {
  padding: 4px 12px;
  background: transparent;
  color: #1976d2;
  border-radius: 12px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}

.clear-search-btn:hover {
  background: #e3f2fd;
}

.message-sender {
  font-size: 12px;
  color: #666;
  margin-bottom: 4px;
  font-weight: 500;
}

/* 成员上下文菜单样式 */
.context-menu {
  position: fixed;
  background: var(--context-menu-bg);
  border-radius: 6px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.12);
  z-index: 1000;
}

.context-menu-item {
  padding: 8px 16px;
  cursor: pointer;
  font-size: 13px;
  color: var(--text-color);
  transition: background 0.2s;
}

.context-menu-item:hover {
  background: var(--context-menu-hover);
}

.context-menu-item.divider {
  height: 1px;
  background: #e0e0e0;
  padding: 0;
  margin: 4px 0;
  cursor: default;
}

.context-menu-item.divider:hover {
  background: #e0e0e0;
}


.context-menu-icon {
  margin-right: 10px;
  font-size: 16px;
  width: 20px;
  text-align: center;
}

.context-menu-divider {
  height: 1px;
  background-color: #e8e8e8;
  margin: 4px 0;
}

.file-link {
  color: #1890ff;
  text-decoration: none;
  cursor: pointer;
}

.file-link:hover {
  text-decoration: underline;
}

/* 图片消息样式 */
.message-image {
  max-width: 320px;
  max-height: 320px;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.12);
  border: 1px solid var(--border-color);
  background-color: var(--list-bg);
  padding: 8px;
}

.message-image:hover {
  transform: scale(1.02);
  box-shadow: 0 6px 12px rgba(0, 0, 0, 0.15);
}

/* 文件消息样式 */
.file-message {
  background: var(--sidebar-bg);
  border-radius: 12px;
  padding: 14px;
  max-width: 400px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  transition: all 0.2s ease;
}

.file-message:hover {
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12);
}

.file-info {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.file-icon {
  font-size: 24px;
  margin-top: 2px;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--list-bg);
  border-radius: 6px;
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.file-details {
  flex: 1;
  min-width: 0;
}

.file-icon-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.file-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 8px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  line-height: 1.4;
}

.file-size {
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.2;
  white-space: nowrap;
  text-align: center;
}

.file-actions {
  display: flex;
  gap: 8px;
  margin-top: 4px;
}

.file-action-btn {
  padding: 6px 16px;
  font-size: 12px;
  border-radius: 8px;
  background-color: var(--primary-light);
  color: var(--primary-color);
  border: none;
  /* border: 1px solid var(--primary-color); */
  cursor: pointer;
  transition: all 0.3s ease;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}

.file-action-btn:hover {
  background-color: var(--primary-color);
  color: #fff;
  border-color: var(--primary-color);
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.3);
  transform: translateY(-1px);
}

.file-action-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 8px rgba(24, 144, 255, 0.3);
}

/* 自己发送的文件消息 */
.message-item.self .file-message {
  background: var(--primary-color);
  color: #fff;
}

.message-item.self .file-name {
  color: #fff;
}

.message-item.self .file-size {
  color: rgba(255, 255, 255, 0.8);
}

.message-item.self .file-icon {
  background-color: rgba(255, 255, 255, 0.15);
  color: #fff;
  border: 1px solid rgba(255, 255, 255, 0.2);
}

.message-item.self .file-action-btn {
  background-color: rgba(255, 255, 255, 0.1);
  color: #fff;
  border: none;
  /* border: 1px solid rgba(255, 255, 255, 0.2); */
}

.message-item.self .file-action-btn:hover {
  background-color: rgba(255, 255, 255, 0.2);
  color: #fff;
  border-color: rgba(255, 255, 255, 0.3);
  box-shadow: 0 4px 12px rgba(255, 255, 255, 0.2);
  transform: translateY(-1px);
}

.message-item.self .file-action-btn:active {
  transform: translateY(0);
  box-shadow: 0 2px 8px rgba(255, 255, 255, 0.2);
}



/* 图片预览弹窗样式 */
.image-preview-modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  backdrop-filter: blur(8px);
}

.image-preview-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  width: 800px;
  max-width: 90%;
  max-height: 80vh;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2), 0 8px 25px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.image-preview-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  justify-content: flex-end;
  align-items: center;
}

.image-preview-header .close-btn {
  background: #f0f0f0;
  border: 1px solid #ddd;
  color: #333;
  font-size: 18px;
  cursor: pointer;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  z-index: 10;
}

.image-preview-header .close-btn i {
  display: block !important;
  font-size: 16px !important;
  line-height: 1 !important;
}

.image-preview-body {
  padding: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  max-height: 60vh;
}

.image-preview-body img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

/* 分享内容预览弹窗样式 */
.share-preview-modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  backdrop-filter: blur(8px);
  animation: modalFadeIn 0.2s ease-out;
}

.share-preview-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  width: 500px;
  max-width: 90%;
  max-height: 80vh;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.2), 0 8px 25px rgba(0, 0, 0, 0.15);
  overflow: hidden;
  animation: modalSlideIn 0.3s ease-out;
  display: flex;
  flex-direction: column;
}

.share-preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: var(--primary-color);
  color: white;
}

.share-preview-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
}

.share-preview-header .close-btn {
  background: rgba(255, 255, 255, 0.2);
  border: none;
  font-size: 18px;
  cursor: pointer;
  color: white;
  padding: 8px;
  border-radius: 50%;
  transition: all 0.3s;
  display: flex;
  align-items: center;
  justify-content: center;
}

.share-preview-header .close-btn:hover {
  background: rgba(255, 255, 255, 0.3);
  transform: rotate(90deg);
}

.share-preview-body {
  padding: 24px;
  overflow-y: auto;
  flex: 1;
}

.share-preview-title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 16px;
  color: var(--text-color);
  word-break: break-word;
}

.share-preview-content-text {
  font-size: 14px;
  line-height: 1.6;
  color: var(--text-color);
  margin-bottom: 24px;
  white-space: pre-wrap;
  word-break: break-word;
}

.share-preview-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
  color: var(--text-secondary);
  font-size: 12px;
}

.share-preview-type {
  background: var(--primary-light);
  padding: 4px 12px;
  border-radius: 12px;
  color: var(--primary-color);
  font-weight: 500;
}

/* 文件分享内容样式 */
.share-file-content {
  display: flex;
  align-items: center;
  margin-bottom: 24px;
}

.share-file-icon {
  width: 60px;
  height: 60px;
  background: var(--primary-light);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  font-size: 24px;
  color: var(--primary-color);
  flex-shrink: 0;
}

.share-file-info {
  flex: 1;
}

.share-file-size {
  color: var(--text-secondary);
  font-size: 14px;
  margin-top: 4px;
}

/* 文件操作按钮 */
.share-preview-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid var(--border-color);
  background: var(--secondary-color);
}

.share-file-action-btn {
  padding: 8px 20px;
  border: 1px solid var(--primary-color);
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s;
  background: white;
  color: var(--primary-color);
}

.share-file-action-btn:hover {
  background: var(--primary-color);
  color: white;
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

/* 炫酷黑主题 */
[data-theme="dark"] .chat-window {
  background: var(--sidebar-bg) !important;
}

[data-theme="dark"] .chat-main {
  background: var(--secondary-color) !important;
  border-top: none !important;
}

[data-theme="dark"] .chat-header {
  background: var(--sidebar-bg) !important;
  box-shadow: var(--shadow-md) !important;
}

[data-theme="dark"] .message-list {
  background: var(--secondary-color) !important;
  box-shadow: 2px 0 10px rgba(0, 0, 0, 0.1) !important;
}

[data-theme="dark"] .message-bubble {
  background: var(--sidebar-bg) !important;
  color: var(--text-color) !important;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2) !important;
  border: 1px solid rgba(255, 255, 255, 0.1) !important;
}

[data-theme="dark"] .share-message {
  background: rgba(255, 255, 255, 0.05) !important;
  border: 1px solid rgba(255, 255, 255, 0.1) !important;
}

[data-theme="dark"] .mini-app-message {
  background: rgba(255, 255, 255, 0.05) !important;
  border: 1px solid rgba(255, 255, 255, 0.1) !important;
}

[data-theme="dark"] .file-message {
  background: rgba(255, 255, 255, 0.05) !important;
  border: 1px solid rgba(255, 255, 255, 0.1) !important;
}

[data-theme="dark"] .news-message {
  background: rgba(255, 255, 255, 0.05) !important;
  border: 1px solid rgba(255, 255, 255, 0.1) !important;
}

/* 小程序列表面板样式 */
.mini-app-panel-container {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
}

.mini-app-panel-backdrop {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
}

.mini-app-panel {
  position: relative;
  background: var(--sidebar-bg);
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.mini-app-panel-header {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.mini-app-panel-header h4 {
  margin: 0;
  color: var(--text-color);
  font-size: 16px;
}

.mini-app-grid {
  padding: 20px;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 16px;
}

.mini-app-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.mini-app-item-icon {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s;
}

.mini-app-item-icon:hover {
  transform: scale(1.05);
}

.mini-app-item-icon img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.mini-app-item-name {
  font-size: 12px;
  color: var(--text-color);
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100px;
}

.mini-app-item-actions {
  margin-top: 4px;
}

.mini-app-action-btn {
  font-size: 10px;
  padding: 2px 8px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.2s;
}

.mini-app-action-btn:hover {
  background: var(--primary-hover);
}

[data-theme="dark"] .message-input {
  background: var(--secondary-color) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .emoji-panel {
  background: var(--sidebar-bg) !important;
  border: 1px solid var(--border-color) !important;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2) !important;
}

[data-theme="dark"] .members-sidebar {
  background: var(--sidebar-bg) !important;
  box-shadow: -2px 0 10px rgba(0, 0, 0, 0.1) !important;
  border-left: 1px solid var(--border-color) !important;
}

/* 暗黑主题下的引用消息样式 */
[data-theme="dark"] .quoted-message-preview {
  background: rgba(255, 255, 255, 0.05) !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3) !important;
}

[data-theme="dark"] .quoted-message-preview:hover {
  background: rgba(255, 255, 255, 0.1) !important;
  box-shadow: 0 2px 5px rgba(0, 0, 0, 0.4) !important;
}

[data-theme="dark"] .message-item.self .quoted-message-preview {
  background: rgba(59, 130, 246, 0.2) !important;
}

[data-theme="dark"] .message-item.self .quoted-message-preview:hover {
  background: rgba(59, 130, 246, 0.3) !important;
}

/* 暗黑主题下的文件消息样式 */
[data-theme="dark"] .file-message {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .file-icon {
  background-color: var(--secondary-color) !important;
  border: 1px solid var(--border-color) !important;
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .file-size {
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .file-action-btn {
  background-color: var(--secondary-color) !important;
  border: none;
  /* border: 1px solid var(--border-color) !important; */
  color: var(--primary-color) !important;
}

[data-theme="dark"] .file-action-btn:hover {
  background-color: var(--primary-color) !important;
  color: #0a0a0a !important;
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.4) !important;
}

/* 暗黑主题下自己发送的文件消息样式 */
[data-theme="dark"] .message-item.self .file-message {
  background: var(--primary-color) !important;
  color: #0a0a0a !important;
  border: none;
  /* border: 1px solid rgba(24, 144, 255, 0.3) !important; */
}

[data-theme="dark"] .message-item.self .file-name {
  color: #0a0a0a !important;
}

[data-theme="dark"] .message-item.self .file-size {
  color: rgba(10, 10, 10, 0.8) !important;
}

[data-theme="dark"] .message-item.self .file-icon {
  background-color: rgba(10, 10, 10, 0.2) !important;
  color: #0a0a0a !important;
  border: 1px solid rgba(10, 10, 10, 0.3) !important;
}

[data-theme="dark"] .message-item.self .file-action-btn {
  background-color: rgba(10, 10, 10, 0.2) !important;
  color: rgba(255, 255, 255, 0.8) !important;
  border: none;
}

[data-theme="dark"] .message-item.self .file-action-btn:hover {
  background-color: rgba(10, 10, 10, 0.3) !important;
  color: #ffffff !important;
  box-shadow: 0 4px 12px rgba(10, 10, 10, 0.3) !important;
}

[data-theme="dark"] .members-header {
  background: var(--sidebar-bg) !important;
}

[data-theme="dark"] .members-content {
  background: var(--sidebar-bg) !important;
}

[data-theme="dark"] .members-search {
  background: var(--sidebar-bg) !important;
}

[data-theme="dark"] .member-search-input {
  background: var(--secondary-color) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .search-container {
  background: var(--sidebar-bg) !important;
}

[data-theme="dark"] .search-input {
  background: var(--secondary-color) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .search-status {
  color: var(--text-color) !important;
  opacity: 0.7 !important;
}

[data-theme="dark"] .search-results-header {
  color: var(--text-color) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .clear-search-btn {
  color: var(--primary-color) !important;
}

[data-theme="dark"] .clear-search-btn:hover {
  background: var(--hover-color) !important;
}

[data-theme="dark"] .message-sender {
  color: var(--text-color) !important;
  opacity: 0.8 !important;
}

[data-theme="dark"] .context-menu {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .context-menu-item {
  color: var(--text-color) !important;
}

[data-theme="dark"] .context-menu-item:hover {
  background: var(--hover-color) !important;
}

[data-theme="dark"] .context-menu-divider {
  background-color: var(--border-color) !important;
}

[data-theme="dark"] .file-message {
  background-color: var(--sidebar-bg) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .file-name {
  color: var(--text-color) !important;
}

/* [data-theme="dark"] .file-action-btn {
  background-color: var(--sidebar-bg) !important;
  color: var(--primary-color) !important;
  border: none;
  border: 1px solid var(--border-color) !important; 
} */

[data-theme="dark"] .file-action-btn:hover {
  background-color: var(--primary-color) !important;
  color: #0a0a0a !important;
}

/* 头部下拉菜单 */
.header-menu {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 8px;
  background: var(--sidebar-bg);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 1000;
  min-width: 180px;
  overflow: hidden;
}

.menu-item {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  transition: background-color 0.2s;
  font-size: 14px;
}

.menu-item:hover {
  background-color: var(--hover-bg);
}

.menu-item i {
  margin-right: 8px;
  color: var(--text-secondary);
}

/* 模态框样式 */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.modal-content {
  background: var(--sidebar-bg);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  width: 90%;
  max-width: 500px;
  overflow: hidden;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: var(--text-color);
}

.modal-body {
  padding: 20px;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 16px 20px;
  border-top: 1px solid var(--border-color);
  gap: 12px;
}

/* 表单样式 */
.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  margin-bottom: 8px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-color);
}

.form-input {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
  background: var(--content-bg);
  color: var(--text-color);
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.form-textarea {
  width: 100%;
  padding: 12px 16px;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  font-size: 14px;
  background: var(--content-bg);
  color: var(--text-color);
  resize: vertical;
  box-sizing: border-box;
}

.form-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.form-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 0;
}

/* 按钮样式 */
.btn {
  padding: 8px 16px;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;
}

.btn-primary {
  background-color: var(--primary-color);
  color: white;
}

.btn-primary:hover {
  background-color: var(--primary-dark);
}

.btn-secondary {
  background-color: var(--content-bg);
  color: var(--text-color);
  border-color: var(--border-color);
}

.btn-secondary:hover {
  background-color: var(--hover-bg);
}

/* 暗黑主题下的模态框样式 */
[data-theme="dark"] .modal-content {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .form-input,
[data-theme="dark"] .form-textarea {
  background: var(--secondary-color) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .form-input:focus,
[data-theme="dark"] .form-textarea:focus {
  border-color: var(--primary-color) !important;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.2) !important;
}

[data-theme="dark"] .btn-secondary {
  background-color: var(--secondary-color) !important;
  color: var(--text-color) !important;
  border-color: var(--border-color) !important;
}

[data-theme="dark"] .btn-secondary:hover {
  background-color: var(--hover-bg) !important;
}



/* 炫酷黑主题 - 发送按钮样式 */
[data-theme="dark"] .send-btn {
  background: #2d3748 !important;
  color: rgba(229, 231, 235, 1) !important;
  border: 1px solid rgba(229, 231, 235, 0.3) !important;
}

[data-theme="dark"] .send-btn:hover:not(:disabled) {
  background: #374151 !important;
  opacity: 1 !important;
}

[data-theme="dark"] .send-btn:disabled {
  background: #1a1a1a !important;
  color: rgba(229, 231, 235, 0.7) !important;
  border: 1px solid rgba(229, 231, 235, 0.3) !important;
}

[data-theme="dark"] .message-manager-content {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .message-manager-header {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .message-manager-body {
  background: var(--secondary-color) !important;
}

[data-theme="dark"] .message-manager-search .search-input {
  background: var(--sidebar-bg) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1) !important;
}

[data-theme="dark"] .filter-group label {
  color: var(--text-color) !important;
  opacity: 0.8 !important;
}

[data-theme="dark"] .filter-select {
  background: var(--sidebar-bg) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1) !important;
}

[data-theme="dark"] .date-input {
  background: var(--sidebar-bg) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1) !important;
}

[data-theme="dark"] .date-range-separator {
  color: var(--text-color) !important;
  opacity: 0.7 !important;
}

[data-theme="dark"] .message-manager-pagination {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.1) !important;
  border-top: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .pagination-info {
  color: var(--text-color) !important;
  opacity: 0.7 !important;
}

[data-theme="dark"] .pagination-btn {
  background: var(--sidebar-bg) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .pagination-btn:hover:not(:disabled) {
  background: var(--hover-color) !important;
  border-color: var(--primary-color) !important;
  color: var(--primary-color) !important;
}

[data-theme="dark"] .pagination-btn:disabled {
  background: var(--sidebar-bg) !important;
  color: var(--text-color) !important;
  opacity: 0.7 !important;
}

[data-theme="dark"] .pagination-current {
  color: var(--text-color) !important;
}

[data-theme="dark"] .message-manager-list {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .message-manager-item {
  border-bottom: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .message-manager-item:hover {
  background: var(--hover-color) !important;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1) !important;
}

[data-theme="dark"] .message-manager-item-content {
  color: var(--text-color) !important;
}

[data-theme="dark"] .empty-message {
  color: var(--text-color) !important;
  opacity: 0.6 !important;
  background: var(--sidebar-bg) !important;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .message-manager-list::-webkit-scrollbar-track,
[data-theme="dark"] .message-manager-body::-webkit-scrollbar-track {
  background: var(--sidebar-bg) !important;
}

[data-theme="dark"] .message-manager-list::-webkit-scrollbar-thumb,
[data-theme="dark"] .message-manager-body::-webkit-scrollbar-thumb {
  background: var(--border-color) !important;
}

[data-theme="dark"] .message-manager-list::-webkit-scrollbar-thumb:hover,
[data-theme="dark"] .message-manager-body::-webkit-scrollbar-thumb:hover {
  background: var(--text-color) !important;
  opacity: 0.5 !important;
}

/* 确认对话框样式 */
.confirm-dialog-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(5px);
}

.confirm-dialog-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
  width: 90%;
  max-width: 400px;
  overflow: hidden;
}

.confirm-dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: var(--sidebar-bg);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.confirm-dialog-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.confirm-dialog-body {
  padding: 24px;
  background: var(--sidebar-bg);
}

.confirm-dialog-body p {
  margin: 0;
  font-size: 14px;
  color: var(--text-color);
  line-height: 1.5;
  text-align: center;
}

.confirm-dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  background: var(--sidebar-bg);
  border-top: 1px solid var(--border-color);
}

.confirm-dialog-footer button {
  padding: 8px 24px;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
}

.confirm-dialog-footer button.cancel {
  background: var(--border-color);
  color: var(--text-color);
}

.confirm-dialog-footer button.confirm {
  background: var(--primary-color);
  color: #fff;
}

.confirm-dialog-footer button:hover {
  opacity: 0.9;
}

.confirm-dialog-footer button.cancel:hover {
  background: var(--hover-color);
}

.confirm-dialog-footer button.confirm:hover {
  background: var(--primary-color);
  opacity: 0.9;
}

/* Toast样式 */
.toast {
  position: fixed;
  top: 20px;
  right: 20px;
  background: #52c41a;
  color: #fff;
  padding: 12px 20px;
  border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  font-size: 14px;
  font-weight: 500;
  z-index: 9999;
  animation: toastSlideIn 0.3s ease;
}

@keyframes toastSlideIn {
  from {
    opacity: 0;
    transform: translateX(100%);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

/* 炫酷黑主题 - 确认对话框 */
[data-theme="dark"] .confirm-dialog-content {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .confirm-dialog-header {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .confirm-dialog-header h3 {
  color: var(--text-color) !important;
}

/* 小程序相关样式已移至MiniAppHandler.vue组件 */

[data-theme="dark"] .confirm-dialog-body {
  background: var(--secondary-color) !important;
}

[data-theme="dark"] .confirm-dialog-body p {
  color: var(--text-color) !important;
}

[data-theme="dark"] .confirm-dialog-footer {
  background: var(--sidebar-bg) !important;
  border-top: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .confirm-dialog-footer button.cancel {
  background: var(--border-color) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .confirm-dialog-footer button.confirm {
  background: var(--primary-color) !important;
  color: white !important;
  border: 1px solid var(--primary-color) !important;
}

/* 炫酷黑主题 - Toast样式 */
[data-theme="dark"] .toast {
  background: var(--primary-color) !important;
  color: white !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3) !important;
}

/* 炫酷黑主题 - 其他按钮样式 */
[data-theme="dark"] .action-btn {
  color: var(--text-color) !important;
}

[data-theme="dark"] .action-btn.primary {
  color: white !important;
}

[data-theme="dark"] .file-action-btn {
  color: var(--text-color) !important;
}

[data-theme="dark"] .share-action-btn {
  color: var(--text-color) !important;
}

[data-theme="dark"] .toolbar-btn {
  color: var(--text-color) !important;
}

[data-theme="dark"] .send-btn {
  color: white !important;
}

[data-theme="dark"] .screenshot-btn {
  color: var(--text-color) !important;
}

[data-theme="dark"] .call-btn {
  color: white !important;
}

/* 炫酷黑主题 - 引用消息移除按钮 */
[data-theme="dark"] .quoted-message-remove {
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .quoted-message-remove:hover {
  background: rgba(255, 255, 255, 0.1) !important;
  color: var(--text-color) !important;
}

/* 截图预览对话框样式 */
.screenshot-preview-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(5px);
}

.screenshot-preview-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  width: 90%;
  max-width: 800px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
  animation: modalFadeIn 0.3s ease;
}

.screenshot-preview-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background: var(--sidebar-bg);
  border-bottom: 1px solid var(--border-color);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.screenshot-preview-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.screenshot-preview-body {
  flex: 1;
  padding: 24px;
  overflow: auto;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--content-bg);
}

.screenshot-image-container {
  max-width: 100%;
  max-height: 60vh;
  overflow: hidden;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  background: #fff;
  padding: 16px;
}

.screenshot-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 4px;
}

.screenshot-preview-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  background: var(--sidebar-bg);
  border-top: 1px solid var(--border-color);
  box-shadow: 0 -2px 4px rgba(0, 0, 0, 0.05);
}

.screenshot-btn {
  padding: 10px 24px;
  border: 1px solid transparent;
  border-radius: 6px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.retake-btn {
  background: var(--list-bg);
  color: var(--text-color);
  border-color: var(--border-color);
}

.retake-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  transform: translateY(-1px);
}

.cancel-btn {
  background: var(--list-bg);
  color: var(--text-color);
  border-color: var(--border-color);
}

.cancel-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  transform: translateY(-1px);
}

.send-btn {
  background: var(--primary-color);
  color: #fff;
  border-color: var(--primary-color);
  box-shadow: 0 2px 4px rgba(24, 144, 255, 0.2);
}

.send-btn:hover {
  background: #40a9ff;
  border-color: #40a9ff;
  box-shadow: 0 4px 8px rgba(24, 144, 255, 0.3);
  transform: translateY(-1px);
}

/* 炫酷黑主题 - 截图预览对话框 */
[data-theme="dark"] .screenshot-preview-content {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .screenshot-preview-header {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .screenshot-preview-header h3 {
  color: var(--text-color) !important;
}

[data-theme="dark"] .screenshot-preview-body {
  background: var(--secondary-color) !important;
}

[data-theme="dark"] .screenshot-image-container {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .screenshot-preview-footer {
  background: var(--sidebar-bg) !important;
  border-top: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .screenshot-btn.retake-btn {
  background: var(--border-color) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .screenshot-btn.cancel-btn {
  background: var(--border-color) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .screenshot-btn.send-btn {
  background: var(--primary-color) !important;
  color: #0a0a0a !important;
  border: 1px solid var(--primary-color) !important;
}

/* 通话模态框样式 */
.call-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  backdrop-filter: blur(10px);
}

.call-modal-content {
  background: var(--sidebar-bg);
  border-radius: 16px;
  width: 90%;
  max-width: 500px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 15px 30px rgba(0, 0, 0, 0.3);
  animation: modalFadeIn 0.3s ease;
}

.call-modal-header {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 20px;
  background: var(--sidebar-bg);
  border-bottom: 1px solid var(--border-color);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.call-modal-header h3 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-color);
}

.call-modal-body {
  flex: 1;
  padding: 32px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: var(--content-bg);
}

.call-info {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 32px;
}

.call-avatar {
  width: 120px;
  height: 120px;
  border-radius: 50%;
  overflow: hidden;
  margin-bottom: 20px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  border: 3px solid var(--primary-color);
}

.call-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.call-name {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-color);
  margin-bottom: 12px;
}

.call-status {
  font-size: 16px;
  color: var(--text-secondary);
}

.status-ringing {
  animation: pulse 1.5s ease-in-out infinite;
  color: #ff9800;
}

.status-answered {
  color: #4caf50;
}

.status-ended {
  color: #f44336;
}

/* 视频通话区域 */
.video-container {
  width: 100%;
  max-width: 400px;
  margin-top: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.local-video,
.remote-video {
  width: 100%;
  height: 200px;
  border-radius: 8px;
  overflow: hidden;
  background: var(--sidebar-bg);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-color);
}

.local-video {
  width: 120px;
  height: 90px;
  position: absolute;
  top: 20px;
  right: 20px;
  border: 2px solid var(--primary-color);
  z-index: 10;
}

.video-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: var(--sidebar-bg);
  color: var(--text-secondary);
}

.video-placeholder i {
  font-size: 48px;
  margin-bottom: 12px;
  opacity: 0.5;
}

.video-placeholder span {
  font-size: 14px;
  font-weight: 500;
}

.call-modal-footer {
  display: flex;
  justify-content: center;
  gap: 24px;
  padding: 24px;
  background: var(--sidebar-bg);
  border-top: 1px solid var(--border-color);
  box-shadow: 0 -2px 4px rgba(0, 0, 0, 0.05);
}

.call-btn {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  border: none;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.call-btn i {
  font-size: 20px;
  margin-bottom: 4px;
}

.reject-btn {
  background: #f44336;
  color: #fff;
}

.reject-btn:hover {
  background: #d32f2f;
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(244, 67, 54, 0.4);
}

.answer-btn {
  background: #4caf50;
  color: #fff;
}

.answer-btn:hover {
  background: #388e3c;
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(76, 175, 80, 0.4);
}

.end-btn {
  background: #f44336;
  color: #fff;
}

.end-btn:hover {
  background: #d32f2f;
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(244, 67, 54, 0.4);
}

.screen-share-btn {
  background: #ff9800;
  color: #fff;
  margin-left: 10px;
}

.screen-share-btn:hover {
  background: #f57c00;
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(255, 152, 0, 0.4);
}

/* 炫酷黑主题 - 通话模态框 */
[data-theme="dark"] .call-modal-content {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 15px 30px rgba(0, 0, 0, 0.5) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .call-modal-header {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .call-modal-header h3 {
  color: var(--text-color) !important;
}

[data-theme="dark"] .call-modal-body {
  background: var(--secondary-color) !important;
}

[data-theme="dark"] .call-name {
  color: var(--text-color) !important;
}

[data-theme="dark"] .call-status {
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .local-video,
[data-theme="dark"] .remote-video {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="dark"] .video-placeholder {
  background: var(--sidebar-bg) !important;
  color: var(--text-secondary) !important;
}

[data-theme="dark"] .call-modal-footer {
  background: var(--sidebar-bg) !important;
  border-top: 1px solid var(--border-color) !important;
  box-shadow: 0 -2px 4px rgba(0, 0, 0, 0.3) !important;
}
/* 时间分隔线样式 */
.time-divider {
  display: flex;
  justify-content: center;
  align-items: center;
  margin: 15px 0;
  position: relative;
}

.time-divider-text {
  background-color: #f0f0f0;
  color: #999;
  font-size: 12px;
  padding: 4px 12px;
  border-radius: 12px;
  text-align: center;
  font-weight: 400;
}

/* 深色主题下的时间分隔线样式 */
.dark-theme .time-divider-text {
  background-color: #333;
  color: #666;
}

/* 网络蓝主题下的时间分隔线样式 */
.netblue-theme .time-divider-text {
  background-color: #e6f7ff;
  color: #1890ff;
}

/* 高雅紫主题下的时间分隔线样式 */
.elegantpurple-theme .time-divider-text {
  background-color: #f9f0ff;
  color: #722ed1;
}

/* 神圣黄主题下的时间分隔线样式 */
.sacredyellow-theme .time-divider-text {
  background-color: #fffbe6;
  color: #d48806;
}

/* 已读用户列表弹窗样式 */
.read-users-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.read-users-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  width: 360px;
  max-height: 480px;
  display: flex;
  flex-direction: column;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.read-users-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: var(--panel-bg);
  border-bottom: 1px solid var(--border-color);
}

.read-users-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
}

.read-users-body {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.empty-read {
  text-align: center;
  color: var(--text-secondary);
  padding: 24px;
  font-size: 14px;
}

.read-users-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.read-user-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  background: var(--list-bg);
  border-radius: 8px;
  transition: all 0.2s;
}

.read-user-item:hover {
  background: var(--hover-color);
}

.read-user-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  margin-right: 12px;
}

.read-user-info {
  flex: 1;
}

.read-user-name {
  font-size: 14px;
  color: var(--text-color);
  font-weight: 500;
}

.read-icon {
  color: #4caf50;
  font-size: 14px;
}

/* 没有更多消息提示 */
.no-more-messages {
  text-align: center;
  padding: 20px 0;
  color: #999;
  font-size: 14px;
  border-bottom: 1px solid #eee;
  margin-bottom: 10px;
}
</style>
