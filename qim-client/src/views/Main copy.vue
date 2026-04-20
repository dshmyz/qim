<template>
  <div class="im-container">
    <!-- 网络连接状态提示 -->
    <div v-if="showNetworkError" class="network-error">
      <div class="network-error-content">
        <i class="fas fa-exclamation-circle error-icon"></i>
        <div class="error-message">
          <p>{{ networkErrorMsg }}</p>
          <div class="error-actions">
            <button class="retry-btn" @click="reconnect">重新连接</button>
            <button class="login-btn" @click="gotoLogin" v-if="sessionExpired">重新登录</button>
          </div>
        </div>
      </div>
    </div>
    
    <!-- 分享弹窗 -->
    <ShareModal
      :visible="showShareModal"
      :shareType="shareType"
      :users="shareUsers"
      :groups="shareGroups"
      @close="closeShareModal"
      @confirm="handleShareConfirm"
    />
    <!-- 自定义窗口控制栏 -->
    <div class="window-controls">
      <div class="window-controls-left">
        <div class="window-title">QIM</div>
      </div>
      <div class="window-controls-right">
        <button class="window-control-btn minimize-btn" @click="minimizeWindow">—</button>
        <button class="window-control-btn maximize-btn" @click="maximizeWindow">☐</button>
        <button class="window-control-btn close-btn" @click="closeWindow">×</button>
      </div>
    </div>
    
    <!-- 主内容区域 -->
    <div class="main-content-area">
      <!-- 左侧垂直选项栏 -->
      <SideOptions
        :activeOption="activeOption"
        @update:activeOption="activeOption = $event"
        @showMoreMenu="showMoreMenu"
        @showThemeMenu="showThemeMenu"
        @showSettingsMenu="showSettingsMenu"
      />
      
      <!-- 主内容区域 -->
      <div class="main-content">
      <!-- 侧边栏（包含我的账号和搜索） -->
      <Sidebar
        :currentUser="currentUser"
        :activeOption="activeOption"
        :searchQuery="searchQuery"
        :conversations="conversations"
        :currentConversationId="currentConversationId"
        :unreadNotificationCount="unreadNotificationCount"
        :serverUrl="serverUrl"
        :orgStructure="orgStructure"
        :selectedGroup="selectedGroup"
        :selectedChannel="selectedChannel"
        :appCategories="appCategories"
        @update:searchQuery="searchQuery = $event"
        @showUserProfile="showUserProfile = true"
        @showNotification="showNotificationCenter"
        @showActionMenu="showActionMenu"
        @selectConversation="handleConversationSelect"
        @conversationContextMenu="(event, conversation) => showContextMenu(event, conversation)"
        @selectUser="handleUserClick"
        @startPrivateChat="startPrivateChat"
        @userContextMenu="showUserContextMenu"
        @selectGroup="(group) => { console.log('Main - Selected group:', group); selectedGroup = group }"
        @enterGroup="handleConversationSelect"
        @inviteMembers="handleInviteMembers"
        @groupContextMenu="showGroupContextMenu"
        @selectChannel="handleChannelSelect"
        @openApp="openApp"
        @openExternalApp="openExternalApp"
        @resetApp="() => { selectedAppId = '' }"
      />
      
      <!-- 聊天窗口 -->
      <ChatWindow
        v-if="currentConversation && activeOption === 'recent'"
        :conversation="currentConversation"
        :messages="messages"
        :getReadUsers="getMessageReadUsers"
        :currentUser="currentUser.value"
        :hasMoreMessages="hasMoreMessages"
        @send="handleSendMessage"
        @recall="handleRecallMessage"
        @inviteMembers="handleInviteMembers"
        @switchConversation="handleSwitchConversation"
        @switch-app="handleSwitchApp"
        @loadMore="handleLoadMore"
        @retry-send="handleRetrySendMessage"
      />
      <div v-else-if="activeOption === 'recent'" class="right-content">
        <div class="right-content-header">
          <h2>{{ getPageTitle() }}</h2>
        </div>
        <div class="right-content-body">
          <div class="empty-content">
            <div class="empty-icon"><i class="fas fa-comments"></i></div>
            <p>选择一个会话开始聊天</p>
          </div>
        </div>
      </div>
      
      <!-- 频道页面的右侧内容 -->
      <div v-else-if="activeOption === 'channels'" class="right-content">
        <div v-if="selectedChannel" class="channel-detail-content">
          <div class="right-content-header">
            <h2>{{ selectedChannel.name }}</h2>
          </div>
          <div class="channel-detail-info">
            <div class="channel-header-info">
              <img :src="selectedChannel.avatar || 'https://api.dicebear.com/7.x/avataaars/svg?seed=channel'" :alt="selectedChannel.name" class="channel-header-avatar" />
              <div class="channel-header-text">
                <p class="channel-description">{{ selectedChannel.description }}</p>
                <div class="channel-meta">
                  <span>创建者: {{ selectedChannel.creator?.name || '未知' }}</span>
                  <span v-if="selectedChannel.created_at">创建时间: {{ formatTime(selectedChannel.created_at) }}</span>
                </div>
              </div>
            </div>
            <div class="channel-header-actions">
              <button 
                v-if="selectedChannel.is_subscribed" 
                class="btn btn-secondary subscribed" 
                @click="unsubscribeChannel(selectedChannel)"
              >
                <i class="fas fa-check"></i> 已订阅
              </button>
              <button 
                v-else 
                class="btn btn-primary" 
                @click="subscribeChannel(selectedChannel)"
              >
                <i class="fas fa-plus"></i> 订阅
              </button>
            </div>
          </div>
          
          <div class="channel-messages">
            <h3>最新消息</h3>
            <div v-if="!selectedChannel.messages || selectedChannel.messages.length === 0" class="empty-messages">
              <i class="fas fa-comment-alt"></i>
              <p>暂无消息</p>
            </div>
            <div v-else class="message-list">
              <div 
                v-for="message in selectedChannel.messages" 
                :key="message.id" 
                class="message-item"
              >
                <img :src="message.sender?.avatar || 'https://api.dicebear.com/7.x/avataaars/svg?seed=user'" :alt="message.sender?.name" class="message-avatar" />
                <div class="message-content">
                  <div class="message-header">
                    <span class="message-sender">{{ message.sender?.name || '未知' }}</span>
                    <span class="message-time">{{ formatTime(message.created_at) }}</span>
                  </div>
                  <div class="message-text">{{ message.content }}</div>
                </div>
              </div>
            </div>
          </div>
          
          <div v-if="isChannelCreator(selectedChannel)" class="message-input-area">
            <textarea 
              v-model="channelMessage" 
              placeholder="输入消息..." 
              rows="2"
              class="message-textarea"
            ></textarea>
            <button class="btn btn-primary send-btn" @click="sendChannelMessage(selectedChannel)" :disabled="!channelMessage.trim()">发送</button>
          </div>
        </div>
        <div v-else class="channels-content">
          <div class="channels-empty-state">
            <div class="empty-icon"><i class="fas fa-bullhorn"></i></div>
            <p>选择一个频道查看详情</p>
          </div>
        </div>
      </div>
      
      <!-- 组织架构用户信息 -->
      <div v-else-if="activeOption === 'org' && selectedUser" class="right-content">
        <div class="right-content-header">
          <h2>用户资料</h2>
        </div>
        <div class="user-profile-container">
          <!-- 顶部背景 -->
          <div class="user-profile-header-bg"></div>
          
          <!-- 用户信息卡片 -->
          <div class="user-profile-card">
            <!-- 头像和基本信息 -->
            <div class="user-profile-avatar-section">
              <div class="user-avatar-container">
                <img
                  :src="(selectedUser?.avatar && selectedUser.avatar.startsWith('http')) ? selectedUser.avatar : (selectedUser?.avatar ? serverUrl + selectedUser.avatar : 'https://api.dicebear.com/7.x/avataaars/svg?seed=user')"
                  :alt="selectedUser?.name"
                  class="user-avatar"
                />
                <div class="online-status-indicator"></div>
              </div>
              <div class="user-basic-info">
                <h2 class="user-full-name">{{ selectedUser.name }}</h2>
                <p class="user-department">{{ selectedUser.department || '暂无部门' }}</p>
                <p class="user-position">{{ selectedUser.position || '暂无职位' }}</p>
              </div>
            </div>
            
            <!-- 信息分组 -->
            <div class="user-info-sections">
              <!-- 基本信息 -->
              <div class="info-section">
                <div class="section-title">
                  <i class="fas fa-user-circle"></i>
                  <h3>基本信息</h3>
                </div>
                <div class="info-grid">
                  <div class="info-item">
                    <span class="info-label">姓名</span>
                    <span class="info-value">{{ selectedUser.name }}</span>
                  </div>
                  <div class="info-item">
                    <span class="info-label">账号</span>
                    <span class="info-value">{{ selectedUser.username || '暂无' }}</span>
                  </div>
                  <div class="info-item">
                    <span class="info-label">邮箱</span>
                    <span class="info-value">{{ selectedUser.email || '暂无' }}</span>
                  </div>
                  <div class="info-item">
                    <span class="info-label">手机</span>
                    <span class="info-value">{{ selectedUser.mobile || '暂无' }}</span>
                  </div>
                </div>
              </div>
              
              <!-- 工作信息 -->
              <div class="info-section">
                <div class="section-title">
                  <i class="fas fa-briefcase"></i>
                  <h3>工作信息</h3>
                </div>
                <div class="info-grid">
                  <div class="info-item">
                    <span class="info-label">部门</span>
                    <span class="info-value">{{ selectedUser.department || '暂无' }}</span>
                  </div>
                  <div class="info-item">
                    <span class="info-label">IP</span>
                    <span class="info-value">{{ selectedUser.ip || '暂无' }}</span>
                  </div>
                </div>
              </div>
            </div>
            
            <!-- 操作按钮 -->
            <div class="user-action-buttons">
              <button class="action-btn primary" @click="startPrivateChat(selectedUser)">
                <i class="fas fa-comment"></i>
                <span>发起私聊</span>
              </button>
              <button class="action-btn secondary" @click="showUserProfile = true">
                <i class="fas fa-id-card"></i>
                <span>详细资料</span>
              </button>
            </div>
          </div>
        </div>
      </div>
      
      <!-- 其他页面的右侧内容 -->
      <div v-else-if="activeOption === 'apps' && !selectedAppId" class="right-content">
        <div class="right-content-header">
          <h2>{{ getPageTitle() }}</h2>
        </div>
        <div class="apps-content">
          <!-- 最近使用的应用 -->
          <div class="recent-apps-section">
            <div class="section-header">
              <h3>最近使用</h3>
            </div>
            <div class="recent-apps-grid">
              <div
                v-for="app in recentApps"
                :key="app.id"
                class="recent-app-grid-item"
                @click="openApp(app.id)"
              >
                <div class="recent-app-grid-icon"><i :class="app.icon"></i></div>
                <span class="recent-app-grid-name">{{ app.name }}</span>
              </div>
              <div v-if="recentApps.length === 0" class="empty-recent-apps">
                <p>暂无最近使用的应用</p>
              </div>
            </div>
          </div>
          
          <!-- 所有应用 -->
          <div class="all-apps-section">
            <div class="section-header">
              <h3>所有应用</h3>
            </div>
            <div class="apps-grid">
              <div 
                v-for="app in allApps" 
                :key="app.id"
                class="app-item"
                @click="app.url ? openApp(app.id) : openApp(app.id)"
              >
                <div class="app-icon"><i :class="app.icon"></i></div>
                <div class="app-name">{{ app.name }}</div>
              </div>
              <div v-if="allApps.length === 0" class="empty-all-apps">
                <p>暂无应用</p>
              </div>
            </div>
          </div>
        </div>
      </div>
      
      <!-- 文件管理应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === '3'" class="right-content">
        <FileManagementApp @back="backToAppList" />
      </div>
      
      <!-- 笔记应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === '7'" class="right-content">
        <NotesApp @back="backToAppList" />
      </div>
      
      <!-- 任务管理应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === '5'" class="right-content">
        <TaskManagementApp @back="backToAppList" />
      </div>
      <!-- 统计报表应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === '1'" class="right-content">
        <StatisticsApp @back="backToAppList" />
      </div>
      
      <!-- 日历应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === '2'" class="right-content">
        <CalendarApp @back="backToAppList" />
      </div>
      

      
      <!-- 便签应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === '6'" class="right-content">
        <StickyNotesApp @back="backToAppList" />
      </div>
      
      <!-- 用户创建的应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === 'user-app' && currentUserApp" class="right-content">
        <div class="right-content-header">
          <div class="header-left">
            <button class="back-button" @click="backToAppList">
              <i class="fas fa-arrow-left"></i>
            </button>
            <h2>{{ currentUserApp.name }}</h2>
          </div>
        </div>
        <div class="user-app-content">
          <div v-if="currentUserApp.url" class="user-app-iframe-container">
            <iframe 
              :src="currentUserApp.url" 
              class="user-app-iframe"
              frameborder="0"
              allowfullscreen
            ></iframe>
          </div>
          <div v-else class="empty-user-app">
            <div class="empty-icon"><i class="fas fa-link"></i></div>
            <p>该应用没有配置URL</p>
            <p class="empty-hint">请在应用管理中编辑应用，添加URL地址</p>
          </div>
        </div>
      </div>
      
      <!-- 应用管理 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === 'app-management'" class="right-content">
        <AppManagementApp @back="backToAppList" />
      </div>
      
      <!-- AI 助手 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === 'ai-assistant'" class="right-content">
        <AIAssistantApp @back="backToAppList" />
      </div>
      
      <!-- 短链接管理应用 -->
      <div v-else-if="activeOption === 'apps' && selectedAppId === 'short-link'" class="right-content">
        <ShortLinkManager @back="backToAppList" />
      </div>
      
      <!-- 群聊详情 -->
      <div v-else-if="activeOption === 'groups' && selectedGroup" class="right-content">
        <div class="right-content-header">
          <h2></h2>

          <!-- <h2>{{ selectedGroup.name }}</h2> -->
        </div>
        <GroupDetail
          :group="selectedGroup"
          @enter="handleConversationSelect($event)"
          @invite="handleInviteMembers($event)"
          @editAnnouncement="editAnnouncement"
          @showMemberContextMenu="(event, member) => showMemberContextMenu(event, member)"
        />
      </div>
      
      <div v-else class="right-content">
        <div class="right-content-header">
          <h2>{{ getPageTitle() }}</h2>
        </div>
        <div class="right-content-body">
          <p>选择左侧的{{ getPageTitle() }}查看详情</p>
        </div>
      </div>
      <!-- 隐藏的便签应用实例，用于处理添加到笔记事件 -->
      <div style="display: none">
        <StickyNotesApp />
      </div>


      


    </div>
    </div>
    
    <!-- 右键菜单 -->
    <div
      v-if="showMenu && selectedConversation"
      class="context-menu"
      :style="{ left: menuPosition.x + 'px', top: menuPosition.y + 'px' }"
      @click.stop
    >
      <div class="context-menu-item" @click="handlePin(selectedConversation)">
        {{ selectedConversation.pinned ? '取消置顶' : '置顶' }}
      </div>
      <div class="context-menu-item" @click="handleMute(selectedConversation)">
        {{ selectedConversation.muted ? '取消免打扰' : '免打扰' }}
      </div>
      <div v-if="selectedConversation.type === 'group'" class="context-menu-item" @click="handleExitGroup(selectedConversation)">
        退出群聊
      </div>
      <div class="context-menu-item divider"></div>
      <div class="context-menu-item" @click="handleRemove(selectedConversation)">
        移除会话
      </div>
    </div>
    
    <!-- 用户信息弹窗 -->
    <UserProfile 
      v-if="showUserProfile && selectedUser" 
      :visible="showUserProfile && selectedUser" 
      :user="selectedUser" 
      @close="closeUserProfile"
      @send-private-message="startPrivateChat"
    />
    
    <!-- 个人资料弹窗 -->
    <div v-if="showUserProfile && !selectedUser" class="user-profile-modal" @click="closeUserProfile">
      <div class="user-profile-content" @click.stop>
        <div class="user-profile-header">
          <h3>个人信息</h3>
          <button class="close-btn" @click="closeUserProfile">×</button>
        </div>
        <div class="user-profile-body">
          <div class="profile-avatar">
            <img 
              :src="(currentUser?.avatar && currentUser.avatar.startsWith('http')) ? currentUser.avatar : (currentUser?.avatar ? serverUrl + currentUser.avatar : 'https://api.dicebear.com/7.x/avataaars/svg?seed=me')" 
              :alt="currentUser?.username || 'avatar'"
              @click="triggerAvatarInput"
              class="avatar-clickable"
            />
            <input type="file" accept="image/*" class="avatar-input" @change="handleAvatarChange" />
          </div>
          <div class="profile-info">
            <div class="info-item">
              <label>昵称</label>
              <input type="text" v-model="userProfile.nickname" class="profile-input" />
            </div>
            <div class="info-item">
              <label>账号</label>
              <span class="profile-value">{{ userProfile.username }}</span>
            </div>
            <div class="info-item">
              <label>签名</label>
              <textarea v-model="userProfile.signature" class="profile-textarea" placeholder="输入个人签名"></textarea>
            </div>
            <div class="info-item">
              <label>部门</label>
              <span class="profile-value">无</span>
            </div>
            <div class="info-item">
              <label>ID</label>
              <span class="profile-value">{{ userProfile.id }}</span>
            </div>
            <div class="info-item">
              <label>加入时间</label>
              <span class="profile-value">{{ userProfile.joinDate }}</span>
            </div>
          </div>
        </div>
        <div class="user-profile-footer">
          <button class="cancel-btn" @click="closeUserProfile">关闭</button>
          <button class="save-btn" @click="saveUserProfile">保存</button>
        </div>
      </div>
    </div>
    
    <!-- 动作菜单 -->
    <div v-if="showActionMenuFlag" class="action-menu" :style="{ left: actionMenuPosition.x + 'px', top: actionMenuPosition.y + 'px' }">
      <div class="action-menu-item" @click="openCreateGroupModal">
        <span class="action-menu-icon"><i class="fas fa-user-friends"></i></span>
        <span>创建群聊</span>
      </div>
      <div class="action-menu-item" @click="createDiscussionGroup">
        <span class="action-menu-icon"><i class="fas fa-comments"></i></span>
        <span>创建讨论组</span>
      </div>
      <div class="action-menu-item" @click="createChannel">
        <span class="action-menu-icon"><i class="fas fa-bullhorn"></i></span>
        <span>创建频道</span>
      </div>
      <div v-if="currentUser?.isAdmin" class="action-menu-item" @click="openSystemMessageModal">
        <span class="action-menu-icon"><i class="fas fa-broadcast-tower"></i></span>
        <span>发布系统消息</span>
      </div>
    </div>
    
    <!-- 用户右键菜单 -->
    <div v-if="showUserContextMenuFlag" class="user-context-menu" :style="{ left: userContextMenuPosition.x + 'px', top: userContextMenuPosition.y + 'px' }" @click.stop>
      <div class="user-context-menu-item" @click="viewUserProfile">
        <span class="user-context-menu-icon"><i class="fas fa-user"></i></span>
        <span>查看资料</span>
      </div>
      <div class="user-context-menu-item" @click="startPrivateChat(selectedEmployee)">
        <span class="user-context-menu-icon"><i class="fas fa-comment"></i></span>
        <span>发起私聊</span>
      </div>
    </div>
    
    <!-- 创建群聊/讨论组弹窗 -->
    <CreateGroupModal 
      :visible="showCreateConversationModal"
      :type="createConversationType"
      :title="createConversationTitle"
      :members="allEmployees"
      @close="closeCreateConversationModal"
      @created="handleConversationCreated"
    />
    
    <!-- 系统消息发布模态框 -->
    <div v-if="showSystemMessageModal" class="user-profile-modal" @click="closeSystemMessageModal">
      <div class="user-profile-content" @click.stop>
        <div class="user-profile-header">
          <h3>发布系统消息</h3>
          <button class="close-btn" @click="closeSystemMessageModal">×</button>
        </div>
        <div class="user-profile-body">
          <div class="profile-info">
            <div class="info-item">
              <label>消息标题</label>
              <input type="text" v-model="systemMessage.title" class="profile-input" placeholder="请输入消息标题" />
            </div>
            <div class="info-item">
              <label>消息内容</label>
              <textarea v-model="systemMessage.content" class="profile-textarea" placeholder="请输入消息内容" rows="4"></textarea>
            </div>
            <div class="info-item">
              <label>发送范围</label>
              <select v-model="systemMessage.target" class="profile-input">
                <option value="all">所有用户</option>
                <option value="group">指定群聊</option>
                <option value="user">指定用户</option>
              </select>
            </div>
            <div v-if="systemMessage.target === 'group'" class="info-item">
              <label>选择群聊</label>
              <select v-model="systemMessage.groupId" class="profile-input">
                <option v-for="group in conversations.filter(c => c.type === 'group')" :key="group.id" :value="group.id">{{ group.name }}</option>
              </select>
            </div>
            <div v-if="systemMessage.target === 'user'" class="info-item">
              <label>选择用户</label>
              <select v-model="systemMessage.userId" class="profile-input">
                <option v-for="employee in allEmployees" :key="employee.id" :value="employee.id">{{ employee.name }}</option>
              </select>
            </div>
          </div>
        </div>
        <div class="user-profile-footer">
          <button class="cancel-btn" @click="closeSystemMessageModal">取消</button>
          <button class="save-btn" @click="sendSystemMessage" :disabled="!systemMessage.title || !systemMessage.content">发布</button>
        </div>
      </div>
    </div>
    
    <!-- 成员上下文菜单 -->
    <div v-if="showMemberContextMenuFlag" class="context-menu" :style="{ left: memberContextMenuPosition.x + 'px', top: memberContextMenuPosition.y + 'px' }">
      <div class="context-menu-item" @click="removeMemberFromGroup">
        <span class="context-menu-icon"><i class="fas fa-trash"></i></span>
        <span>移除群聊</span>
      </div>
      <div class="context-menu-item" @click="viewMemberInfo">
        <span class="context-menu-icon"><i class="fas fa-user"></i></span>
        <span>查看资料</span>
      </div>
      <div class="context-menu-item" @click="setAsAdmin">
        <span class="context-menu-icon"><i class="fas fa-star"></i></span>
        <span>设为管理员</span>
      </div>
    </div>
    
    <!-- 群成员模态框 -->
    <div v-if="showGroupMembersModal" class="add-members-modal" @click="closeGroupMembersModal">
      <div class="add-members-content" @click.stop>
        <div class="add-members-header">
          <h3>群成员列表</h3>
          <button class="close-btn" @click="closeGroupMembersModal">×</button>
        </div>
        <div class="add-members-body">
          <div class="group-info">
            <div class="group-avatar">
              <img :src="selectedGroup?.avatar || 'https://api.dicebear.com/7.x/identicon/svg?seed=group'" :alt="selectedGroup?.name" />
            </div>
            <div class="group-details">
              <div class="group-name">{{ selectedGroup?.name }}</div>
              <div class="group-members-count">{{ groupMembers.length }} 位成员</div>
            </div>
          </div>
          
          <div class="members-section">
            <div class="section-header">
              <span>成员列表</span>
            </div>
            <div class="members-list">
              <div 
                v-for="member in groupMembers" 
                :key="member.id" 
                class="member-item"
              >
                <div class="member-avatar">
                  <img :src="member.avatar" :alt="member.name" />
                </div>
                <div class="member-info">
                  <div class="member-name">{{ member.name }}</div>
                  <div class="member-position">{{ member.position || '无职位信息' }}</div>
                </div>
                <div class="member-actions">
                  <button 
                    class="remove-member-btn" 
                    @click="removeMember(member)"
                    v-if="member.id !== currentUser.id"
                  >
                    <i class="fas fa-trash-alt"></i>
                  </button>
                </div>
              </div>
              <div v-if="groupMembers.length === 0" class="empty-state">
                <p>暂无成员</p>
              </div>
            </div>
          </div>
        </div>
        <div class="add-members-footer">
          <button class="cancel-btn" @click="closeGroupMembersModal">关闭</button>
        </div>
      </div>
    </div>

    <!-- 群资料模态框 -->
    <div v-if="showGroupInfoModal" class="add-members-modal" @click="closeGroupInfoModal">
      <div class="add-members-content" @click.stop>
        <div class="add-members-header">
          <h3>群聊资料</h3>
          <button class="close-btn" @click="closeGroupInfoModal">×</button>
        </div>
        <div class="add-members-body">
          <div class="group-info">
            <div class="group-avatar" style="width: 80px; height: 80px;">
              <img :src="selectedGroup?.avatar || 'https://api.dicebear.com/7.x/identicon/svg?seed=group'" :alt="selectedGroup?.name" style="width: 100%; height: 100%;" />
            </div>
            <div class="group-details">
              <div class="group-name" style="font-size: 20px;">{{ selectedGroup?.name }}</div>
              <div class="group-members-count">{{ selectedGroup?.members?.length || 0 }} 位成员</div>
            </div>
          </div>
          
          <div class="group-details-section">
            <div class="detail-item">
              <div class="detail-label">群聊ID</div>
              <div class="detail-value">{{ selectedGroup?.id }}</div>
            </div>
            <div class="detail-item">
              <div class="detail-label">创建时间</div>
              <div class="detail-value">{{ selectedGroup?.createdAt ? formatTime(selectedGroup.createdAt) : '未知' }}</div>
            </div>
            <div class="detail-item">
              <div class="detail-label">群聊类型</div>
              <div class="detail-value">群聊</div>
            </div>
          </div>
        </div>
        <div class="add-members-footer">
          <button class="cancel-btn" @click="closeGroupInfoModal">关闭</button>
        </div>
      </div>
    </div>

    <!-- 添加成员模态框 -->
    <div v-if="showAddMembersModal" class="add-members-modal" @click="closeAddMembersModal">
      <div class="add-members-content" @click.stop>
        <div class="add-members-header">
          <h3>邀请成员加入群聊</h3>
          <button class="close-btn" @click="closeAddMembersModal">×</button>
        </div>
        <div class="add-members-body">
          <div class="group-info">
            <div class="group-avatar">
              <img :src="selectedGroup?.avatar || 'https://api.dicebear.com/7.x/identicon/svg?seed=group'" :alt="selectedGroup?.name" />
            </div>
            <div class="group-details">
              <div class="group-name">{{ selectedGroup?.name }}</div>
              <div class="group-members-count">{{ selectedGroup?.members?.length || 0 }} 位成员</div>
            </div>
          </div>
          
          <div class="search-section">
            <div class="search-box">
              <input 
                type="text" 
                v-model="addMembersSearchQuery" 
                placeholder="搜索成员..." 
                class="search-input"
              />
            </div>
          </div>
          
          <div class="members-section">
            <div class="section-header">
              <span>选择成员</span>
              <span class="selected-count">{{ selectedAddMembers.length }} 已选择</span>
            </div>
            <div class="members-list">
              <div 
                v-for="employee in filteredAddMembersEmployees" 
                :key="employee.id" 
                class="member-item"
                :class="{ selected: selectedAddMembers.some(m => m.id === employee.id) }"
                @click="toggleAddMember(employee)"
              >
                <div class="member-avatar">
                  <img :src="employee.avatar" :alt="employee.name" />
                </div>
                <div class="member-info">
                  <div class="member-name">{{ employee.name }}</div>
                  <div class="member-position">{{ employee.position || '无职位信息' }}</div>
                </div>
                <div class="member-checkbox">
                  <input 
                    type="checkbox" 
                    v-model="selectedAddMembers" 
                    :value="employee"
                    class="checkbox"
                  />
                </div>
              </div>
              <div v-if="filteredAddMembersEmployees.length === 0" class="empty-state">
                <p>没有找到匹配的成员</p>
              </div>
            </div>
          </div>
        </div>
        <div class="add-members-footer">
          <button class="cancel-btn" @click="closeAddMembersModal">取消</button>
          <button 
            class="confirm-btn" 
            @click="confirmAddMembers"
            :disabled="selectedAddMembers.length === 0"
          >
            邀请 ({{ selectedAddMembers.length }})
          </button>
        </div>
      </div>
    </div>
    
    <!-- 群聊上下文菜单 -->
    <div v-if="showGroupContextMenuFlag" class="context-menu" :style="{ left: groupContextMenuPosition.x + 'px', top: groupContextMenuPosition.y + 'px' }">
      <div class="context-menu-item" @click="viewGroupMembers">
        <span class="context-menu-icon"><i class="fas fa-user-friends"></i></span>
        <span>查看群成员</span>
      </div>
      <div class="context-menu-item" @click="viewGroupInfo">
        <span class="context-menu-icon"><i class="fas fa-info-circle"></i></span>
        <span>查看群资料</span>
      </div>
      <div class="context-menu-item" @click="addMembersToGroup">
        <span class="context-menu-icon"><i class="fas fa-plus"></i></span>
        <span>添加成员</span>
      </div>
      <div v-if="isGroupOwner(selectedGroup)" class="context-menu-item" @click="editAnnouncement">
        <span class="context-menu-icon"><i class="fas fa-bullhorn"></i></span>
        <span>编辑群公告</span>
      </div>
      <div v-if="isGroupOwner(selectedGroup)" class="context-menu-item" @click="dissolveGroup">
        <span class="context-menu-icon"><i class="fas fa-trash-alt"></i></span>
        <span>解散群聊</span>
      </div>
      <div class="context-menu-divider"></div>
      <div class="context-menu-item" @click="exitGroup">
        <span class="context-menu-icon"><i class="fas fa-sign-out-alt"></i></span>
        <span>退出群聊</span>
      </div>
    </div>
    
    <!-- 编辑群公告模态框 -->
    <div v-if="showEditAnnouncementModal" class="add-members-modal" @click="closeEditAnnouncementModal">
      <div class="add-members-content" @click.stop>
        <div class="add-members-header">
          <h3>编辑群公告</h3>
          <button class="close-btn" @click="closeEditAnnouncementModal">×</button>
        </div>
        <div class="add-members-body">
          <div class="group-info">
            <div class="group-avatar">
              <img :src="selectedGroup?.avatar || 'https://api.dicebear.com/7.x/identicon/svg?seed=group'" :alt="selectedGroup?.name" />
            </div>
            <div class="group-details">
              <div class="group-name">{{ selectedGroup?.name }}</div>
            </div>
          </div>
          
          <div class="announcement-edit-section">
            <textarea 
              v-model="editAnnouncementContent" 
              placeholder="输入群公告内容..." 
              class="announcement-textarea"
              rows="6"
            ></textarea>
            <p class="announcement-tip">群公告将对所有群成员可见</p>
          </div>
        </div>
        <div class="add-members-footer">
          <button class="cancel-btn" @click="closeEditAnnouncementModal">取消</button>
          <button class="confirm-btn" @click="saveAnnouncement">保存</button>
        </div>
      </div>
    </div>
    
    <!-- 设置菜单 -->
    <div v-if="showSettingsMenuFlag" class="context-menu" :style="{ left: settingsMenuPosition.x + 'px', top: settingsMenuPosition.y + 'px' }">
      <div class="context-menu-item" @click="aboutApp">
        <span class="context-menu-icon"><i class="fas fa-info-circle"></i></span>
        <span>关于</span>
      </div>
      <div class="context-menu-item" @click="checkForUpdates">
        <span class="context-menu-icon"><i class="fas fa-sync"></i></span>
        <span>检查更新</span>
      </div>
      <div class="context-menu-item" @click="openSettings">
        <span class="context-menu-icon"><i class="fas fa-sliders"></i></span>
        <span>设置</span>
      </div>
      <div class="context-menu-divider"></div>
      <div class="context-menu-item" @click="logout">
        <span class="context-menu-icon"><i class="fas fa-sign-out-alt"></i></span>
        <span>退出登录</span>
      </div>
    </div>
    
    <!-- 主题菜单 -->
    <div v-if="showThemeMenuFlag" class="context-menu" :style="{ left: themeMenuPosition.x + 'px', top: themeMenuPosition.y + 'px' }">
      <div class="context-menu-item" @click="setTheme('modern-light')">
        <span class="context-menu-icon theme-icon modern-light-theme"></span>
        <span>现代浅色</span>
      </div>
      <div class="context-menu-item" @click="setTheme('elegant-dark')">
        <span class="context-menu-icon theme-icon elegant-dark-theme"></span>
        <span>优雅深色</span>
      </div>
      <div class="context-menu-item" @click="setTheme('ocean-blue')">
        <span class="context-menu-icon theme-icon ocean-blue-theme"></span>
        <span>海洋蓝</span>
      </div>
      <div class="context-menu-item" @click="setTheme('elegant-purple')">
        <span class="context-menu-icon theme-icon elegant-purple-theme"></span>
        <span>高雅紫</span>
      </div>
      <div class="context-menu-item" @click="setTheme('warm-amber')">
        <span class="context-menu-icon theme-icon warm-amber-theme"></span>
        <span>温暖琥珀</span>
      </div>
      <div class="context-menu-item" @click="setTheme('crimson-red')">
        <span class="context-menu-icon theme-icon crimson-red-theme"></span>
        <span>绯红</span>
      </div>
      <div class="context-menu-item" @click="setTheme('emerald-green')">
        <span class="context-menu-icon theme-icon emerald-green-theme"></span>
        <span>翡翠绿</span>
      </div>
    </div>
    
    <!-- 更多菜单 -->
    <div v-if="showMoreMenuFlag" class="context-menu" :style="{ left: moreMenuPosition.x + 'px', top: moreMenuPosition.y + 'px' }">
      <div class="context-menu-item" @click="activeOption = 'channels'; closeMoreMenu()">
        <span class="context-menu-icon"><i class="fas fa-bullhorn"></i></span>
        <span>频道</span>
      </div>
    </div>
    
    <!-- 关于对话框 -->
    <div v-if="showAboutDialog" class="about-dialog-overlay" @click="closeAboutDialog">
      <div class="about-dialog" @click.stop>
        <div class="about-dialog-header">
          <h3>关于</h3>
          <button class="about-dialog-close" @click="closeAboutDialog">×</button>
        </div>
        <div class="about-dialog-content">
          <div class="about-dialog-logo">
            <i class="fas fa-comments fa-4x"></i>
          </div>
          <h2>QIM</h2>
          <p class="version">版本: 1.0.0</p>
          <p class="date">发布日期: 2026-04-11</p>
          <p class="author">作者: huangqun@buaa.edu.cn</p>
          <p class="copyright">© 2026 QIM</p>
          <p class="description">
            一个现代化的即时通讯界面，提供简洁、高效的聊天体验。
          </p>
        </div>
        <div class="about-dialog-footer">
          <button class="about-dialog-button" @click="closeAboutDialog">确定</button>
        </div>
      </div>
    </div>
    
    <!-- 退出登录确认对话框 -->
    <div v-if="showLogoutDialog" class="logout-dialog-overlay" @click="cancelLogout">
      <div class="logout-dialog" @click.stop>
        <div class="logout-dialog-header">
          <h3>退出登录</h3>
          <button class="logout-dialog-close" @click="cancelLogout">×</button>
        </div>
        <div class="logout-dialog-content">
          <p class="logout-dialog-message">确定要退出登录吗？</p>
        </div>
        <div class="logout-dialog-footer">
          <button class="logout-dialog-button cancel-button" @click="cancelLogout">取消</button>
          <button class="logout-dialog-button confirm-button" @click="confirmLogout">确定</button>
        </div>
      </div>
    </div>
    
    <!-- 检查更新对话框 -->
    <div v-if="showUpdateDialog" class="update-dialog-overlay" @click="closeUpdateDialog">
      <div class="update-dialog" @click.stop>
        <div class="update-dialog-header">
          <h3>检查更新</h3>
          <button class="update-dialog-close" @click="closeUpdateDialog">×</button>
        </div>
        <div class="update-dialog-content">
          <div v-if="isCheckingUpdate" class="update-loading">
            <div class="loading-spinner"></div>
            <p class="loading-text">正在检查更新...</p>
          </div>
          <div v-else-if="isDownloading" class="update-downloading">
            <div class="download-icon">
              <i class="fas fa-download"></i>
            </div>
            <p class="download-text">正在下载更新...</p>
            <div class="download-progress">
              <div class="progress-bar">
                <div class="progress-fill" :style="{ width: downloadProgress + '%' }"></div>
              </div>
              <p class="progress-text">{{ downloadProgress }}%</p>
            </div>
          </div>
          <div v-else class="update-result">
            <div class="result-icon" :class="{ 'new-version': hasNewVersion }">
              <i v-if="hasNewVersion" class="fas fa-arrow-circle-up"></i>
              <i v-else class="fas fa-check-circle"></i>
            </div>
            <p class="result-text">{{ updateResult }}</p>
            <p class="version-info">当前版本: 1.0.0</p>
          </div>
        </div>
        <div class="update-dialog-footer">
          <button v-if="hasNewVersion && !isDownloading" class="update-dialog-button update-button" @click="downloadUpdate">升级</button>
          <button class="update-dialog-button" @click="closeUpdateDialog">关闭</button>
        </div>
      </div>
    </div>
    
    <!-- 通知中心 -->
    <NotificationCenter 
      ref="notificationCenterRef"
      :show="showNotificationCenterFlag" 
      :position="notificationCenterPosition"
      @close="closeNotificationCenter"
      @notification-click="handleNotificationCenterClick"
    />

    <!-- 语音通话模态框 -->
    <div v-if="showVoiceCallModal" class="voice-call-modal" @click="endVoiceCall">
      <div class="voice-call-content" @click.stop>
        <div class="voice-call-header">
          <h3>语音通话</h3>
        </div>
        <div class="voice-call-body">
          <div class="call-status">
            <div v-if="voiceCallStatus === 'calling'" class="call-status-text">
              <i class="fas fa-phone-alt"></i>
              <span>正在呼叫...</span>
            </div>
            <div v-else-if="voiceCallStatus === 'ringing'" class="call-status-text">
              <i class="fas fa-phone-ring"></i>
              <span>对方正在接听...</span>
            </div>
            <div v-else-if="voiceCallStatus === 'active'" class="call-status-text">
              <i class="fas fa-phone"></i>
              <span>通话中</span>
              <div class="call-duration">{{ formatCallDuration(voiceCallDuration) }}</div>
            </div>
            <div v-else-if="voiceCallStatus === 'ended'" class="call-status-text">
              <i class="fas fa-phone-slash"></i>
              <span>通话已结束</span>
            </div>
          </div>
        </div>
        <div class="voice-call-footer">
          <button class="end-call-btn" @click="endVoiceCall">
            <i class="fas fa-phone-slash"></i>
            结束通话
          </button>
        </div>
      </div>
    </div>
  </div>
  
  <!-- 系统设置页面 -->
  <div v-if="showSettingsModal" class="settings-modal" @click="closeSettingsModal">
    <div class="settings-content" @click.stop>
      <div class="settings-header">
        <h3>系统设置</h3>
        <button class="close-btn" @click="closeSettingsModal">×</button>
      </div>
      <div class="settings-body">
        <div class="settings-sidebar">
          <div 
            class="settings-sidebar-item" 
            :class="{ active: activeSettingsTab === 'basic' }" 
            @click="activeSettingsTab = 'basic'"
          >
            <i class="fas fa-user"></i>
            <span>基本设置</span>
          </div>
          <div 
            class="settings-sidebar-item" 
            :class="{ active: activeSettingsTab === 'message' }" 
            @click="activeSettingsTab = 'message'"
          >
            <i class="fas fa-comment"></i>
            <span>消息设置</span>
          </div>
          <div 
            class="settings-sidebar-item" 
            :class="{ active: activeSettingsTab === 'appearance' }" 
            @click="activeSettingsTab = 'appearance'"
          >
            <i class="fas fa-paint-brush"></i>
            <span>外观设置</span>
          </div>
          <div 
            class="settings-sidebar-item" 
            :class="{ active: activeSettingsTab === 'advanced' }" 
            @click="activeSettingsTab = 'advanced'"
          >
            <i class="fas fa-cog"></i>
            <span>高级设置</span>
          </div>
          <div 
            class="settings-sidebar-item" 
            :class="{ active: activeSettingsTab === 'file' }" 
            @click="activeSettingsTab = 'file'"
          >
            <i class="fas fa-file"></i>
            <span>文件设置</span>
          </div>
        </div>
        <div class="settings-main">
          <!-- 基本设置 -->
          <div v-if="activeSettingsTab === 'basic'" class="settings-section">
            <div class="settings-section-header">
              <h4>个人信息</h4>
            </div>
            <div class="settings-item">
              <label>头像</label>
              <div class="avatar-setting">
                <div class="current-avatar">
                  <img :src="(currentUser?.avatar && currentUser.avatar.startsWith('http')) ? currentUser.avatar : (currentUser?.avatar ? serverUrl + currentUser.avatar : 'https://api.dicebear.com/7.x/avataaars/svg?seed=me')" :alt="currentUser?.username || 'avatar'" />
                  <button class="change-avatar-btn">更换</button>
                </div>
              </div>
            </div>
            <div class="settings-item">
              <label>昵称</label>
              <input type="text" v-model="settingsProfile.nickname" class="settings-input" />
            </div>
            <div class="settings-item">
              <label>账号</label>
              <span class="settings-value">{{ currentUser?.username || '' }}</span>
            </div>
            <div class="settings-item">
              <label>签名</label>
              <textarea v-model="settingsProfile.signature" class="settings-textarea" placeholder="输入个人签名"></textarea>
            </div>
          </div>
          
          <!-- 消息设置 -->
          <div v-if="activeSettingsTab === 'message'" class="settings-section">
            <div class="settings-section-header">
              <h4>消息通知</h4>
            </div>
            <div class="settings-item">
              <label>开启消息通知</label>
              <label class="switch">
                <input type="checkbox" v-model="messageSettings.notificationsEnabled" />
                <span class="slider round"></span>
              </label>
            </div>
            <div class="settings-item">
              <label>声音提醒</label>
              <label class="switch">
                <input type="checkbox" v-model="messageSettings.soundEnabled" />
                <span class="slider round"></span>
              </label>
            </div>
            <div class="settings-item">
              <label>桌面通知</label>
              <label class="switch">
                <input type="checkbox" v-model="messageSettings.desktopNotificationsEnabled" />
                <span class="slider round"></span>
              </label>
            </div>
            <div class="settings-item">
              <label>消息免打扰</label>
              <div class="dnd-setting">
                <select v-model="messageSettings.dndMode" class="settings-select">
                  <option value="none">关闭</option>
                  <option value="work">工作时间</option>
                  <option value="custom">自定义</option>
                </select>
              </div>
            </div>
          </div>
          
          <!-- 外观设置 -->
          <div v-if="activeSettingsTab === 'appearance'" class="settings-section">
            <div class="settings-section-header">
              <h4>主题设置</h4>
            </div>
            <div class="settings-item">
              <label>主题</label>
              <div class="theme-selector">
                <div 
                  class="theme-option" 
                  :class="{ active: appearanceSettings.theme === 'modern-light' }" 
                  @click="appearanceSettings.theme = 'modern-light'"
                >
                  <div class="theme-preview modern-light-theme"></div>
                  <span>现代浅色</span>
                </div>
                <div 
                  class="theme-option" 
                  :class="{ active: appearanceSettings.theme === 'elegant-dark' }" 
                  @click="appearanceSettings.theme = 'elegant-dark'"
                >
                  <div class="theme-preview elegant-dark-theme"></div>
                  <span>优雅深色</span>
                </div>
                <div 
                  class="theme-option" 
                  :class="{ active: appearanceSettings.theme === 'ocean-blue' }" 
                  @click="appearanceSettings.theme = 'ocean-blue'"
                >
                  <div class="theme-preview ocean-blue-theme"></div>
                  <span>海洋蓝</span>
                </div>
                <div 
                  class="theme-option" 
                  :class="{ active: appearanceSettings.theme === 'elegant-purple' }" 
                  @click="appearanceSettings.theme = 'elegant-purple'"
                >
                  <div class="theme-preview elegant-purple-theme"></div>
                  <span>高雅紫</span>
                </div>
                <div 
                  class="theme-option" 
                  :class="{ active: appearanceSettings.theme === 'warm-amber' }" 
                  @click="appearanceSettings.theme = 'warm-amber'"
                >
                  <div class="theme-preview warm-amber-theme"></div>
                  <span>温暖琥珀</span>
                </div>
                <div 
                  class="theme-option" 
                  :class="{ active: appearanceSettings.theme === 'crimson-red' }" 
                  @click="appearanceSettings.theme = 'crimson-red'"
                >
                  <div class="theme-preview crimson-red-theme"></div>
                  <span>绯红</span>
                </div>
                <div 
                  class="theme-option" 
                  :class="{ active: appearanceSettings.theme === 'emerald-green' }" 
                  @click="appearanceSettings.theme = 'emerald-green'"
                >
                  <div class="theme-preview emerald-green-theme"></div>
                  <span>翡翠绿</span>
                </div>
              </div>
            </div>
            <div class="settings-item">
              <label>字体大小</label>
              <div class="font-size-slider">
                <input 
                  type="range" 
                  v-model.number="appearanceSettings.fontSize" 
                  min="12" 
                  max="18" 
                  step="1"
                />
                <span class="font-size-value">{{ appearanceSettings.fontSize }}px</span>
              </div>
            </div>
          </div>
          
          <!-- 高级设置 -->
          <div v-if="activeSettingsTab === 'advanced'" class="settings-section">
            <div class="settings-section-header">
              <h4>高级设置</h4>
            </div>
            <div class="settings-item">
              <label>清除缓存</label>
              <button class="clear-cache-btn" @click="clearCache">清除</button>
            </div>
            <div class="settings-item">
              <label>双因素认证</label>
              <label class="switch">
                <input type="checkbox" v-model="advancedSettings.twoFactorEnabled" @change="saveTwoFactorSetting" />
                <span class="slider round"></span>
              </label>
              <div class="settings-hint">开启后，下次登录需要输入验证码</div>
            </div>
            <div class="settings-item">
              <label>账号安全</label>
              <button class="security-btn" @click="openSecuritySettings">查看</button>
            </div>
            <div class="settings-item">
              <label>关于</label>
              <div class="about-info">
                <span>版本：1.0.0</span>
              </div>
            </div>
          </div>
          
          <!-- 文件设置 -->
          <div v-if="activeSettingsTab === 'file'" class="settings-section">
            <div class="settings-section-header">
              <h4>文件设置</h4>
            </div>
            <div class="settings-item">
              <label>默认保存目录</label>
              <div class="file-path-setting">
                <input type="text" v-model="fileSettings.defaultSaveDirectory" class="settings-input" placeholder="选择默认保存目录" />
                <button class="browse-btn" @click="browseDefaultSaveDirectory">浏览</button>
              </div>
              <div class="settings-hint">设置接收文件的默认保存位置</div>
            </div>
            <div class="settings-item">
              <label>文件自动下载</label>
              <label class="switch">
                <input type="checkbox" v-model="fileSettings.autoDownload" />
                <span class="slider round"></span>
              </label>
            </div>
            <div class="settings-item">
              <label>最大上传文件大小</label>
              <div class="file-size-setting">
                <input type="number" v-model.number="fileSettings.maxFileSize" class="settings-input" placeholder="文件大小限制" />
                <span class="size-unit">MB</span>
              </div>
              <div class="settings-hint">设置单个文件的最大上传大小</div>
            </div>
            <div class="settings-item">
              <label>允许的文件类型</label>
              <input type="text" v-model="fileSettings.allowedFileTypes" class="settings-input" placeholder="例如：jpg,png,pdf,doc" />
              <div class="settings-hint">设置允许上传的文件类型，用逗号分隔</div>
            </div>
            <div class="settings-item">
              <label>图片自动预览</label>
              <label class="switch">
                <input type="checkbox" v-model="fileSettings.autoPreviewImages" />
                <span class="slider round"></span>
              </label>
            </div>
            <div class="settings-item">
              <label>文件历史记录</label>
              <label class="switch">
                <input type="checkbox" v-model="fileSettings.enableFileHistory" />
                <span class="slider round"></span>
              </label>
            </div>
          </div>
        </div>
      </div>
      <div class="settings-footer">
        <button class="cancel-btn" @click="closeSettingsModal">取消</button>
        <button class="save-btn" @click="saveSettings">保存</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, defineComponent, onMounted, onUnmounted, watch } from 'vue'
import type { Conversation, Message, User } from '../types'
import { ElMessage } from 'element-plus'
import axios from 'axios'
import CalendarApp from '../components/apps/CalendarApp.vue'
import StatisticsApp from '../components/apps/StatisticsApp.vue'
import StickyNotesApp from '../components/apps/StickyNotesApp.vue'
import NotesApp from '../components/apps/NotesApp.vue'
import TaskManagementApp from '../components/apps/TaskManagementApp.vue'
import FileManagementApp from '../components/apps/FileManagementApp.vue'
import AppManagementApp from '../components/apps/AppManagementApp.vue'
import AIAssistantApp from '../components/apps/AIAssistantApp.vue'
import ShortLinkManager from '../components/apps/ShortLinkManager.vue'

// 声明 window.electron 变量
declare global {
  interface Window {
    electron: {
      ipcRenderer: {
        send: (channel: string, data?: any) => void
      }
    } | undefined
  }
}
import Sidebar from '../components/Sidebar.vue'
import SideOptions from '../components/SideOptions.vue'
import ChatWindow from '../components/ChatWindow.vue'
import GroupList from '../components/GroupList.vue'
import GroupDetail from '../components/GroupDetail.vue'
import ShareModal from '../components/ShareModal.vue'
import UserProfile from '../components/UserProfile.vue'
import NotificationCenter from '../components/NotificationCenter.vue'
import ChannelList from '../components/ChannelList.vue'
import CreateGroupModal from '../components/CreateGroupModal.vue'
import { API_BASE_URL } from '../config'
import { generateAvatar } from '../utils/avatar'

// 服务器地址
const serverUrl = ref(localStorage.getItem('serverUrl') || API_BASE_URL)

// 获取群聊群主
const getGroupOwner = (group: Conversation | null) => {
  if (!group || !group.members) return ''
  const owner = group.members.find((member: User) => member.role === 'owner')
  return owner ? owner.name : ''
}

// 检查当前用户是否是群主
const isGroupOwner = (group: Conversation | null) => {
  if (!group || !group.members || !currentUser.value) return false
  const owner = group.members.find((member: User) => member.role === 'owner')
  return owner && owner.id === currentUser.value.id
}

// 解散群聊
const dissolveGroup = async () => {
  if (!selectedGroup.value) {
    closeGroupContextMenu()
    return
  }
  
  openConfirmDialog('确认解散群聊', `确定要解散群聊 "${selectedGroup.value.name}" 吗？此操作不可恢复。`, async () => {
    try {
      const response = await request(`/api/v1/conversations/${selectedGroup.value.id}`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json'
        }
      })
      
      if (response.code === 0) {
        ElMessage.success('群聊已成功解散')
        // 从群聊列表中移除
        const index = conversations.value.findIndex(c => c.id === selectedGroup.value?.id)
        if (index > -1) {
          conversations.value.splice(index, 1)
        }
        // 清空选中的群聊
        selectedGroup.value = null
      } else {
        ElMessage.error(response.message || '解散群聊失败')
      }
    } catch (error) {
      console.error('解散群聊失败:', error)
      ElMessage.error('网络错误，解散群聊失败')
    }
    closeGroupContextMenu()
  })
}

// 编辑群公告
const editAnnouncement = () => {
  if (!selectedGroup.value) return
  editAnnouncementContent.value = selectedGroup.value.announcement || ''
  showEditAnnouncementModal.value = true
}

// 关闭编辑群公告模态框
const closeEditAnnouncementModal = () => {
  showEditAnnouncementModal.value = false
  editAnnouncementContent.value = ''
}

// 保存群公告
const saveAnnouncement = async () => {
  if (!selectedGroup.value) {
    closeEditAnnouncementModal()
    return
  }
  
  try {
    const response = await request(`/api/v1/conversations/${selectedGroup.value.id}/announcement`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ announcement: editAnnouncementContent.value })
    })
    
    if (response.code === 0) {
      ElMessage.success('群公告已成功更新')
      // 更新本地群聊数据
      selectedGroup.value.announcement = editAnnouncementContent.value
      // 更新群聊列表中的数据
      const index = conversations.value.findIndex(c => c.id === selectedGroup.value?.id)
      if (index > -1) {
        conversations.value[index].announcement = editAnnouncementContent.value
      }
    } else {
      ElMessage.error(response.message || '更新群公告失败')
    }
  } catch (error) {
    console.error('更新群公告失败:', error)
    ElMessage.error('网络错误，更新群公告失败')
  }
  closeEditAnnouncementModal()
}

// 获取成员头像
const getMemberAvatar = (member: User) => {
  if (!member) return generateAvatar('成员')
  if (member.avatar && member.avatar.startsWith('http')) {
    return member.avatar
  }
  if (member.avatar) {
    return serverUrl.value + member.avatar
  }
  return generateAvatar(member.name || '成员')
}

// 网络连接状态
const showNetworkError = ref(false)
const networkErrorMsg = ref('网络连接失败，正在尝试重新连接...')
const sessionExpired = ref(false)
const reconnectAttempts = ref(0)
const maxReconnectAttempts = 5
const reconnectTimer = ref<number | null>(null)

// 获取 token
const getToken = () => {
  return localStorage.getItem('token')
}

// 本地存储工具（已禁用）
const storage = {
  // 存储消息
  saveMessages: (conversationId: string, messages: any[]) => {
    // 已禁用本地存储
  },
  
  // 获取消息
  getMessages: (conversationId: string) => {
    // 已禁用本地存储
    return []
  },
  
  // 存储会话
  saveConversations: (conversations: any[]) => {
    // 已禁用本地存储
  },
  
  // 获取会话
  getConversations: () => {
    // 已禁用本地存储
    return []
  },
  
  // 存储文件缓存信息
  saveFileCache: (fileId: string, filePath: string) => {
    // 已禁用本地存储
  },
  
  // 获取文件缓存信息
  getFileCache: (fileId: string) => {
    // 已禁用本地存储
    return null
  },
  
  // 清除过期的文件缓存
  clearExpiredFileCache: () => {
    // 已禁用本地存储
  }
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
  messageElement.style.transition = 'all 0.3s ease'
  
  // 设置内容
  messageElement.textContent = message
  
  // 添加到页面
  document.body.appendChild(messageElement)
  
  // 自动移除
  setTimeout(() => {
    messageElement.style.animation = 'messageFadeOut 0.3s ease'
    setTimeout(() => {
      if (document.body.contains(messageElement)) {
        document.body.removeChild(messageElement)
      }
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

// 通用请求方法
const request = async (url: string, options?: RequestInit) => {
  const token = getToken()
  
  // 构建基础headers
  const headers: Record<string, string> = {}
  
  // 只有当不是FormData时才设置Content-Type
  if (!options?.body || !(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }
  
  // 添加Authorization头
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  
  const fullUrl = `${serverUrl.value}${url}`
  console.log('发送请求:', fullUrl, options)
  
  try {
    const response = await fetch(fullUrl, {
      ...options,
      headers: {
        ...headers,
        ...options?.headers
      }
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
    console.error('网络请求错误:', error)
    throw error
  }
}

const currentConversationId = ref<string | null>(null)
const activeOption = ref('recent')
const searchQuery = ref('')
const unreadNotificationCount = ref(0)
const selectedChannel = ref(null)
const selectedGroup = ref(null)

// 会话数据
const conversations = ref<Conversation[]>([])
const isLoading = ref(false)

// 用户资料弹窗
const showUserProfile = ref(false)
const selectedUser = ref(null)

// 关闭用户资料弹窗
const closeUserProfile = () => {
  showUserProfile.value = false
  selectedUser.value = null
}

// 处理频道选择
const handleChannelSelect = (channel) => {
  selectedChannel.value = channel
}

// 分享相关状态
const showShareModal = ref(false)
const shareType = ref('')

const shareUsers = ref<any[]>([])
const shareGroups = ref<any[]>([])

// 频道相关状态
const channelMessage = ref('')

// 检查当前用户是否是频道创建者
const isChannelCreator = (channel) => {
  return currentUser.value?.id === channel.creator_id?.toString()
}

// 订阅频道
const subscribeChannel = async (channel) => {
  try {
    const response = await fetch(`${serverUrl.value}/api/v1/channels/${channel.id}/subscribe`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    
    if (response.ok) {
      channel.is_subscribed = true
      ElMessage.success('订阅成功')
    } else {
      ElMessage.error('订阅失败')
    }
  } catch (error) {
    console.error('订阅频道失败:', error)
    ElMessage.error('订阅失败')
  }
}

// 取消订阅频道
const unsubscribeChannel = async (channel) => {
  try {
    const response = await fetch(`${serverUrl.value}/api/v1/channels/${channel.id}/unsubscribe`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      }
    })
    
    if (response.ok) {
      channel.is_subscribed = false
      ElMessage.success('取消订阅成功')
    } else {
      ElMessage.error('取消订阅失败')
    }
  } catch (error) {
    console.error('取消订阅频道失败:', error)
    ElMessage.error('取消订阅失败')
  }
}

// 发送频道消息
const sendChannelMessage = async (channel) => {
  if (!channelMessage.value.trim()) return
  
  try {
    const response = await fetch(`${serverUrl.value}/api/v1/channels/${channel.id}/messages`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${localStorage.getItem('token')}`
      },
      body: JSON.stringify({
        content: channelMessage.value
      })
    })
    
    if (response.ok) {
      const newMessage = await response.json()
      if (!channel.messages) {
        channel.messages = []
      }
      channel.messages.push(newMessage)
      channelMessage.value = ''
      ElMessage.success('发送成功')
    } else {
      ElMessage.error('发送失败')
    }
  } catch (error) {
    console.error('发送频道消息失败:', error)
    ElMessage.error('发送失败')
  }
}

// 获取当前登录用户信息
const getCurrentUser = () => {
  const userStr = localStorage.getItem('user')
  if (userStr) {
    try {
      const user = JSON.parse(userStr)
      if (user && user.id) {
        user.isAdmin = true
        return user
      }
    } catch (error) {
      console.error('解析用户信息失败:', error)
    }
  }
  // 模拟用户信息，用于测试
  return {
    id: '1',
    username: 'admin',
    nickname: '管理员',
    avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=admin',
    isAdmin: true
  }
}

// 当前登录用户信息
const currentUser = ref(getCurrentUser())

// 用户资料弹窗
const userProfile = ref({
  nickname: currentUser.value?.nickname || currentUser.value?.username || '我的账号',
  username: currentUser.value?.username || '',
  signature: currentUser.value?.signature || '这个人很懒，什么都没留下',
  id: currentUser.value?.id?.toString() || 'user_123456',
  joinDate: '2023-01-01'
})

// 同步用户资料
const syncUserProfile = () => {
  userProfile.value = {
    nickname: currentUser.value?.nickname || currentUser.value?.username || '我的账号',
    username: currentUser.value?.username || '',
    signature: currentUser.value?.signature || '这个人很懒，什么都没留下',
    id: currentUser.value?.id?.toString() || 'user_123456',
    joinDate: '2023-01-01'
  }
}

// 监听 currentUser 变化，同步更新 userProfile
watch(() => currentUser.value, () => {
  syncUserProfile()
}, { deep: true })

// 处理会话数据，确保 sender 字段正确
const processConversation = (conv: any) => {
  const members = conv.members ? conv.members.map((member: any) => ({
    id: member.user && member.user.id ? member.user.id.toString() : (member.UserID ? member.UserID.toString() : (member.user_id ? member.user_id.toString() : '')),
    name: member.user ? (member.user.nickname || member.user.username || '') : (member.User ? (member.User.Nickname || member.User.Username || '') : ''),
    username: member.user ? member.user.username || '' : (member.User ? member.User.Username || '' : ''),
    avatar: (member.user && member.user.avatar && member.user.avatar.startsWith('http')) ? member.user.avatar : (member.user && member.user.avatar ? serverUrl.value + member.user.avatar : (member.User && member.User.Avatar ? serverUrl.value + member.User.Avatar : '')),
    role: member.role || member.Role || 'member'
  })) : []
  
  // 为单聊会话设置对方用户的头像和名称
  let avatar = conv.avatar || ''
  let name = conv.name || ''
  if ((conv.type !== 'group' && conv.type !== 'discussion') && members.length > 1) {
    const currentUserId = currentUser.value?.id?.toString() || ''
    const otherMember = members.find((m: any) => m.id !== currentUserId)
    if (otherMember) {
      avatar = otherMember.avatar || ''
      name = otherMember.name || ''
    }
  }
  
  // 处理群聊和讨论组头像
  if ((conv.type === 'group' || conv.type === 'discussion') && conv.avatar) {
    avatar = (conv.avatar.startsWith('http')) ? conv.avatar : serverUrl.value + conv.avatar
  }
  
  // 检查是否有未读消息
  let unreadCount = conv.unread_count || 0
  
  // 日志：检查服务器返回的会话数据
  console.log('Server conversation data:', conv)
  console.log('Server last_message:', conv.last_message)
  console.log('Server last_message.sender:', conv.last_message?.sender)
  
  // 获取发送人信息
  const getSenderInfo = (senderId: string, members: any[]) => {
    // 首先尝试从 members 中找到对应的用户
    const member = members.find(m => m.id === senderId)
    if (member) {
      return {
        id: member.id,
        name: member.name || '',
        avatar: member.avatar || ''
      }
    }
    // 如果找不到，返回空对象
    return {
      id: '',
      name: '',
      avatar: ''
    }
  }
  
  const conversationObj = {
    id: conv.id ? conv.id.toString() : (conv.ID ? conv.ID.toString() : ''),
    name: name || '',
    avatar: avatar || 'https://api.dicebear.com/7.x/avataaars/svg?seed=user',
    lastMessage: conv.lastMessage || conv.last_message ? {
      id: (conv.lastMessage?.id || conv.last_message?.id) ? (conv.lastMessage?.id || conv.last_message?.id).toString() : '',
      content: conv.lastMessage?.content || conv.last_message?.content || '',

      sender: (conv.lastMessage?.sender || conv.last_message?.sender) ? {
        id: (conv.lastMessage?.sender?.id || conv.last_message?.sender?.id) ? (conv.lastMessage?.sender?.id || conv.last_message?.sender?.id).toString() : '',
        name: conv.lastMessage?.sender?.nickname || conv.lastMessage?.sender?.username || conv.lastMessage?.sender?.name || conv.lastMessage?.sender?.user?.nickname || conv.lastMessage?.sender?.user?.username || conv.last_message?.sender?.nickname || conv.last_message?.sender?.username || conv.last_message?.sender?.name || conv.last_message?.sender?.user?.nickname || conv.last_message?.sender?.user?.username || '',
        username: conv.lastMessage?.sender?.username || conv.lastMessage?.sender?.user?.username || conv.last_message?.sender?.username || conv.last_message?.sender?.user?.username || '',
        avatar: ((conv.lastMessage?.sender?.avatar || conv.last_message?.sender?.avatar) && (conv.lastMessage?.sender?.avatar || conv.last_message?.sender?.avatar).startsWith('http')) ? (conv.lastMessage?.sender?.avatar || conv.last_message?.sender?.avatar) : ((conv.lastMessage?.sender?.avatar || conv.last_message?.sender?.avatar) ? serverUrl.value + (conv.lastMessage?.sender?.avatar || conv.last_message?.sender?.avatar) : ''),
        // 保存原始 sender 对象，以便在需要时访问更多属性
        user: conv.lastMessage?.sender || conv.last_message?.sender
      } : {
        id: '',
        name: '',
        username: '',
        avatar: ''
      },
      timestamp: conv.lastMessage?.created_at || conv.last_message?.created_at ? new Date(conv.lastMessage?.created_at || conv.last_message?.created_at).getTime() : Date.now(),
      type: conv.lastMessage?.type || conv.last_message?.type || 'text',
      isSelf: false,
      miniAppData: (() => {
        try {
          const content = conv.lastMessage?.content || conv.last_message?.content
          if ((conv.lastMessage?.type || conv.last_message?.type) === 'miniApp' && content && content !== '[消息已撤回]') {
            return JSON.parse(content)
          }
          return undefined
        } catch (e) {
          console.error('解析小程序数据失败:', e)
          return undefined
        }
      })(),
      shareData: (() => {
        try {
          const content = conv.lastMessage?.content || conv.last_message?.content
          if ((conv.lastMessage?.type || conv.last_message?.type) === 'share' && content && content !== '[消息已撤回]') {
            return JSON.parse(content)
          }
          return undefined
        } catch (e) {
          console.error('解析分享数据失败:', e)
          return undefined
        }
      })()
    } : undefined,
    unreadCount: unreadCount,
    timestamp: conv.last_message_at ? new Date(conv.last_message_at).getTime() : (conv.created_at ? new Date(conv.created_at).getTime() : Date.now()),
    type: (conv.type === 'group' || conv.type === 'Group' || conv.type === 'GROUP') ? 'group' : (conv.type === 'discussion' || conv.type === 'Discussion' || conv.type === 'DISCUSSION') ? 'discussion' : (conv.type === 'bot' ? 'bot' : 'single'),
    members: members,
    pinned: conv.is_pinned || false,
    muted: conv.muted || false
  }
  
  // 如果是群聊，并且 lastMessage 存在，但 sender.name 为空，尝试从 members 中获取发送人信息
  if (conversationObj.type === 'group' && conversationObj.lastMessage) {
    const senderId = conversationObj.lastMessage.sender?.id || (conv.lastMessage?.sender_id || conv.last_message?.sender_id)?.toString() || ''
    if (senderId && (!conversationObj.lastMessage.sender?.name || conversationObj.lastMessage.sender.name === '')) {
      const senderInfo = getSenderInfo(senderId, members)
      if (senderInfo.name) {
        conversationObj.lastMessage.sender.name = senderInfo.name
        if (senderInfo.avatar) {
          conversationObj.lastMessage.sender.avatar = senderInfo.avatar
        }
      }
    }
  }
  
  // 日志：检查处理后的会话对象
  console.log('Processed conversation:', conversationObj)
  console.log('Processed lastMessage:', conversationObj.lastMessage)
  console.log('Processed lastMessage.sender:', conversationObj.lastMessage?.sender)
  
  return conversationObj
}

// 加载会话列表
const loadConversations = async () => {
  isLoading.value = true
  try {
    // 从服务器获取最新会话
    const response = await request('/api/v1/conversations')
    if (response.code === 0 && response.data) {
      const serverConversations = response.data.map((conv: any) => processConversation(conv))
      
      conversations.value = serverConversations
    } else {
      conversations.value = []
    }
  } catch (error) {
    console.error('加载会话失败:', error)
    conversations.value = []
  } finally {
    isLoading.value = false
  }
}

// 加载组织架构
const loadOrganizationTree = async () => {
  try {
    const response = await request('/api/v1/organization/tree')
    if (response.code === 0) {
      // 处理组织架构数据
      console.log('组织架构数据:', response.data)
      // 将后端返回的数据转换为前端期望的格式
      const convertDepartments = (departments) => {
        return departments.map(dept => ({
          id: dept.id ? dept.id.toString() : '',
          name: dept.name || '',
          subDepartments: dept.subDepartments ? convertDepartments(dept.subDepartments) : [],
          employees: dept.employees ? dept.employees.map(emp => ({
            id: emp.id ? emp.id.toString() : '',
            name: emp.nickname || emp.username || '',
            username: emp.username || '',
            avatar: (emp.avatar && emp.avatar.startsWith('http')) ? emp.avatar : (emp.avatar ? serverUrl.value + emp.avatar : 'https://api.dicebear.com/7.x/avataaars/svg?seed=emp'),
            position: '', // 后端没有提供职位信息
            department: dept.name, // 添加部门信息
            status: emp.status || 'offline' // 在线状态，默认为 offline
          })) : []
        }))
      }
      // 将转换后的数据赋值给 orgStructure
      orgStructure.value = convertDepartments(response.data)
    }
  } catch (error) {
    console.error('加载组织架构失败:', error)
  }
}

// 点击组织架构中的用户
const handleUserClick = (employee) => {
  selectedUser.value = employee
}

// 计算部门人数和在线人数
const getDepartmentStats = (department) => {
  let totalCount = 0
  let onlineCount = 0

  // 统计当前部门员工
  if (department.employees) {
    totalCount += department.employees.length
    // 根据员工的 status 字段判断是否在线
    onlineCount += department.employees.filter(emp => emp.status === 'online').length
  }

  // 递归统计子部门
  if (department.subDepartments) {
    department.subDepartments.forEach(subDept => {
      const stats = getDepartmentStats(subDept)
      totalCount += stats.total
      onlineCount += stats.online
    })
  }

  return { total: totalCount, online: onlineCount }
}

// 轮询检查新消息
let messagePollingInterval: number | null = null

// 开始轮询


// 初始化数据
onMounted(async () => {
  loadConversations()
  loadOrganizationTree()
  // 连接WebSocket（不再使用轮询，完全依赖WebSocket）
  connectWebSocket()
  
  // 加载用户创建的应用
  await loadUserApps()
  
  // 监听分享便签事件
  window.addEventListener('shareStickyNote', (event: CustomEvent) => {
    const note = event.detail
    openShareModal('sticky', note)
  })
  
  // 监听消息转发事件
  window.addEventListener('forwardMessage', (event: CustomEvent) => {
    const message = event.detail.message
    if (message) {
      openShareModal('message', message)
    }
  })
  
  // 监听文件分享事件
  window.addEventListener('openShareModal', (event: CustomEvent) => {
    const { type, data } = event.detail
    openShareModal(type, data)
  })
  
  // 监听刷新用户应用事件
  window.addEventListener('refresh-user-apps', async () => {
    await loadUserApps()
  })
})

// WebSocket连接
let ws: WebSocket | null = null
const baseReconnectDelay = 1000 // 1秒

// 连接WebSocket
const connectWebSocket = () => {
  if (ws && ws.readyState === WebSocket.OPEN) return
  
  // 获取token
  const token = localStorage.getItem('token')
  if (!token) return
  
  // 隐藏网络错误提示
  showNetworkError.value = false
  networkErrorMsg.value = '网络连接失败，正在尝试重新连接...'
  sessionExpired.value = false
  
  // 连接WebSocket
  ws = new WebSocket(`ws://${serverUrl.value.replace('http://', '')}/api/v1/ws?token=${token}`)
  
  ws.onopen = () => {
    console.log('WebSocket连接成功')
    // 重置重连尝试次数
    reconnectAttempts.value = 0
  }
  
  ws.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data)
      handleWebSocketMessage(message)
    } catch (error) {
      console.error('解析WebSocket消息失败:', error)
    }
  }
  
  ws.onclose = () => {
    console.log('WebSocket连接关闭')
    // 尝试重连
    if (reconnectAttempts.value < maxReconnectAttempts) {
      // 指数退避策略
      const delay = baseReconnectDelay * Math.pow(2, reconnectAttempts.value)
      console.log(`WebSocket重连尝试 ${reconnectAttempts.value + 1}/${maxReconnectAttempts}，延迟 ${delay}ms`)
      
      // 显示网络错误提示
      showNetworkError.value = true
      networkErrorMsg.value = `网络连接失败，正在尝试重新连接... (${reconnectAttempts.value + 1}/${maxReconnectAttempts})`
      
      reconnectTimer.value = window.setTimeout(() => {
        reconnectAttempts.value++
        connectWebSocket()
      }, delay)
    } else {
      console.log('WebSocket重连失败，已达到最大重试次数')
      // 显示最终错误提示
      showNetworkError.value = true
      networkErrorMsg.value = '网络连接失败，请检查网络设置或稍后重试'
    }
  }
  
  ws.onerror = (error) => {
    console.error('WebSocket错误:', error)
    // 检查是否是会话过期错误
    if (error.message && error.message.includes('401')) {
      sessionExpired.value = true
      showNetworkError.value = true
      networkErrorMsg.value = '会话已过期，请重新登录'
    }
  }
}

// 处理WebSocket消息
const handleWebSocketMessage = (message: any) => {
  switch (message.type) {
    case 'message_read':
      // 处理已读回执
      handleReadReceipt(message.data)
      break
    case 'new_message':
      // 处理新消息
      handleNewMessage(message.data)
      break
    case 'message_recalled':
      // 处理消息撤回
      handleMessageRecalled(message.data)
      break
    case 'message_deleted':
      // 处理消息删除
      handleMessageDeleted(message.data)
      break
    case 'group_invitation':
      // 处理群聊邀请
      handleGroupInvitation(message.data)
      break
    case 'added_to_group':
      // 处理被添加到群聊
      handleAddedToGroup(message.data)
      break
    case 'group_member_left':
      // 处理成员退出群聊
      handleGroupMemberLeft(message.data)
      break
    case 'group_member_joined':
      // 处理成员加入群聊
      handleGroupMemberJoined(message.data)
      break
    case 'group_member_role_updated':
      // 处理群成员角色更新
      handleGroupMemberRoleUpdated(message.data)
      break
    case 'group_owner_transferred':
      // 处理群主转让
      handleGroupOwnerTransferred(message.data)
      break
    case 'conversation_updated':
      // 处理会话更新
      handleConversationUpdated(message.data)
      break
    case 'group_announcement_updated':
      // 处理群公告更新
      handleGroupAnnouncementUpdated(message.data)
      break
    case 'notification':
      // 处理通知
      handleNotification(message.data)
      break
    case 'new_notification':
      // 处理新通知
      handleNewNotification(message.data)
      break
    case 'system_message':
      // 处理系统消息
      handleSystemMessage(message.data)
      break
    default:
      break
  }
}

// 处理群聊邀请
const handleGroupInvitation = (data: any) => {
  console.log('收到群聊邀请:', data)
  // 显示群聊邀请通知
  showMessage({
    message: `您收到了加入群聊 "${data.group_name}" 的邀请`,
    type: 'info',
    duration: 5000
  })
}

// 处理已读回执事件


// 处理被添加到群聊
const handleAddedToGroup = (data: any) => {
  console.log('被添加到群聊:', data)
  
  // 构建群聊会话对象
  const groupConversation = {
    id: data.conversation_id.toString(),
    name: data.group_name,
    avatar: data.group_avatar || 'https://api.dicebear.com/7.x/identicon/svg?seed=group',
    lastMessage: null,
    unreadCount: 0,
    timestamp: Date.now(),
    type: 'group',
    members: data.members || []
  }
  
  // 检查会话是否已存在
  const existingIndex = conversations.value.findIndex(c => c.id === groupConversation.id)
  if (existingIndex === -1) {
    // 添加到会话列表
    conversations.value.unshift(groupConversation)
    // 强制触发响应式更新
    conversations.value = [...conversations.value]
  } else {
    // 更新现有会话的成员列表
    const updatedConversation = {
      ...conversations.value[existingIndex],
      members: data.members || []
    }
    conversations.value.splice(existingIndex, 1, updatedConversation)
    // 强制触发响应式更新
    conversations.value = [...conversations.value]
  }
  
  // 保存会话到本地存储
  storage.saveConversations(conversations.value)
  
  // 显示通知
  showMessage({
    message: `您已被添加到群聊 "${data.group_name}"`,
    type: 'success',
    duration: 5000
  })
}

// 处理成员退出群聊
const handleGroupMemberLeft = (data: any) => {
  console.log('成员退出群聊:', data)
  
  const conversationId = data.conversation_id.toString()
  const userId = data.user_id.toString()
  
  // 更新会话列表中的群聊成员信息
  const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
  if (conversationIndex !== -1) {
    const conversation = conversations.value[conversationIndex]
    if (conversation.members) {
      // 过滤掉退出的成员
      const updatedMembers = conversation.members.filter(member => member.id !== userId)
      
      // 创建新的会话对象，确保响应式更新
      const updatedConversation = {
        ...conversation,
        members: updatedMembers
      }
      
      // 替换会话对象，触发响应式更新
      conversations.value.splice(conversationIndex, 1, updatedConversation)
      
      // 强制触发响应式更新
      conversations.value = [...conversations.value]
      
      // 保存会话到本地存储
      storage.saveConversations(conversations.value)
    }
  }
  
  // 如果是当前用户退出群聊，标记为已退出
  if (userId === currentUser.value?.id?.toString()) {
    const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
    if (conversationIndex !== -1) {
      const updatedConversation = {
        ...conversations.value[conversationIndex],
        isExited: true
      }
      conversations.value.splice(conversationIndex, 1, updatedConversation)
      conversations.value = [...conversations.value]
      storage.saveConversations(conversations.value)
    }
  }
}

// 处理成员加入群聊
const handleGroupMemberJoined = (data: any) => {
  console.log('成员加入群聊:', data)

  const conversationId = data.conversation_id.toString()
  const newMember = data.member
  const memberName = newMember.nickname || newMember.username || (newMember.name !== undefined ? newMember.name : '未知用户')

  const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
  if (conversationIndex !== -1) {
    const conversation = conversations.value[conversationIndex]

    const memberExists = conversation.members && conversation.members.some(member => member.id === newMember.id?.toString())

    if (!memberExists) {
      const updatedMembers = [...(conversation.members || []), {
        id: newMember.id?.toString() || '',
        name: memberName,
        avatar: newMember.avatar || ''
      }]

      const updatedConversation = {
        ...conversation,
        members: updatedMembers
      }

      conversations.value.splice(conversationIndex, 1, updatedConversation)
      conversations.value = [...conversations.value]
      storage.saveConversations(conversations.value)

      if (currentConversationId.value === conversationId) {
        const systemMessage = {
          id: `system_${Date.now()}`,
          type: 'system',
          content: `${memberName} 加入了群聊`,
          timestamp: Date.now(),
          sender: {
            id: 'system',
            name: '系统',
            avatar: ''
          },
          isSelf: false,
          isRead: true
        }
        messages.value.push(systemMessage)
      }
    }
  }
}

// 处理系统消息
const handleSystemMessage = (data: any) => {
  console.log('收到系统消息:', data)
  
  // 显示系统消息通知
  showMessage({
    message: `系统消息: ${data.title}`,
    type: 'info',
    duration: 5000
  })
  
  // 将系统消息添加到通知中心
  if (notificationCenterRef.value) {
    const newNotification = {
      id: Date.now().toString(),
      title: data.title,
      content: data.content,
      timestamp: Date.now(),
      read: false,
      type: 'system' as const
    }
    
    // 获取当前通知列表
    const currentNotifications = notificationCenterRef.value.notifications || []
    // 添加新通知到列表开头
    notificationCenterRef.value.notifications = [newNotification, ...currentNotifications]
    // 重新过滤通知
    notificationCenterRef.value.filterNotifications()
    
    // 更新未读通知计数
    unreadNotificationCount.value++
  }
  
  // 可以在这里添加更新系统消息列表的逻辑
  // 例如，重新加载系统消息列表
  // loadSystemMessages()
}

// 处理消息删除
const handleMessageDeleted = (data: any) => {
  console.log('消息被删除:', data)
  // 从消息列表中移除被删除的消息
  const index = messages.value.findIndex(msg => msg.id === data.message_id)
  if (index !== -1) {
    messages.value.splice(index, 1)
  }
}

// 处理通知
const handleNotification = (data: any) => {
  console.log('收到通知:', data)
  // 显示通知
  showMessage({
    message: data.content,
    type: 'info',
    duration: 5000
  })
  
  // 将通知添加到通知中心
  if (notificationCenterRef.value) {
    const newNotification = {
      id: Date.now().toString(),
      title: data.title,
      content: data.content,
      timestamp: Date.now(),
      read: false,
      type: data.type as 'group_invitation' | 'group_member_added' | 'other'
    }
    
    // 获取当前通知列表
    const currentNotifications = notificationCenterRef.value.notifications || []
    // 添加新通知到列表开头
    notificationCenterRef.value.notifications = [newNotification, ...currentNotifications]
    // 重新过滤通知
    notificationCenterRef.value.filterNotifications()
    
    // 更新未读通知计数
    unreadNotificationCount.value++
  }
}

// 处理会话更新
const handleConversationUpdated = (data: any) => {
  console.log('会话更新:', data)
  // 更新会话列表中的会话信息
  const conversationIndex = conversations.value.findIndex(c => c.id === data.id.toString())
  if (conversationIndex !== -1) {
    conversations.value[conversationIndex] = {
      ...conversations.value[conversationIndex],
      ...data
    }
    // 强制触发响应式更新
    conversations.value = [...conversations.value]
    // 保存会话到本地存储
    storage.saveConversations(conversations.value)
  }
}

// 处理群公告更新
const handleGroupAnnouncementUpdated = (data: any) => {
  console.log('群公告更新:', data)

  const conversationId = data.conversation_id.toString()
  const newAnnouncement = data.announcement || ''

  const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
  if (conversationIndex !== -1) {
    conversations.value[conversationIndex] = {
      ...conversations.value[conversationIndex],
      announcement: newAnnouncement
    }
    conversations.value = [...conversations.value]
    storage.saveConversations(conversations.value)

    if (currentConversationId.value === conversationId) {
      const updaterName = data.updater_name || data.operator_name || '未知用户'
      const systemMessage = {
        id: `system_${Date.now()}`,
        type: 'system',
        content: `${updaterName} 更新了群公告: ${newAnnouncement || '(无)'}`,
        timestamp: Date.now(),
        sender: {
          id: 'system',
          name: '系统',
          avatar: ''
        },
        isSelf: false,
        isRead: true
      }
      messages.value.push(systemMessage)
      
      // 保存系统消息到本地存储
      storage.saveMessages(conversationId, messages.value)
    }
  }
}

// 处理群成员角色更新
const handleGroupMemberRoleUpdated = (data: any) => {
  console.log('群成员角色更新:', data)
  // 更新群成员列表中的角色信息
  const conversationIndex = conversations.value.findIndex(c => c.id === data.conversation_id.toString())
  if (conversationIndex !== -1) {
    const conversation = conversations.value[conversationIndex]
    if (conversation.members) {
      const memberIndex = conversation.members.findIndex(m => m.id === data.user_id.toString())
      if (memberIndex !== -1) {
        conversation.members[memberIndex].role = data.role
        // 强制触发响应式更新
        conversations.value = [...conversations.value]
        // 保存会话到本地存储
        storage.saveConversations(conversations.value)
      }
    }
  }
}

// 处理群主转让
const handleGroupOwnerTransferred = (data: any) => {
  console.log('群主转让:', data)
  // 更新群成员列表中的角色信息
  const conversationIndex = conversations.value.findIndex(c => c.id === data.conversation_id.toString())
  if (conversationIndex !== -1) {
    const conversation = conversations.value[conversationIndex]
    if (conversation.members) {
      // 更新旧群主的角色
      const oldOwnerIndex = conversation.members.findIndex(m => m.id === data.old_owner_id.toString())
      if (oldOwnerIndex !== -1) {
        conversation.members[oldOwnerIndex].role = 'member'
      }
      // 更新新群主的角色
      const newOwnerIndex = conversation.members.findIndex(m => m.id === data.new_owner_id.toString())
      if (newOwnerIndex !== -1) {
        conversation.members[newOwnerIndex].role = 'owner'
      }
      // 强制触发响应式更新
      conversations.value = [...conversations.value]
      // 保存会话到本地存储
      storage.saveConversations(conversations.value)
    }
  }
}

// 处理已读回执
const handleReadReceipt = (data: any) => {
  const { conversation_id, user_id } = data
  
  // 只处理当前会话的已读回执
  if (currentConversationId.value !== conversation_id.toString()) return
  
  // 每次收到已读回执都处理，不再检查重复
  // 这样可以确保所有消息的已读状态都能正确更新
  
  // 更新消息的已读状态
  messages.value = messages.value.map(msg => {
    // 更新自己发送的消息（对方已读）
    if (msg.isSelf) {
      return { ...msg, isRead: true }
    }
    return msg
  })
  
  // 强制触发响应式更新，确保UI及时更新
  messages.value = [...messages.value]
  
  console.log('处理已读回执，更新了消息状态，当前消息数量:', messages.value.length)
}

// 处理消息撤回
const handleMessageRecalled = (data: any) => {
  const messageId = data.id.toString()
  const conversationId = data.conversation_id.toString()
  
  console.log('收到消息撤回通知:', data)
  
  // 更新消息列表中的消息状态
  messages.value = messages.value.map(msg => {
    if (msg.id === messageId) {
      return { ...msg, content: '[消息已撤回]', isRecalled: true }
    }
    return msg
  })
  
  // 更新会话列表中的最后消息
  const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
  if (conversationIndex !== -1) {
    const conversation = conversations.value[conversationIndex]
    if (conversation.lastMessage && conversation.lastMessage.id === messageId) {
      // 创建新的会话对象，确保响应式更新
      const updatedConversation = {
        ...conversation,
        lastMessage: {
          ...conversation.lastMessage,
          content: '[消息已撤回]',
          isRecalled: true
        }
      }
      
      // 替换会话对象，触发响应式更新
      conversations.value.splice(conversationIndex, 1, updatedConversation)
      
      // 强制触发响应式更新
      conversations.value = [...conversations.value]
    }
  }
}

// 处理新消息
const handleNewMessage = (data: any) => {
  const conversationId = data.conversation_id.toString()
  
  console.log('收到新消息:', data)
  console.log('新消息中的引用消息:', data.quoted_message)
  
  // 构建新消息对象
  let quotedMessageData = undefined
  if (data.quoted_message) {
    quotedMessageData = {
      id: data.quoted_message.id?.toString() || '',
      content: data.quoted_message.content || '',

      sender: data.quoted_message.sender ? {
        id: data.quoted_message.sender?.id?.toString() || '',
        name: data.quoted_message.sender?.nickname || data.quoted_message.sender?.username || data.quoted_message.sender?.name || '未知用户',
        avatar: data.quoted_message.sender?.avatar || ''
      } : {
        id: '',
        name: data.quoted_message.name || '未知用户',
        avatar: ''
      },
      timestamp: data.quoted_message.created_at ? new Date(data.quoted_message.created_at).getTime() : Date.now(),
      type: data.quoted_message.type || 'text',
      isSelf: data.quoted_message.sender?.id?.toString() === currentUser.value?.id?.toString()
    }
    console.log('构建的引用消息数据:', quotedMessageData)
  }
  
  // 使用 processMessage 函数处理新消息
  const newMessage = processMessage(data)
  
  console.log('构建的新消息对象:', newMessage)
  console.log('新消息是否包含引用消息:', !!newMessage.quotedMessage)
  
  // 更新会话列表中的未读计数和最后消息
  const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
  if (conversationIndex !== -1) {
    // 创建新的会话对象，确保响应式更新
    const updatedConversation = {
      ...conversations.value[conversationIndex],
      lastMessage: newMessage,
      timestamp: newMessage.timestamp
    }
    
    // 更新未读计数（如果不是当前会话）
    if (currentConversationId.value !== conversationId) {
      updatedConversation.unreadCount = (updatedConversation.unreadCount || 0) + 1
    }
    
    // 替换会话对象，触发响应式更新
    conversations.value.splice(conversationIndex, 1, updatedConversation)
    
    // 重新排序会话列表（按时间倒序）
    conversations.value.sort((a, b) => b.timestamp - a.timestamp)
    
    // 强制触发响应式更新
    conversations.value = [...conversations.value]
  }
  
  // 如果是当前会话的消息，添加到消息列表
  if (currentConversationId.value === conversationId) {
    // 检查消息是否已经存在，避免重复添加
    const messageExists = messages.value.some(msg => msg.id === newMessage.id)
    if (!messageExists) {
      messages.value.push(newMessage)
      console.log('消息列表长度:', messages.value.length)
      console.log('最后一条消息:', messages.value[messages.value.length - 1])
      
      // 滚动到底部
      nextTick(() => {
        const messageContainer = document.querySelector('.message-list')
        if (messageContainer) {
          messageContainer.scrollTop = messageContainer.scrollHeight
        }
      })
    }
  }
}

// 重新连接
const reconnect = () => {
  // 重置重连尝试次数
  reconnectAttempts.value = 0
  // 清除之前的定时器
  if (reconnectTimer.value) {
    clearTimeout(reconnectTimer.value)
    reconnectTimer.value = null
  }
  // 重新连接
  connectWebSocket()
}

// 跳转到登录页
const gotoLogin = () => {
  // 清除本地存储的token和用户信息
  localStorage.removeItem('token')
  localStorage.removeItem('user')
  // 跳转到登录页
  window.location.href = '/'
}

// 组件销毁时关闭WebSocket连接
onUnmounted(() => {
  if (ws) {
    ws.close()
  }
  // 清除重连定时器
  if (reconnectTimer.value) {
    clearTimeout(reconnectTimer.value)
  }
})

// 过滤后的会话列表
const filteredConversations = computed(() => {
  let filtered = conversations.value
  
  // 过滤掉已退出的群聊
  filtered = filtered.filter(conv => !conv.isExited)
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    filtered = filtered.filter(conv => 
      // 搜索会话名称
      conv.name.toLowerCase().includes(query) ||
      // 搜索最后一条消息的内容
      (conv.lastMessage?.content && conv.lastMessage.content.toLowerCase().includes(query)) ||
      // 搜索会话中的用户（针对群聊）
      (conv.members && conv.members.some(member => 
        member.name.toLowerCase().includes(query)
      )) ||
      // 搜索会话类型
      (conv.type === 'group' && '群聊'.includes(query)) ||
      (conv.type === 'single' && '用户'.includes(query))
    )
  }
  
  // 排序：置顶的会话排在前面，然后按时间戳降序
  return filtered.sort((a, b) => {
    if (a.pinned && !b.pinned) return -1
    if (!a.pinned && b.pinned) return 1
    return b.timestamp - a.timestamp
  })
})

// 当前选中的会话
const currentConversation = computed(() => {
  return conversations.value.find(conv => conv.id === currentConversationId.value)
})

// 消息数据
const messages = ref<Message[]>([])

// 已处理的已读回执，用于避免重复处理
const readReceiptsProcessed = ref<Set<string> | null>(null)

// 标记消息为已读
const markMessagesAsRead = async (conversationId: string) => {
  try {
    console.log('标记消息已读，conversationId:', conversationId)
    const url = `/api/v1/conversations/${conversationId}/read`
    console.log('请求URL:', url)
    const response = await request(url, {
      method: 'POST'
    })
    console.log('标记消息已读成功:', response)
  } catch (error) {
    console.error('标记消息已读失败:', error)
    console.error('错误详情:', error)
  }
}

// 处理消息数据，确保 sender 字段正确
const processMessage = (msg: any) => {
  const messageObj: any = {
    id: msg.id ? msg.id.toString() : '',
    content: msg.content || '',

    sender: msg.sender ? {
      id: msg.sender.id ? msg.sender.id.toString() : '',
      name: msg.sender.nickname || msg.sender.username || msg.sender.name || msg.sender.user?.nickname || msg.sender.user?.username || '',
      avatar: msg.sender.avatar || '',
      // 保存原始 sender 对象，以便在需要时访问更多属性
      user: msg.sender
    } : {
      id: '',
      name: '',
      avatar: ''
    },
    timestamp: msg.created_at ? new Date(msg.created_at).getTime() : Date.now(),
    type: msg.type || 'text',
    isSelf: msg.sender && msg.sender.id ? msg.sender.id.toString() === currentUser.value?.id?.toString() : false,
    isRead: msg.is_read || false,
    isRecalled: msg.is_recalled || false,
    quotedMessage: msg.quoted_message ? {
      id: msg.quoted_message.id?.toString() || '',
      content: msg.quoted_message.content || '',

      sender: msg.quoted_message.sender ? {
        id: msg.quoted_message.sender.id?.toString() || '',
        name: msg.quoted_message.sender?.nickname || msg.quoted_message.sender?.username || msg.quoted_message.sender?.name || msg.quoted_message.sender?.user?.nickname || msg.quoted_message.sender?.user?.username || '未知用户',
        avatar: msg.quoted_message.sender.avatar || ''
      } : {
        id: '',
        name: '未知用户',
        avatar: ''
      },
      timestamp: msg.quoted_message.created_at ? new Date(msg.quoted_message.created_at).getTime() : Date.now(),
      type: msg.quoted_message.type || 'text',
      isSelf: msg.quoted_message.sender?.id?.toString() === currentUser.value?.id?.toString()
    } : undefined,
  }
  
  // 处理分享消息（从content字段解析）
  if (msg.type === 'share' && msg.content) {
    try {
      // 尝试解析JSON
      const shareData = JSON.parse(msg.content)
      // 存储解析后的分享数据
      messageObj.shareData = shareData
    } catch (e) {
      // 如果解析失败，将原始内容作为分享数据
      messageObj.shareData = {
        type: 'text',
        content: msg.content
      }
    }
  }
  
  // 处理小程序消息
  if (msg.type === 'miniApp' && msg.content) {
    try {
      messageObj.miniAppData = JSON.parse(msg.content)
    } catch (e) {
      console.error('解析小程序数据失败:', e)
    }
  }
  
  // 处理资讯消息
  if (msg.type === 'news' && msg.content) {
    try {
      messageObj.newsData = JSON.parse(msg.content)
    } catch (e) {
      console.error('解析资讯数据失败:', e)
    }
  }
  
  // 处理引用消息中的文件类型
  if (messageObj.quotedMessage && messageObj.quotedMessage.type === 'file' && messageObj.quotedMessage.content) {
    try {
      const fileData = JSON.parse(messageObj.quotedMessage.content)
      // 对于文件类型的引用消息，我们只需要知道它是文件类型，不需要显示具体内容
    } catch (e) {
      // 如果解析失败，保持原样
    }
  }
  
  return messageObj
}

// 加载会话消息
// 分页参数
const messagePage = ref(1)
const messagePageSize = ref(20)
const hasMoreMessages = ref(true)
const isLoadingMessages = ref(false)

const loadMessages = async (conversationId: string, reset: boolean = true) => {
  if (isLoadingMessages.value) return
  
  if (reset) {
    messagePage.value = 1
    hasMoreMessages.value = true
  }
  
  if (!hasMoreMessages.value) return
  
  isLoadingMessages.value = true
  try {
    // 从服务器获取消息，添加分页参数
    const response = await request(`/api/v1/conversations/${conversationId}/messages?page=${messagePage.value}&page_size=${messagePageSize.value}`)
    if (response.code === 0) {
      // 确保 response.data 是一个数组，防止 null 或其他类型导致错误
      const serverMessages = Array.isArray(response.data) ? response.data.map((msg: any) => processMessage(msg)) : []
      
      // 保存当前滚动位置
      const messageListElement = document.querySelector('.message-list')
      let scrollTop = 0
      let initialHeight = 0
      if (messageListElement) {
        scrollTop = messageListElement.scrollTop
        initialHeight = messageListElement.scrollHeight
      }
      
      if (reset) {
        messages.value = serverMessages
      } else {
        messages.value = [...serverMessages, ...messages.value]
      }
      
      // 调整滚动位置，保持用户查看的内容不变
      setTimeout(() => {
        if (messageListElement) {
          const newHeight = messageListElement.scrollHeight
          const heightDiff = newHeight - initialHeight
          messageListElement.scrollTop = scrollTop + heightDiff
        }
      }, 0)
      
      // 处理分页信息
      if (response.pagination) {
        const { current_page, total_pages } = response.pagination
        // 检查是否还有更多消息
        hasMoreMessages.value = current_page < total_pages
        messagePage.value = current_page + 1
      } else {
        // 兼容旧版本，没有分页信息时的处理
        hasMoreMessages.value = serverMessages.length === messagePageSize.value
        messagePage.value++
      }
      
      // 重置未读消息计数
      const conversationIndex = conversations.value.findIndex(c => c.id === conversationId)
      if (conversationIndex !== -1) {
        conversations.value[conversationIndex].unreadCount = 0
      }
      
      // 标记消息为已读
      try {
        await markMessagesAsRead(conversationId)
      } catch (error) {
        console.error('标记消息已读失败:', error)
      }
    } else {
      if (reset) {
        messages.value = []
      }
      hasMoreMessages.value = false
    }
  } catch (error) {
    console.error('加载消息失败:', error)
    if (reset) {
      messages.value = []
    }
    hasMoreMessages.value = false
  } finally {
    isLoadingMessages.value = false
  }
}

// 获取消息的已读用户列表
const getMessageReadUsers = async (messageId: string) => {
  try {
    const response = await request(`/api/v1/messages/${messageId}/read-users`)
    if (response.code === 0) {
      return response.data
    }
    return { read_users: [], total_members: 0 }
  } catch (error) {
    console.error('获取已读用户列表失败:', error)
    return { read_users: [], total_members: 0 }
  }
}

// 播放消息提示音


// 发送消息
const handleSendMessage = async (messageData: any) => {
  if (!currentConversationId.value) return
  
  // 确保currentConversationId是字符串
  const conversationId = String(currentConversationId.value)
  
  // 检查是否为模拟会话
  if (conversationId.startsWith('conv_')) {
    showMessage({ message: '会话创建失败，请重试', type: 'error' })
    return
  }
  
  console.log('发送消息时的原始数据:', messageData)
  
  // 检查WebSocket连接状态
  const isWebSocketConnected = ws && ws.readyState === WebSocket.OPEN
  
  try {
    let requestData: any = {}
    let messageType = 'text'
    let messageContent = ''
    let miniAppData = null
    let newsData = null
    
    // 处理JSON字符串格式的消息数据（来自小程序和资讯消息）
    if (typeof messageData === 'string') {
      try {
        const parsedData = JSON.parse(messageData)
        if (parsedData.type === 'miniApp' && parsedData.data) {
          messageType = 'miniApp'
          messageContent = JSON.stringify(parsedData.data)
          miniAppData = parsedData.data
        } else if (parsedData.type === 'news' && parsedData.data) {
          messageType = 'news'
          messageContent = JSON.stringify(parsedData.data)
          newsData = parsedData.data
        } else {
          messageType = parsedData.type || 'text'
          messageContent = parsedData.content || messageData
        }
      } catch (e) {
        // 如果解析失败，当作普通文本消息处理
        messageType = 'text'
        messageContent = messageData
      }
    } else {
      // 处理对象格式的消息数据
      messageType = messageData.type || 'text'
      messageContent = messageData.content
      miniAppData = messageData.miniAppData
      newsData = messageData.newsData
    }
    
    // 准备请求参数
    requestData = {
      type: messageType,
      content: messageContent
    }
    
    // 只有当有引用消息时才添加quoted_message_id
    if (messageData.quotedMessage && messageData.quotedMessage.id) {
      requestData.quoted_message_id = parseInt(messageData.quotedMessage.id)
      console.log('添加引用消息ID:', requestData.quoted_message_id)
    }
    

    
    console.log('发送消息的请求数据:', requestData)
    
    // 如果WebSocket连接断开，直接标记消息为发送失败
    if (!isWebSocketConnected) {
      console.error('WebSocket连接已断开，消息发送失败')
      showMessage({ message: '网络连接已断开，消息发送失败', type: 'error' })
      
      // 创建发送失败的消息对象
      const failedMessage = {
        id: Date.now().toString(),
        content: messageContent,
        sender: {
          id: currentUser.value?.id?.toString() || '',
          name: currentUser.value?.nickname || currentUser.value?.username || '',
          avatar: currentUser.value?.avatar || ''
        },
        timestamp: new Date().getTime(),
        type: messageType,
        isSelf: true,
        isRead: false,
        isFailed: true,
        quotedMessage: messageData.quotedMessage,
        miniAppData: miniAppData,
        newsData: newsData,
        originalData: messageData // 保存原始消息数据，用于重新发送
      }
      
      console.log('添加发送失败的消息:', failedMessage)
      
      // 添加到消息列表
      messages.value.push(failedMessage)
      
      // 保存消息到本地存储
      storage.saveMessages(conversationId, messages.value)
      
      // 更新会话列表中的最后消息
      const conversationIndex = conversations.value.findIndex(c => c.id.toString() === conversationId)
      if (conversationIndex !== -1) {
        conversations.value[conversationIndex].lastMessage = failedMessage
        conversations.value[conversationIndex].timestamp = failedMessage.timestamp
        
        // 保存会话到本地存储
        storage.saveConversations(conversations.value)
      }
      
      return
    }
    
    const response = await request(`/api/v1/conversations/${conversationId}/messages`, {
      method: 'POST',
      body: JSON.stringify(requestData)
    })
    
    console.log('发送消息的响应:', response)
    
    if (response.code === 0) {
      // 直接使用客户端的引用消息数据，确保引用消息能正确显示
      const newMessage = {
        id: response.data.id?.toString() || Date.now().toString(),
        content: response.data.content,

        sender: {
          id: response.data.sender?.id?.toString() || currentUser.value?.id?.toString() || '',
          name: response.data.sender?.nickname || response.data.sender?.username || currentUser.value?.nickname || currentUser.value?.username || '',
          avatar: response.data.sender?.avatar || currentUser.value?.avatar || ''
        },
        timestamp: new Date().getTime(),
        type: response.data.type || messageType,
        isSelf: true,
        isRead: false,
        quotedMessage: messageData.quotedMessage,
        miniAppData: miniAppData,
        newsData: newsData
      }
      
      console.log('添加到消息列表的新消息:', newMessage)
      
      messages.value.push(newMessage)
      
      // 保存消息到本地存储
      storage.saveMessages(conversationId, messages.value)
      
      // 更新会话列表中的最后消息
      const conversationIndex = conversations.value.findIndex(c => c.id.toString() === conversationId)
      if (conversationIndex !== -1) {
        conversations.value[conversationIndex].lastMessage = newMessage
        conversations.value[conversationIndex].timestamp = newMessage.timestamp
        
        // 保存会话到本地存储
        storage.saveConversations(conversations.value)
      }
      
      // 播放消息发送成功的提示音
      // playMessageSound() // 暂时注释掉，因为该函数未定义
    } else {
      console.error('发送消息失败:', response.message)
      showMessage({ message: '消息发送失败: ' + response.message, type: 'error' })
      
      // 创建发送失败的消息对象
      const failedMessage = {
        id: Date.now().toString(),
        content: messageContent,
        sender: {
          id: currentUser.value?.id?.toString() || '',
          name: currentUser.value?.nickname || currentUser.value?.username || '',
          avatar: currentUser.value?.avatar || ''
        },
        timestamp: new Date().getTime(),
        type: messageType,
        isSelf: true,
        isRead: false,
        isFailed: true,
        quotedMessage: messageData.quotedMessage,
        miniAppData: miniAppData,
        newsData: newsData,
        originalData: messageData // 保存原始消息数据，用于重新发送
      }
      
      console.log('添加发送失败的消息:', failedMessage)
      
      // 添加到消息列表
      messages.value.push(failedMessage)
      
      // 保存消息到本地存储
      storage.saveMessages(String(currentConversationId.value), messages.value)
      
      // 更新会话列表中的最后消息
      const conversationIndex = conversations.value.findIndex(c => c.id.toString() === String(currentConversationId.value))
      if (conversationIndex !== -1) {
        conversations.value[conversationIndex].lastMessage = failedMessage
        conversations.value[conversationIndex].timestamp = failedMessage.timestamp
        
        // 保存会话到本地存储
        storage.saveConversations(conversations.value)
      }
    }
  } catch (error) {
    console.error('发送消息失败:', error)
    showMessage({ message: '网络错误，消息发送失败', type: 'error' })
    
    // 创建发送失败的消息对象
    let messageType = 'text'
    let messageContent = ''
    let miniAppData = null
    let newsData = null
    
    if (typeof messageData === 'string') {
      try {
        const parsedData = JSON.parse(messageData)
        if (parsedData.type === 'miniApp' && parsedData.data) {
          messageType = 'miniApp'
          messageContent = JSON.stringify(parsedData.data)
          miniAppData = parsedData.data
        } else if (parsedData.type === 'news' && parsedData.data) {
          messageType = 'news'
          messageContent = JSON.stringify(parsedData.data)
          newsData = parsedData.data
        } else {
          messageType = parsedData.type || 'text'
          messageContent = parsedData.content || messageData
        }
      } catch (e) {
        messageType = 'text'
        messageContent = messageData
      }
    } else {
      messageType = messageData.type || 'text'
      messageContent = messageData.content
      miniAppData = messageData.miniAppData
      newsData = messageData.newsData
    }
    
    const failedMessage = {
      id: Date.now().toString(),
      content: messageContent,

      sender: {
        id: currentUser.value?.id?.toString() || '',
        name: currentUser.value?.nickname || currentUser.value?.username || '',
        avatar: currentUser.value?.avatar || ''
      },
      timestamp: new Date().getTime(),
      type: messageType,
      isSelf: true,
      isRead: false,
      isFailed: true,
      quotedMessage: messageData.quotedMessage,
      miniAppData: miniAppData,
      newsData: newsData,
      originalData: messageData // 保存原始消息数据，用于重新发送
    }
    
    console.log('添加发送失败的消息:', failedMessage)
    
    // 添加到消息列表
    messages.value.push(failedMessage)
    
    // 保存消息到本地存储
    storage.saveMessages(String(currentConversationId.value), messages.value)
    
    // 更新会话列表中的最后消息
    const conversationIndex = conversations.value.findIndex(c => c.id.toString() === String(currentConversationId.value))
    if (conversationIndex !== -1) {
      conversations.value[conversationIndex].lastMessage = failedMessage
      conversations.value[conversationIndex].timestamp = failedMessage.timestamp
      
      // 保存会话到本地存储
      storage.saveConversations(conversations.value)
    }
  }
}

// 获取文件名
const getFileName = (message: any): string => {
  try {
    // 尝试解析content为JSON
    const contentObj = JSON.parse(message.content)
    if (contentObj.fileName) {
      return contentObj.fileName
    }
  } catch (e) {
    // 解析失败，从content字符串中提取文件名
  }
  return message.content.split('/').pop() || '文件'
}

// 处理消息撤回
const handleRecallMessage = async (messageId: number) => {
  try {
    // 更新本地消息状态
    const index = messages.value.findIndex(m => m.id === messageId.toString())
    if (index !== -1) {
      messages.value[index].content = '[消息已撤回]'
      messages.value[index].isRecalled = true
      
      // 保存消息到本地存储
      if (currentConversationId.value) {
        storage.saveMessages(currentConversationId.value, messages.value)
      }
    }
    
    // 更新会话列表中的最后消息
    if (currentConversationId.value) {
      const conversationIndex = conversations.value.findIndex(c => c.id === currentConversationId.value)
      if (conversationIndex !== -1) {
        const conversation = conversations.value[conversationIndex]
        if (conversation.lastMessage && conversation.lastMessage.id === messageId.toString()) {
          // 创建新的会话对象，确保响应式更新
          const updatedConversation = {
            ...conversation,
            lastMessage: {
              ...conversation.lastMessage,
              content: '[消息已撤回]',
              isRecalled: true
            }
          }
          
          // 替换会话对象，触发响应式更新
          conversations.value.splice(conversationIndex, 1, updatedConversation)
          
          // 强制触发响应式更新
          conversations.value = [...conversations.value]
          
          // 保存会话到本地存储
          storage.saveConversations(conversations.value)
        }
      }
    }
  } catch (error) {
    console.error('消息撤回失败:', error)
    showMessage({ message: '消息撤回失败，请稍后重试', type: 'error' })
  }
}

// 处理加载更多消息
const handleLoadMore = (conversationId: string) => {
  // 调用loadMessages函数加载更多消息，使用分页逻辑
  loadMessages(conversationId, false)
}

// 处理重新发送失败的消息
const handleRetrySendMessage = (failedMessage: any) => {
  console.log('重新发送失败消息:', failedMessage)
  
  // 从消息列表中移除失败的消息
  const messageIndex = messages.value.findIndex(msg => msg.id === failedMessage.id)
  if (messageIndex !== -1) {
    messages.value.splice(messageIndex, 1)
    // 保存更新后的消息列表
    if (currentConversationId.value) {
      storage.saveMessages(currentConversationId.value, messages.value)
    }
  }
  
  // 显示重新发送的提示
  showMessage({ message: '正在重新发送消息...', type: 'info' })
  
  // 使用原始消息数据重新发送
  if (failedMessage.originalData) {
    handleSendMessage(failedMessage.originalData)
  } else {
    // 如果没有原始数据，使用当前消息数据重新发送
    handleSendMessage(failedMessage)
  }
}

// 处理会话选择
const handleConversationSelect = (conversation: Conversation) => {
  // 切换到最近联系人选项卡
  activeOption.value = 'recent'
  currentConversationId.value = conversation.id
  loadMessages(conversation.id)
  // 重置未读消息计数
  const conversationIndex = conversations.value.findIndex(c => c.id === conversation.id)
  if (conversationIndex !== -1) {
    conversations.value[conversationIndex].unreadCount = 0
  }
}

// 格式化时间
const formatTime = (timestamp: number): string => {
  const date = new Date(timestamp)
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
    // 更早的消息，显示具体日期
    return date.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
  }
}

// 格式化消息预览
const formatMessagePreview = (message: any, conversation: any): string => {
  if (!message) {
    return '暂无消息'
  }
  
  let previewText = ''
  
  switch (message.type) {
    case 'text':
      previewText = message.content || '无内容'
      break
    case 'image':
      const imageName = getFileName(message) || '图片'
      previewText = `[图片] ${imageName}`
      break
    case 'file':
      const fileName = getFileName(message) || '文件'
      previewText = `[文件] ${fileName}`
      break
    case 'miniApp':
      if (message.miniAppData) {
        previewText = `[小程序] ${message.miniAppData.name || '小程序'}`
      } else {
        previewText = '[小程序]'
      }
      break
    case 'share':
      if (message.shareData) {
        const shareType = message.shareData.type === 'file' ? '文件' : message.shareData.type === 'note' ? '笔记' : message.shareData.type === 'sticky' ? '便签' : '分享'
        const shareName = message.shareData.name || message.content || '分享内容'
        previewText = `[${shareType}] ${shareName}`
      } else {
        previewText = '[分享]'
      }
      break
    default:
      previewText = message.content || '无内容'
  }
  
  // 群聊消息显示发送人名字
  if (conversation && message.sender) {
    console.log('Conversation type:', conversation.type)
    console.log('Message sender:', message.sender)
    if ((conversation.type === 'group' || conversation.type === 'Group' || conversation.type === 'GROUP')) {
      const senderName = message.sender.name || message.sender.nickname || message.sender.username || message.sender.user?.nickname || message.sender.user?.username
      console.log('Sender name:', senderName)
      if (senderName) {
        return `${senderName}: ${previewText}`
      }
    }
  }
  
  return previewText
}

// 获取搜索占位符
const getSearchPlaceholder = (): string => {
  switch (activeOption.value) {
    case 'recent':
      return '搜索会话'
    case 'org':
      return '搜索组织成员'
    case 'groups':
      return '搜索群聊'
    case 'apps':
      return '搜索应用'
    default:
      return '搜索'
  }
}

// 获取页面标题
const getPageTitle = (): string => {
  switch (activeOption.value) {
    case 'recent':
      return '最近会话'
    case 'org':
      return '组织架构'
    case 'groups':
      return '群聊'
    case 'channels':
      return '频道'
    case 'apps':
      return '应用'
    default:
      return 'QIM'
  }
}

// 开始私聊
const startPrivateChat = async (user: any) => {
  try {
    // 检查用户ID格式
    let userId = user.id
    
    // 确保userId是数字类型
    if (typeof userId === 'string') {
      // 如果是字符串格式（如 'emp1'），尝试提取数字部分
      if (userId.startsWith('emp')) {
        userId = userId.replace('emp', '')
      }
      // 转换为数字
      userId = parseInt(userId)
    }
    
    const response = await request('/api/v1/conversations/single', {
      method: 'POST',
      body: JSON.stringify({
        user_id: userId
      })
    })
    
    if (response.code === 0) {
      // 切换到最近联系人选项卡
      activeOption.value = 'recent'
      // 重新加载会话列表
      loadConversations()
      // 选择新创建的会话
      currentConversationId.value = response.data.id.toString()
      loadMessages(response.data.id.toString())
    }
  } catch (error) {
    console.error('创建私聊失败:', error)
    // 模拟创建会话（当API调用失败时）
    activeOption.value = 'recent'
    // 创建一个模拟的会话
    const mockConversation = {
      id: `conv_${Date.now()}`,
      name: user.name,
      avatar: user.avatar,
      lastMessage: null,
      unreadCount: 0,
      timestamp: Date.now(),
      type: 'single',
      members: [
        { id: currentUser.value?.id || 'me', name: currentUser.value?.nickname || currentUser.value?.username || '我', avatar: currentUser.value?.avatar || 'https://api.dicebear.com/7.x/avataaars/svg?seed=me' },
        { id: user.id, name: user.name, avatar: user.avatar }
      ]
    }
    // 添加到会话列表
    conversations.value.unshift(mockConversation)
    // 选择新创建的会话
    currentConversationId.value = mockConversation.id
    // 初始化消息列表
    messages.value = []
  }
  closeUserContextMenu()
}

// 语音通话相关
const showVoiceCallModal = ref(false)
const voiceCallStatus = ref('idle') // idle, calling, ringing, active, ended
const voiceCallDuration = ref(0)
const voiceCallTimer = ref<number | null>(null)

// 发起语音通话
const startVoiceCall = async (userId: string) => {
  try {
    voiceCallStatus.value = 'calling'
    showVoiceCallModal.value = true
    
    // 模拟语音通话连接
    setTimeout(() => {
      voiceCallStatus.value = 'ringing'
    }, 1000)
    
    // 模拟对方接听
    setTimeout(() => {
      voiceCallStatus.value = 'active'
      startVoiceCallTimer()
    }, 3000)
  } catch (error) {
    console.error('发起语音通话失败:', error)
    voiceCallStatus.value = 'ended'
    showMessage({ message: '发起语音通话失败', type: 'error' })
  }
}

// 开始语音通话计时器
const startVoiceCallTimer = () => {
  voiceCallDuration.value = 0
  voiceCallTimer.value = window.setInterval(() => {
    voiceCallDuration.value++
  }, 1000)
}

// 结束语音通话
const endVoiceCall = () => {
  if (voiceCallTimer.value) {
    clearInterval(voiceCallTimer.value)
    voiceCallTimer.value = null
  }
  voiceCallStatus.value = 'ended'
  setTimeout(() => {
    showVoiceCallModal.value = false
    voiceCallStatus.value = 'idle'
    voiceCallDuration.value = 0
  }, 1000)
}

// 格式化通话时长
const formatCallDuration = (seconds: number): string => {
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = seconds % 60
  return `${minutes.toString().padStart(2, '0')}:${remainingSeconds.toString().padStart(2, '0')}`
}



// 右键菜单相关
const showMenu = ref(false)
const menuPosition = ref({ x: 0, y: 0 })
const selectedConversation = ref<Conversation | null>(null)

const showContextMenu = (event: MouseEvent, conversation: Conversation) => {
  event.preventDefault()
  showMenu.value = true
  menuPosition.value = { x: event.clientX, y: event.clientY }
  selectedConversation.value = conversation
  
  // 点击其他地方关闭菜单
  setTimeout(() => {
    document.addEventListener('click', closeContextMenu)
  }, 0)
}

const closeContextMenu = () => {
  showMenu.value = false
  selectedConversation.value = null
  document.removeEventListener('click', closeContextMenu)
}





// 动作菜单
const showActionMenuFlag = ref(false)
const actionMenuPosition = ref({ x: 0, y: 0 })



// 用户右键菜单
const showUserContextMenuFlag = ref(false)
const userContextMenuPosition = ref({ x: 0, y: 0 })
const selectedEmployee = ref<any>(null)

const showUserContextMenu = (event: MouseEvent, employee: any) => {
  event.preventDefault()
  showUserContextMenuFlag.value = true
  userContextMenuPosition.value = { x: event.clientX, y: event.clientY }
  selectedEmployee.value = employee
  
  // 点击其他地方关闭菜单
  setTimeout(() => {
    document.addEventListener('click', closeUserContextMenu)
  }, 0)
}

const closeUserContextMenu = () => {
  showUserContextMenuFlag.value = false
  selectedEmployee.value = null
  document.removeEventListener('click', closeUserContextMenu)
}

const viewUserProfile = () => {
  if (selectedEmployee.value) {
    selectedUser.value = selectedEmployee.value
    showUserProfile.value = true
    console.log('查看用户资料:', selectedEmployee.value)
  }
  closeUserContextMenu()
}

// 触发头像选择
const triggerAvatarInput = () => {
  const input = document.querySelector('.avatar-input') as HTMLInputElement
  if (input) {
    input.click()
  }
}

// 处理头像变化
const handleAvatarChange = async (event: Event) => {
  const input = event.target as HTMLInputElement
  if (input.files && input.files.length > 0) {
    const file = input.files[0]
    
    // 验证文件类型
    if (!file.type.startsWith('image/')) {
      showMessage({ message: '请选择图片文件', type: 'error' })
      return
    }
    
    // 验证文件大小
    if (file.size > 5 * 1024 * 1024) { // 5MB限制
      showMessage({ message: '图片大小不能超过5MB', type: 'error' })
      return
    }
    
    try {
      // 创建FormData
      const formData = new FormData()
      formData.append('file', file)
      
      // 上传文件
      const response = await request('/api/v1/upload', {
        method: 'POST',
        headers: {
          // 注意：FormData不需要设置Content-Type
        },
        body: formData
      })
      
      if (response.code === 0 && response.data && response.data.url) {
        // 更新用户头像
        const updateResponse = await request('/api/v1/users/me', {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            avatar: response.data.url
          })
        })
        
        if (updateResponse.code === 0 && updateResponse.data) {
          // 更新当前用户信息
          if (currentUser.value) {
            currentUser.value.avatar = updateResponse.data.avatar || response.data.url
            // 更新本地存储中的用户信息
            localStorage.setItem('user', JSON.stringify(currentUser.value))
          }
          showMessage({ message: '头像更新成功', type: 'success' })
        } else {
          showMessage({ message: '头像更新失败: ' + updateResponse.message, type: 'error' })
        }
      } else {
        showMessage({ message: '文件上传失败: ' + response.message, type: 'error' })
      }
    } catch (error) {
      console.error('头像上传失败:', error)
      showMessage({ message: '头像上传失败: ' + error.message, type: 'error' })
    }
  }
}

// 保存用户资料
const saveUserProfile = async () => {
  try {
    const response = await request('/api/v1/users/me', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        nickname: userProfile.value.nickname,
        signature: userProfile.value.signature
      })
    })
    
    if (response.code === 0) {
      // 更新当前用户信息
      if (currentUser.value) {
        currentUser.value.nickname = userProfile.value.nickname
      }
      showMessage({ message: '保存成功', type: 'success' })
      closeUserProfile()
    } else {
      showMessage({ message: '保存失败: ' + response.message, type: 'error' })
    }
  } catch (error) {
    console.error('保存用户资料失败:', error)
    showMessage({ message: '保存失败: ' + error.message, type: 'error' })
  }
}









// 切换部门展开/折叠
const toggleDepartment = (departmentId: string) => {
  const index = expandedDepartments.value.indexOf(departmentId)
  if (index > -1) {
    expandedDepartments.value.splice(index, 1)
  } else {
    expandedDepartments.value.push(departmentId)
  }
}

// 切换子部门展开/折叠
const toggleSubDepartment = (departmentId: string, subDepartmentId: string) => {
  if (!expandedSubDepartments.value[departmentId]) {
    expandedSubDepartments.value[departmentId] = []
  }
  const index = expandedSubDepartments.value[departmentId].indexOf(subDepartmentId)
  if (index > -1) {
    expandedSubDepartments.value[departmentId].splice(index, 1)
  } else {
    expandedSubDepartments.value[departmentId].push(subDepartmentId)
  }
}

// 系统消息数据
const systemMessage = ref({
  title: '',
  content: '',
  target: 'all',
  groupId: '',
  userId: ''
})

// 显示系统消息发布模态框
const showSystemMessageModal = ref(false)

// 创建会话相关状态
const showCreateConversationModal = ref(false)
const createConversationType = ref('group')
const createConversationTitle = ref('创建群聊')

// 添加成员相关
const showAddMembersModal = ref(false)
const addMembersSearchQuery = ref('')
const selectedAddMembers = ref<any[]>([])

// 组织架构数据
const orgStructure = ref([
  {
    id: 'company',
    name: '总公司',
    subDepartments: [
      {
        id: 'dept1',
        name: '技术部',
        subDepartments: [
          {
            id: 'subdept1-1',
            name: '前端开发',
            employees: [
              { id: 'emp1', name: '张三', position: '前端工程师', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=1', status: 'online' },
              { id: 'emp2', name: '李四', position: '高级前端工程师', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=2', status: 'online' },
              { id: 'emp3', name: '王五', position: '前端实习生', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=3', status: 'offline' }
            ]
          },
          {
            id: 'subdept1-2',
            name: '后端开发',
            employees: [
              { id: 'emp4', name: '赵六', position: '后端工程师', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=4', status: 'online' },
              { id: 'emp5', name: '钱七', position: '高级后端工程师', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=5', status: 'offline' }
            ]
          },
          {
            id: 'subdept1-3',
            name: '测试',
            employees: [
              { id: 'emp6', name: '孙八', position: '测试工程师', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=6', status: 'online' }
            ]
          }
        ]
      },
      {
        id: 'dept2',
        name: '产品部',
        subDepartments: [
          {
            id: 'subdept2-1',
            name: '产品设计',
            employees: [
              { id: 'emp7', name: '周九', position: '产品经理', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=7', status: 'online' },
              { id: 'emp8', name: '吴十', position: 'UI设计师', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=8', status: 'offline' }
            ]
          }
        ]
      },
      {
        id: 'dept3',
        name: '市场部',
        subDepartments: [
          {
            id: 'subdept3-1',
            name: '市场营销',
            employees: [
              { id: 'emp9', name: '郑一', position: '市场专员', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=9' }
            ]
          },
          {
            id: 'subdept3-2',
            name: '销售',
            employees: [
              { id: 'emp10', name: '王二', position: '销售经理', avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=10', status: 'offline' }
            ]
          }
        ]
      }
    ]
  }
])

// 展开的部门
const expandedDepartments = ref<string[]>(['company', 'dept1'])

// 展开的子部门
const expandedSubDepartments = ref<Record<string, string[]>>({
  dept1: ['subdept1-1', 'subdept1-2', 'subdept1-3'],
  dept2: ['subdept2-1'],
  dept3: ['subdept3-1', 'subdept3-2'],
  'subdept1-1': ['subdept1-1-1'],
  'subdept1-2': ['subdept1-2-1'],
  'subdept1-3': ['subdept1-3-1']
})

// 应用相关数据
// 从本地存储加载最近使用的应用
const loadRecentApps = () => {
  try {
    const storedRecentApps = localStorage.getItem('recentApps')
    if (storedRecentApps) {
      return JSON.parse(storedRecentApps)
    }
  } catch (error) {
    console.error('加载最近使用的应用失败:', error)
  }
  // 默认最近使用的应用
  return [
    { id: '1', name: '统计报表', icon: 'fas fa-chart-bar' },
    { id: '2', name: '日历', icon: 'fas fa-calendar' },
    { id: '5', name: '任务管理', icon: 'fas fa-check-square' }
  ]
}

const recentApps = ref(loadRecentApps())

// 所有应用列表（包括内置应用、外链应用和自定义应用）
const allApps = computed(() => {
  const apps: any[] = []
  
  // 遍历所有应用分类，收集所有应用
  appCategories.value.forEach(category => {
    category.apps.forEach(app => {
      apps.push(app)
    })
  })
  
  return apps
})

// 加载用户创建的应用
const loadUserApps = async () => {
  try {
    const token = localStorage.getItem('token')
    const serverUrl = localStorage.getItem('serverUrl') || API_BASE_URL
    const response = await axios.get(`${serverUrl}/api/v1/apps`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (response.data.code === 0) {
      const userApps = response.data.data
      // 找到自定义应用分类
      const customCategory = appCategories.value.find(cat => cat.id === '3')
      if (customCategory) {
        // 清空现有的自定义应用
        customCategory.apps = []
        // 添加用户创建的应用
        userApps.forEach((app: any) => {
          customCategory.apps.push({
            id: 'user-' + app.id.toString(),
            name: app.name,
            icon: app.icon,
            url: app.url,
            openType: app.open_type || app.openType || 'in-app' // 默认为在应用内打开
          })
        })
      }
    }
  } catch (error) {
    console.error('加载用户应用失败:', error)
  }
}

const appCategories = ref([
  {
    id: '1',
    name: '内置应用',
    expanded: true,
    apps: [
      { id: '1', name: '统计报表', icon: 'fas fa-chart-bar' },
      { id: '2', name: '日历', icon: 'fas fa-calendar' },
      { id: '3', name: '文件管理', icon: 'fas fa-folder' },
      { id: '5', name: '任务管理', icon: 'fas fa-check-square' },
      { id: '6', name: '便签', icon: 'fas fa-sticky-note' },
      { id: '7', name: '笔记', icon: 'fas fa-book' },
      { id: 'ai-assistant', name: 'AI 助手', icon: 'fas fa-robot' },
      { id: 'short-link', name: '短链接管理', icon: 'fas fa-link' }
    ]
  },
  {
    id: '2',
    name: '外链应用',
    expanded: false,
    apps: [
      { id: '10', name: 'GitHub', icon: 'fab fa-github', url: 'https://github.com' }
    ]
  },
  {
    id: '3',
    name: '自定义应用',
    expanded: false,
    apps: [
      // 这里可以添加用户自定义的应用
    ]
  },
  {
    id: '4',
    name: '应用管理',
    expanded: false,
    apps: [
      { id: 'app-management', name: '管理应用', icon: 'fas fa-cog' }
    ]
  }
])



// 应用管理相关状态
const apps = ref<any[]>([])
const showAppModal = ref(false)
const editingApp = ref<any>(null)
const newApp = ref({
  name: '',
  icon: 'fas fa-cube',
  url: '',
  categoryId: ''
})

// 应用管理相关函数
const loadApps = async () => {
  try {
    const token = getToken()
    const response = await axios.get(`${serverUrl.value}/api/v1/apps`, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (response.data.code === 0) {
      apps.value = response.data.data
    }
  } catch (error) {
    console.error('加载应用失败:', error)
    ElMessage.error('加载应用失败，请稍后重试')
  }
}

const createApp = async () => {
  try {
    const token = getToken()
    const response = await axios.post(`${serverUrl.value}/api/v1/apps`, newApp.value, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (response.data.code === 0) {
      apps.value.push(response.data.data)
      closeAppModal()
      showMessage({ message: '应用创建成功', type: 'success' })
    }
  } catch (error) {
    console.error('创建应用失败:', error)
    // 模拟创建应用
    const mockApp = {
      id: `app_${Date.now()}`,
      name: newApp.value.name,
      icon: newApp.value.icon,
      url: newApp.value.url,
      categoryId: newApp.value.categoryId
    }
    apps.value.push(mockApp)
    closeAppModal()
    showMessage({ message: '应用创建成功', type: 'success' })
  }
}

const updateApp = async () => {
  try {
    const token = getToken()
    const response = await axios.put(`${serverUrl.value}/api/v1/apps/${editingApp.value.id}`, editingApp.value, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (response.data.code === 0) {
      const index = apps.value.findIndex(app => app.id === editingApp.value.id)
      if (index !== -1) {
        apps.value[index] = response.data.data
      }
      closeAppModal()
      showMessage({ message: '应用更新成功', type: 'success' })
    }
  } catch (error) {
    console.error('更新应用失败:', error)
    // 模拟更新应用
    const index = apps.value.findIndex(app => app.id === editingApp.value.id)
    if (index !== -1) {
      apps.value[index] = editingApp.value
    }
    closeAppModal()
    showMessage({ message: '应用更新成功', type: 'success' })
  }
}

const deleteApp = async (appId: string) => {
  if (confirm('确定要删除这个应用吗？')) {
    try {
      const token = getToken()
      const response = await axios.delete(`${serverUrl.value}/api/v1/apps/${appId}`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      })
      if (response.data.code === 0) {
        apps.value = apps.value.filter(app => app.id !== appId)
        showMessage({ message: '应用删除成功', type: 'success' })
      }
    } catch (error) {
      console.error('删除应用失败:', error)
      // 模拟删除应用
      apps.value = apps.value.filter(app => app.id !== appId)
      showMessage({ message: '应用删除成功', type: 'success' })
    }
  }
}

const openAppModal = (app?: any) => {
  if (app) {
    editingApp.value = { ...app }
  } else {
    newApp.value = {
      name: '',
      icon: 'fas fa-cube',
      url: '',
      categoryId: '1'
    }
    editingApp.value = null
  }
  showAppModal.value = true
}

const closeAppModal = () => {
  showAppModal.value = false
  editingApp.value = null
  newApp.value = {
    name: '',
    icon: 'fas fa-cube',
    url: '',
    categoryId: '1'
  }
}

// 切换应用分类展开/折叠
const toggleCategory = (categoryId: string) => {
  const category = appCategories.value.find(c => c.id === categoryId)
  if (category) {
    category.expanded = !category.expanded
  }
}

// 选中的应用ID
const selectedAppId = ref('')

// 当前打开的用户应用
const currentUserApp = ref<any>(null)

// 应用面板的tab切换
const activeAppTab = ref('categories')

// 笔记数据


// 打开应用
// 记录最近使用的应用
const addToRecentApps = (appId: string, appName: string, appIcon: string) => {
  // 从最近使用列表中移除已存在的该应用
  recentApps.value = recentApps.value.filter(app => app.id !== appId)
  
  // 将应用添加到最近使用列表的开头
  recentApps.value.unshift({ id: appId, name: appName, icon: appIcon })
  
  // 限制最近使用的应用数量为5个
  if (recentApps.value.length > 5) {
    recentApps.value = recentApps.value.slice(0, 5)
  }
  
  // 保存到本地存储
  localStorage.setItem('recentApps', JSON.stringify(recentApps.value))
}

const openApp = async (appId: string) => {
  console.log('打开应用:', appId)
  
  // 查找应用信息
  let appName = ''
  let appIcon = ''
  let appUrl = ''
  let openType = 'in-app' // 默认为在应用内打开
  
  // 从应用分类中查找应用
  let foundApp: any = null
  for (const category of appCategories.value) {
    const app = category.apps.find(a => a.id === appId)
    if (app) {
      foundApp = app
      appName = app.name
      appIcon = app.icon
      appUrl = app.url || ''
      openType = app.openType || 'in-app'
      break
    }
  }
  
  // 记录最近使用的应用
  if (appName && appIcon) {
    addToRecentApps(appId, appName, appIcon)
  }
  
  // 特殊处理短链接应用
  if (appId === 'short-link') {
    console.log('打开短链接管理应用')
    selectedAppId.value = 'short-link'
    return
  }
  
  // 检查应用是否有URL
  if (appUrl) {
    console.log('打开带URL的应用:', appName, appUrl, 'openType:', openType)
    
    // 根据openType决定如何打开应用
    if (openType === 'external') {
      // 使用默认浏览器打开
      console.log('使用默认浏览器打开应用:', appUrl)
      if (typeof window !== 'undefined') {
        try {
          // 检查是否在Electron环境中
          if (window.electron && window.electron.shell && typeof window.electron.shell.openExternal === 'function') {
            console.log('使用Electron shell.openExternal打开链接（系统默认浏览器）')
            window.electron.shell.openExternal(appUrl)
          } else {
            // 在普通浏览器环境中，使用window.open
            console.log('使用window.open打开链接')
            window.open(appUrl, '_blank', 'noopener,noreferrer')
          }
        } catch (error) {
          console.error('打开外部应用失败:', error)
          // 作为后备，使用window.open在新窗口打开
          window.open(appUrl, '_blank', 'noopener,noreferrer')
        }
      }
    } else {
      // 在应用内打开
      console.log('在应用内打开:', appName, appUrl)
      selectedAppId.value = 'user-app'
      currentUserApp.value = {
        id: appId,
        name: appName,
        icon: appIcon,
        url: appUrl
      }
      console.log('设置selectedAppId:', selectedAppId.value)
      console.log('设置currentUserApp:', currentUserApp.value)
    }
  } else {
    // 没有URL的应用，按原来的方式处理
    selectedAppId.value = appId
    
    // 数据加载由各独立应用组件内部处理
  }
}

// 打开用户创建的应用
const openUserApp = (app: any) => {
  console.log('打开用户创建的应用:', app)
  selectedAppId.value = 'user-app'
  currentUserApp.value = app
  
  // 记录最近使用的应用
  if (app.name && app.icon) {
    addToRecentApps(app.id, app.name, app.icon)
  }
}

// 监听打开用户应用的事件
window.addEventListener('open-user-app', (event: any) => {
  const app = event.detail
  openUserApp(app)
})

// 打开外部应用
const openExternalApp = (url: string) => {
  console.log('打开外部链接:', url)
  
  // 查找外部应用信息
  let appName = ''
  let appIcon = ''
  
  // 从应用分类中查找外部应用
  for (const category of appCategories.value) {
    const app = category.apps.find(a => a.url === url)
    if (app) {
      appName = app.name
      appIcon = app.icon
      break
    }
  }
  
  // 记录最近使用的应用
  if (appName && appIcon) {
    addToRecentApps(url, appName, appIcon)
  }
  
  // 尝试使用系统默认浏览器打开链接
  if (typeof window !== 'undefined') {
    try {
      // 检查是否在Electron环境中
      if (window.electron && window.electron.shell && typeof window.electron.shell.openExternal === 'function') {
        console.log('使用Electron shell.openExternal打开链接（系统默认浏览器）')
        window.electron.shell.openExternal(url)
      } else {
        // 在非Electron环境中，使用新窗口打开
        console.log('使用window.open打开链接（新窗口）')
        window.open(url, '_blank', 'noopener,noreferrer')
      }
    } catch (error) {
      console.error('打开外部链接失败:', error)
      // 出错时回退到使用新窗口打开
      window.open(url, '_blank', 'noopener,noreferrer')
    }
  }
}

// 创建新笔记


// 返回应用列表
const backToAppList = () => {
  selectedAppId.value = ''
}



// 打开应用管理











// 获取所有员工
const allEmployees = computed(() => {
  const employees = []
  
  // 递归收集员工
  const collectEmployees = (departments) => {
    departments.forEach(dept => {
      // 收集当前部门的员工
      if (dept.employees) {
        employees.push(...dept.employees)
      }
      // 递归处理子部门
      if (dept.subDepartments) {
        collectEmployees(dept.subDepartments)
      }
    })
  }
  
  collectEmployees(orgStructure.value)
  return employees
})

// 过滤可添加的成员列表
const filteredAddMembersEmployees = computed(() => {
  if (!addMembersSearchQuery.value) {
    return allEmployees.value
  }
  const query = addMembersSearchQuery.value.toLowerCase()
  return allEmployees.value.filter(employee => 
    employee.name.toLowerCase().includes(query)
  )
})

// 成员上下文菜单
const showMemberContextMenuFlag = ref(false)
const memberContextMenuPosition = ref({ x: 0, y: 0 })
const selectedMember = ref(null)

// 群聊上下文菜单
const showGroupContextMenuFlag = ref(false)
const groupContextMenuPosition = ref({ x: 0, y: 0 })

// 群成员模态框
const showGroupMembersModal = ref(false)
const groupMembers = ref([])

// 群资料模态框
const showGroupInfoModal = ref(false)

// 群公告
const showEditAnnouncementModal = ref(false)
const editAnnouncementContent = ref('')

// 设置菜单
const showSettingsMenuFlag = ref(false)
const settingsMenuPosition = ref({ x: 0, y: 0 })

// 主题菜单相关
const showThemeMenuFlag = ref(false)
const themeMenuPosition = ref({ x: 0, y: 0 })

// 更多菜单状态
const showMoreMenuFlag = ref(false)
const moreMenuPosition = ref({ x: 0, y: 0 })

// 主题名称映射 - 将旧主题名称映射到新主题名称
const themeNameMap: Record<string, string> = {
  'light': 'modern-light',
  'dark': 'elegant-dark',
  'netblue': 'ocean-blue',
  'elegantpurple': 'elegant-purple',
  'sacredyellow': 'warm-amber',
  'chinesered': 'crimson-red',
  'grassgreen': 'emerald-green'
}

// 获取并转换当前主题
const savedTheme = localStorage.getItem('theme') || 'light'
const currentTheme = ref(themeNameMap[savedTheme] || savedTheme)

// 系统设置相关
const showSettingsModal = ref(false)
const activeSettingsTab = ref('basic')
const settingsProfile = ref({
  nickname: currentUser.value?.nickname || currentUser.value?.username || '我的账号',
  signature: '这个人很懒，什么都没留下'
})
const messageSettings = ref({
  notificationsEnabled: true,
  soundEnabled: true,
  desktopNotificationsEnabled: true,
  dndMode: 'none'
})
const appearanceSettings = ref({
  theme: currentTheme.value,
  fontSize: 14
})

// 高级设置
const advancedSettings = ref({
  twoFactorEnabled: currentUser.value?.two_factor_enabled || false
})

// 文件设置
const fileSettings = ref({
  defaultSaveDirectory: '',
  autoDownload: false,
  maxFileSize: 50,
  allowedFileTypes: 'jpg,png,gif,pdf,doc,docx,xls,xlsx,ppt,pptx,zip,rar',
  autoPreviewImages: true,
  enableFileHistory: true
})

// 关于对话框
const showAboutDialog = ref(false)

const showActionMenu = (event: MouseEvent) => {
  event.stopPropagation()
  
  // 切换动作菜单显示状态
  if (showActionMenuFlag.value) {
    closeActionMenu()
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
  
  actionMenuPosition.value = {
    x,
    y
  }
  showActionMenuFlag.value = true
  
  // 点击其他地方关闭菜单
  setTimeout(() => {
    document.addEventListener('click', closeActionMenu)
  }, 0)
}

const closeActionMenu = () => {
  showActionMenuFlag.value = false
  document.removeEventListener('click', closeActionMenu)
}

// 打开创建群聊弹窗
const openCreateGroupModal = () => {
  closeActionMenu()
  createConversationType.value = 'group'
  createConversationTitle.value = '创建群聊'
  showCreateConversationModal.value = true
}

// 打开创建讨论组弹窗
const createDiscussionGroup = () => {
  closeActionMenu()
  createConversationType.value = 'discussion'
  createConversationTitle.value = '创建讨论组'
  showCreateConversationModal.value = true
}

// 关闭创建会话弹窗
const closeCreateConversationModal = () => {
  showCreateConversationModal.value = false
}

// 处理会话创建成功
const handleConversationCreated = () => {
  // 重新加载会话列表
  loadConversations()
}

// 打开系统消息发布模态框
const openSystemMessageModal = () => {
  systemMessage.value = {
    title: '',
    content: '',
    target: 'all',
    groupId: '',
    userId: ''
  }
  showSystemMessageModal.value = true
}

// 关闭系统消息发布模态框
const closeSystemMessageModal = () => {
  showSystemMessageModal.value = false
  systemMessage.value = {
    title: '',
    content: '',
    target: 'all',
    groupId: '',
    userId: ''
  }
}

// 发送系统消息
const sendSystemMessage = async () => {
  if (!systemMessage.value.title || !systemMessage.value.content) return
  
  try {
    const token = getToken()
    const response = await axios.post(`${serverUrl.value}/api/v1/system-messages`, systemMessage.value, {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      }
    })
    
    if (response.data.code === 0) {
      showMessage({ message: '系统消息发布成功', type: 'success' })
      closeSystemMessageModal()
    } else {
      showMessage({ message: '系统消息发布失败: ' + response.data.message, type: 'error' })
    }
  } catch (error) {
    console.error('发布系统消息失败:', error)
    showMessage({ message: '系统消息发布失败', type: 'error' })
  }
}





const createChannel = () => {
  closeActionMenu()
  // 这里可以实现创建频道的逻辑
  ElMessage.info('创建频道功能开发中...')
  console.log('创建频道')
}

const showMemberContextMenu = (event: MouseEvent, member: any) => {
  event.stopPropagation()
  memberContextMenuPosition.value = {
    x: event.clientX,
    y: event.clientY
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

const removeMemberFromGroup = async () => {
  if (selectedMember.value && selectedGroup.value) {
    try {
      const response = await request(`/api/v1/conversations/${selectedGroup.value.id}/members/${selectedMember.value.id}`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json'
        }
      })
      
      if (response.code === 0) {
        ElMessage.success('成员已成功移除')
        // 从本地群聊成员列表中移除
        const index = selectedGroup.value.members.findIndex(member => member.id === selectedMember.value.id)
        if (index > -1) {
          selectedGroup.value.members.splice(index, 1)
        }
      } else {
        ElMessage.error(response.message || '移除成员失败')
      }
    } catch (error) {
      console.error('移除成员失败:', error)
      ElMessage.error('网络错误，移除成员失败')
    }
  }
  closeMemberContextMenu()
}

const viewMemberInfo = () => {
  if (selectedMember.value) {
    ElMessage.info(`查看${selectedMember.value.name}的资料`)
    console.log('查看成员资料:', selectedMember.value)
  }
  closeMemberContextMenu()
}

const setAsAdmin = async () => {
  if (selectedMember.value && selectedGroup.value) {
    try {
      const response = await request(`/api/v1/conversations/${selectedGroup.value.id}/members/${selectedMember.value.id}/role`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ role: 'admin' })
      })
      
      if (response.code === 0) {
        ElMessage.success('已成功设为管理员')
        // 更新本地成员角色
        const member = selectedGroup.value.members.find(m => m.id === selectedMember.value.id)
        if (member) {
          member.role = 'admin'
        }
      } else {
        ElMessage.error(response.message || '设置管理员失败')
      }
    } catch (error) {
      console.error('设置管理员失败:', error)
      ElMessage.error('网络错误，设置管理员失败')
    }
  }
  closeMemberContextMenu()
}

const showGroupContextMenu = (event: MouseEvent, group: any) => {
  event.stopPropagation()
  
  // 计算菜单位置，确保在屏幕内显示
  const menuWidth = 180 // 菜单宽度
  const menuHeight = 160 // 菜单高度
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
  
  groupContextMenuPosition.value = {
    x,
    y
  }
  selectedGroup.value = group
  showGroupContextMenuFlag.value = true
  
  // 点击其他地方关闭菜单
  setTimeout(() => {
    document.addEventListener('click', closeGroupContextMenu)
  }, 0)
}

const closeGroupContextMenu = () => {
  showGroupContextMenuFlag.value = false
  document.removeEventListener('click', closeGroupContextMenu)
}

const viewGroupMembers = () => {
  if (selectedGroup.value) {
    // 映射成员数据，确保使用昵称而不是账号
    groupMembers.value = (selectedGroup.value.members || []).map((member: any) => ({
      id: member.user && member.user.id ? member.user.id.toString() : (member.id ? member.id.toString() : ''),
      name: member.user ? (member.user.nickname || member.user.username || '') : (member.name || ''),
      avatar: member.user ? (
        member.user.avatar && member.user.avatar.startsWith('http')
          ? member.user.avatar
          : (member.user.avatar ? serverUrl.value + member.user.avatar : '')
      ) : (member.avatar || ''),
      position: member.user ? (member.user.position || '无职位信息') : (member.position || '无职位信息')
    }))
    showGroupMembersModal.value = true
  }
  closeGroupContextMenu()
  selectedGroup.value = null
}

const viewGroupInfo = () => {
  if (selectedGroup.value) {
    // 显示群资料模态框
    showGroupInfoModal.value = true
  }
  closeGroupContextMenu()
  selectedGroup.value = null
}

const addMembersToGroup = () => {
  if (selectedGroup.value) {
    // 重置选择
    selectedAddMembers.value = []
    addMembersSearchQuery.value = ''
    // 打开添加成员模态框
    showAddMembersModal.value = true
    // 关闭群聊上下文菜单
    closeGroupContextMenu()
  }
}

// 处理邀请成员
const handleInviteMembers = (groupOrId) => {
  let group = null
  // 处理传递的是ID的情况（来自ChatWindow）
  if (typeof groupOrId === 'string') {
    group = conversations.value.find(c => c.id === groupOrId)
  } else {
    // 处理传递的是完整group对象的情况（来自GroupDetail）
    group = groupOrId
  }
  
  if (group) {
    selectedGroup.value = group
    // 重置选择
    selectedAddMembers.value = []
    addMembersSearchQuery.value = ''
    // 打开添加成员模态框
    showAddMembersModal.value = true
  }
}

// 处理切换应用
const handleSwitchApp = (app) => {
  // 切换到指定的应用
  activeOption.value = 'apps'
  selectedAppId.value = app
  console.log('切换到应用:', app)
}

// 处理切换会话
const handleSwitchConversation = async (conversationId) => {
  // 切换到最近联系人选项卡
  activeOption.value = 'recent'
  // 重新加载会话列表
  await loadConversations()
  // 选择新会话
  currentConversationId.value = conversationId
  // 加载新会话的消息
  await loadMessages(conversationId)
}

// 打开分享弹窗
const openShareModal = (type, data) => {
  shareType.value = type
  window.shareData = data // 临时存储分享数据
  
  // 加载可分享的用户和群聊
  loadShareUsersAndGroups()
  
  showShareModal.value = true
}

// 关闭分享弹窗
const closeShareModal = () => {
  showShareModal.value = false
  shareType.value = ''
  window.shareData = null // 清除临时分享数据
}

// 加载可分享的用户和群聊
const loadShareUsersAndGroups = async () => {
  try {
    // 加载组织架构中的用户
    const orgResponse = await request('/api/v1/organization/tree')
    if (orgResponse.code === 0) {
      const users = []
      
      // 递归提取所有用户
      const extractUsers = (departments) => {
        departments.forEach(dept => {
          if (dept.employees) {
            dept.employees.forEach(emp => {
              users.push({
                id: emp.id.toString(),
                name: emp.nickname || emp.username,
                avatar: (emp.avatar && emp.avatar.startsWith('http')) ? emp.avatar : (emp.avatar ? serverUrl.value + emp.avatar : 'https://api.dicebear.com/7.x/avataaars/svg?seed=emp'),
                department: dept.name
              })
            })
          }
          if (dept.subDepartments) {
            extractUsers(dept.subDepartments)
          }
        })
      }
      
      extractUsers(orgResponse.data)
      shareUsers.value = users
    }
    
    // 加载群聊列表
    const convResponse = await request('/api/v1/conversations')
    if (convResponse.code === 0) {
      const groups = convResponse.data.filter(conv => conv.type === 'group')
      shareGroups.value = groups.map(group => ({
        id: group.id.toString(),
        name: group.name,
        avatar: group.avatar || 'https://api.dicebear.com/7.x/avataaars/svg?seed=group',
        members: group.members || []
      }))
    }
  } catch (error) {
    console.error('加载分享数据失败:', error)
  }
}

// 处理分享确认
const handleShareConfirm = async (selection) => {
  try {
    const { users, groups } = selection
    const shareData = window.shareData
    
    // 构建分享消息内容
    let shareContent = ''
    let shareName = ''
    switch (shareType.value) {
      case 'file':
        shareContent = `分享了文件: ${shareData.name}`
        shareName = shareData.name
        break
      case 'note':
        shareContent = `分享了笔记: ${shareData.title}`
        shareName = shareData.title
        break
      case 'sticky':
        shareContent = `分享了便签: ${shareData.title}`
        shareName = shareData.title
        break
      case 'message':
        if (shareData.type === 'text') {
          shareContent = `转发了消息: ${shareData.content.substring(0, 20)}${shareData.content.length > 20 ? '...' : ''}`
          shareName = '文本消息'
        } else if (shareData.type === 'image') {
          shareContent = '转发了图片'
          shareName = '图片消息'
        } else {
          shareContent = '转发了消息'
          shareName = '消息'
        }
        break
      default:
        shareContent = '分享了内容'
        shareName = '内容'
    }
    
    // 准备分享数据
    const shareDataObj = {
      type: shareType.value,
      id: shareData.id || shareData.messageId,
      name: shareName,
      content: shareContent,
      originalMessage: shareType.value === 'message' ? shareData : undefined // 存储原始消息数据
    }
    
    // 发送分享消息给选择的用户
    for (const userId of users) {
      // 创建私聊会话
      const convResponse = await request('/api/v1/conversations/single', {
        method: 'POST',
        body: JSON.stringify({ user_id: parseInt(userId) })
      })
      
      if (convResponse.code === 0) {
        // 发送分享消息
        let messageData = {
          type: 'share',
          content: JSON.stringify(shareDataObj)
        }
        
        // 如果是转发消息，根据原始消息类型发送相应的消息
        if (shareType.value === 'message' && shareDataObj.originalMessage) {
          const originalMessage = shareDataObj.originalMessage
          if (originalMessage.type === 'text') {
            messageData = {
              type: 'text',
              content: `[转发] ${originalMessage.content}`
            }
          } else if (originalMessage.type === 'image' || originalMessage.type === 'file' || originalMessage.type === 'miniApp') {
            // 对于图片、文件和小程序，直接复制消息类型和内容
            messageData = {
              type: originalMessage.type,
              content: originalMessage.content
            }
          }
        }
        
        const messageResponse = await request(`/api/v1/conversations/${convResponse.data.id}/messages`, {
          method: 'POST',
          body: JSON.stringify(messageData)
        })
        
        // 如果当前正在查看这个会话，手动添加消息到前端列表
        if (currentConversationId.value === convResponse.data.id.toString()) {
          const newMessage = {
            id: messageResponse.data.id.toString(),
            content: messageData.content,
            sender: currentUser.value,
            timestamp: Date.now(),
            type: messageData.type,
            isSelf: true,
            isRead: false
          }
          // 检查消息是否已经存在，避免重复添加
          const messageExists = messages.value.some(msg => msg.id === newMessage.id)
          if (!messageExists) {
            messages.value.push(newMessage)
          }
        }
      }
    }
    
    // 发送分享消息给选择的群聊
    for (const groupId of groups) {
      // 发送分享消息
      let messageData = {
        type: 'share',
        content: JSON.stringify(shareDataObj)
      }
      
      // 如果是转发消息，根据原始消息类型发送相应的消息
      if (shareType.value === 'message' && shareDataObj.originalMessage) {
        const originalMessage = shareDataObj.originalMessage
        if (originalMessage.type === 'text') {
          messageData = {
            type: 'text',
            content: `[转发] ${originalMessage.content}`
          }
        } else if (originalMessage.type === 'image' || originalMessage.type === 'file' || originalMessage.type === 'miniApp') {
          // 对于图片、文件和小程序，直接复制消息类型和内容
          messageData = {
            type: originalMessage.type,
            content: originalMessage.content
          }
        }
      }
      
      const messageResponse = await request(`/api/v1/conversations/${parseInt(groupId)}/messages`, {
        method: 'POST',
        body: JSON.stringify(messageData)
      })
      
      // 如果当前正在查看这个会话，手动添加消息到前端列表
      if (currentConversationId.value === groupId) {
        const newMessage = {
          id: messageResponse.data.id.toString(),
          content: messageData.content,
          sender: currentUser.value,
          timestamp: Date.now(),
          type: messageData.type,
          isSelf: true,
          isRead: false
        }
        // 检查消息是否已经存在，避免重复添加
        const messageExists = messages.value.some(msg => msg.id === newMessage.id)
        if (!messageExists) {
          messages.value.push(newMessage)
        }
      }
    }
    
    showMessage({ message: '分享成功', type: 'success' })
    // 不需要手动刷新会话列表，WebSocket会自动处理新消息和会话更新
    
    // 打开第一个分享对象的聊天界面
    if (users.length > 0) {
      // 打开第一个用户的聊天界面
      const firstUserId = users[0]
      // 重新加载会话列表，确保新创建的会话存在
      await loadConversations()
      // 查找对应的会话ID
      const conversation = conversations.value.find(conv => 
        conv.type === 'single' && 
        conv.members && 
        conv.members.some(member => member.id === firstUserId)
      )
      if (conversation) {
        // 调用handleSwitchConversation，确保与正常切换会话的逻辑一致
        await handleSwitchConversation(conversation.id)
      }
    } else if (groups.length > 0) {
      // 打开第一个群聊的聊天界面
      const firstGroupId = groups[0]?.id
      // 调用handleSwitchConversation，确保与正常切换会话的逻辑一致
      if (firstGroupId) {
        await handleSwitchConversation(firstGroupId)
      }
    }
  } catch (error) {
    console.error('分享失败:', error)
    showMessage({ message: '分享失败', type: 'error' })
  } finally {
    closeShareModal()
  }
}

const exitGroup = async () => {
  if (selectedGroup.value) {
    if (confirm(`确定要退出${selectedGroup.value.name}吗？`)) {
      try {
        const response = await request(`/api/v1/conversations/${selectedGroup.value.id}/exit`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          }
        })
        
        if (response.code === 200) {
          // 标记群聊为已退出
          const conversationIndex = conversations.value.findIndex(c => c.id === selectedGroup.value.id)
          if (conversationIndex !== -1) {
            // 更新会话状态为已退出
            const updatedConversation = {
              ...conversations.value[conversationIndex],
              isExited: true
            }
            conversations.value.splice(conversationIndex, 1, updatedConversation)
            // 强制触发响应式更新
            conversations.value = [...conversations.value]
            // 保存会话到本地存储
            storage.saveConversations(conversations.value)
          }
          // 关闭群聊上下文菜单
          closeGroupContextMenu()
          showMessage({ message: '退出群聊成功', type: 'success' })
        } else {
          showMessage({ message: '退出群聊失败: ' + response.message, type: 'error' })
        }
      } catch (error) {
        console.error('退出群聊失败:', error)
        showMessage({ message: '退出群聊失败，请稍后重试', type: 'error' })
      }
    }
  }
  closeGroupContextMenu()
  selectedGroup.value = null
}

// 移除成员
const removeMember = async (member) => {
  if (selectedGroup.value) {
    // 使用更美观的确认对话框
    if (window.confirm(`确定要将${member.name}移出群聊吗？`)) {
      try {
        const response = await request(`/api/v1/conversations/${selectedGroup.value.id}/members/${member.id}`, {
          method: 'DELETE'
        })
        
        if (response.code === 0) {
          // 更新群成员列表
          groupMembers.value = groupMembers.value.filter(m => m.id !== member.id)
          // 更新选中群的成员列表
          if (selectedGroup.value.members) {
            selectedGroup.value.members = selectedGroup.value.members.filter(m => m.id !== member.id)
          }
          // 更新会话列表中对应群聊的成员数
          const conversationIndex = conversations.value.findIndex(c => c.id === selectedGroup.value.id)
          if (conversationIndex !== -1) {
            conversations.value[conversationIndex].members = selectedGroup.value.members
            // 强制触发响应式更新
            conversations.value = [...conversations.value]
          }
          showMessage({ message: '移除成员成功', type: 'success' })
        } else {
          showMessage({ message: '移除成员失败: ' + response.message, type: 'error' })
        }
      } catch (error) {
        console.error('移除成员失败:', error)
        showMessage({ message: '移除成员失败，请稍后重试', type: 'error' })
      }
    }
  }
}

// 关闭添加成员模态框
const closeAddMembersModal = () => {
  showAddMembersModal.value = false
  selectedAddMembers.value = []
  addMembersSearchQuery.value = ''
  selectedGroup.value = null
}

// 关闭群成员模态框
const closeGroupMembersModal = () => {
  showGroupMembersModal.value = false
  groupMembers.value = []
}

// 关闭群资料模态框
const closeGroupInfoModal = () => {
  showGroupInfoModal.value = false
}

// 切换成员选择状态
const toggleAddMember = (employee: any) => {
  const index = selectedAddMembers.value.findIndex(m => m.id === employee.id)
  if (index > -1) {
    selectedAddMembers.value.splice(index, 1)
  } else {
    selectedAddMembers.value.push(employee)
  }
}

// 确认添加成员
const confirmAddMembers = async () => {
  if (!selectedGroup.value || selectedAddMembers.value.length === 0) {
    return
  }
  
  try {
    const memberIDs = selectedAddMembers.value.map(m => parseInt(m.id))
    const response = await request(`/api/v1/conversations/${selectedGroup.value.id}/members`, {
      method: 'POST',
      body: JSON.stringify({ member_ids: memberIDs })
    })
    
    if (response.code === 0) {
      // 确保response.data是一个数组
      const newMembers = (Array.isArray(response.data) ? response.data : []).map(member => ({
        id: member.id?.toString() || '',
        name: member.nickname || member.username || (member.name !== undefined ? member.name : '未知用户'),
        avatar: member.avatar || ''
      }))
      
      // 更新群聊成员列表
      if (selectedGroup.value.members) {
        selectedGroup.value.members = [...selectedGroup.value.members, ...newMembers]
      } else {
        selectedGroup.value.members = newMembers
      }
      
      // 更新groupMembers，确保群聊人数显示正确
      groupMembers.value = selectedGroup.value.members || []
      
      // 更新会话列表中对应群聊的成员数
      const conversationIndex = conversations.value.findIndex(c => c.id === selectedGroup.value.id)
      if (conversationIndex !== -1) {
        conversations.value[conversationIndex].members = selectedGroup.value.members
        // 强制触发响应式更新
        conversations.value = [...conversations.value]
      }
      
      ElMessage.success('添加成员成功')
      closeAddMembersModal()
    } else {
      ElMessage.error('添加成员失败: ' + response.message)
    }
  } catch (error) {
    console.error('添加成员失败:', error)
    ElMessage.error('添加成员失败，请稍后重试')
  }
}

const handlePin = async (conversation: Conversation | null) => {
  if (conversation) {
    try {
      const newValue = !conversation.pinned
      console.log('切换置顶状态:', { conversationId: conversation.id, is_pinned: newValue })
      const token = getToken()
      const response = await axios.put(`${serverUrl.value}/api/v1/conversations/${conversation.id}/pin`, {
        "is_pinned": newValue
      }, {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        }
      })
      if (response.data.code === 0) {
        // 直接更新pinned字段，确保状态正确切换
        conversation.pinned = newValue
        // 重新排序会话列表，确保UI正确更新
        conversations.value = [...conversations.value]
        showMessage({ message: newValue ? '会话已置顶' : '会话已取消置顶', type: 'success' })
      }
    } catch (error) {
      console.error('切换置顶状态失败:', error)
      showMessage({ message: '切换置顶状态失败', type: 'error' })
    }
  }
  closeContextMenu()
}

const handleMute = (conversation: Conversation | null) => {
  if (conversation) {
    conversation.muted = !conversation.muted
  }
  closeContextMenu()
}

const handleRemove = (conversation: Conversation | null) => {
  if (conversation) {
    const index = conversations.value.findIndex(c => c.id === conversation.id)
    if (index > -1) {
      conversations.value.splice(index, 1)
      if (currentConversationId.value === conversation.id) {
        currentConversationId.value = null
      }
    }
  }
  closeContextMenu()
}

const handleExitGroup = async (conversation: Conversation | null) => {
  if (conversation && conversation.type === 'group') {
    if (confirm(`确定要退出${conversation.name}吗？`)) {
      try {
        const response = await request(`/api/v1/conversations/${conversation.id}/exit`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          }
        })
        
        if (response.code === 200) {
          // 不从会话列表中移除该群聊，只更新会话状态
          const conversationIndex = conversations.value.findIndex(c => c.id === conversation.id)
          if (conversationIndex !== -1) {
            // 更新会话状态为已退出
            const updatedConversation = {
              ...conversations.value[conversationIndex],
              isExited: true
            }
            conversations.value.splice(conversationIndex, 1, updatedConversation)
            // 强制触发响应式更新
            conversations.value = [...conversations.value]
            // 保存会话到本地存储
            storage.saveConversations(conversations.value)
          }
          closeContextMenu()
          showMessage({ message: '退出群聊成功', type: 'success' })
        } else {
          showMessage({ message: '退出群聊失败: ' + response.message, type: 'error' })
        }
      } catch (error) {
        console.error('退出群聊失败:', error)
        showMessage({ message: '退出群聊失败，请稍后重试', type: 'error' })
      }
    }
  }
  closeContextMenu()
}

// 点击其他地方关闭菜单由showContextMenu和showGroupContextMenu函数内部处理

// 窗口控制方法
const minimizeWindow = () => {
  console.log('Minimize window clicked')
  console.log('window.electron:', window.electron)
  console.log('window.electron?.ipcRenderer:', window.electron?.ipcRenderer)
  if (window.electron?.ipcRenderer) {
    console.log('Sending minimize-window event via ipcRenderer')
    window.electron.ipcRenderer.send('minimize-window')
  } else {
    console.log('Electron not available')
  }
}

const maximizeWindow = () => {
  console.log('Maximize window clicked')
  console.log('window.electron:', window.electron)
  console.log('window.electron?.ipcRenderer:', window.electron?.ipcRenderer)
  if (window.electron?.ipcRenderer) {
    console.log('Sending maximize-window event via ipcRenderer')
    window.electron.ipcRenderer.send('maximize-window')
  } else {
    console.log('Electron not available')
  }
}

const closeWindow = () => {
  console.log('Close window clicked')
  console.log('window.electron:', window.electron)
  console.log('window.electron?.ipcRenderer:', window.electron?.ipcRenderer)
  if (window.electron?.ipcRenderer) {
    console.log('Sending close-window event via ipcRenderer')
    window.electron.ipcRenderer.send('close-window')
  } else {
    console.log('Electron not available')
  }
}



const showSettingsMenu = (event: MouseEvent) => {
  event.stopPropagation()
  
  // 关闭主题菜单和更多菜单
  closeThemeMenu()
  closeMoreMenu()
  
  // 获取设置按钮的DOM元素
  const settingsButton = event.currentTarget as HTMLElement
  if (settingsButton) {
    // 计算按钮的位置
    const rect = settingsButton.getBoundingClientRect()
    
    // 菜单宽度和高度
    const menuWidth = 180
    const menuHeight = 160
    const windowWidth = window.innerWidth
    const windowHeight = window.innerHeight
    
    // 计算菜单位置：按钮右侧2px，底部与鼠标点击位置对齐
    let x = rect.right + 2
    let y = event.clientY - menuHeight
    
    // 调整x坐标，确保菜单不超出屏幕右侧
    if (x + menuWidth > windowWidth) {
      x = rect.left - menuWidth - 10
    }
    
    // 调整y坐标，确保菜单不超出屏幕底部
    if (y + menuHeight > windowHeight) {
      y = windowHeight - menuHeight - 10
    }
    
    // 确保y坐标不小于0
    if (y < 0) {
      y = 10
    }
    
    settingsMenuPosition.value = {
      x,
      y
    }
    showSettingsMenuFlag.value = true
    
    // 点击其他地方关闭菜单
    setTimeout(() => {
      document.addEventListener('click', closeSettingsMenu)
    }, 0)
  }
}

const closeSettingsMenu = () => {
  showSettingsMenuFlag.value = false
  document.removeEventListener('click', closeSettingsMenu)
}

const openSettings = () => {
  console.log('打开设置')
  // 打开系统设置模态框
  showSettingsModal.value = true
  // 加载设置值
  settingsProfile.value = {
    nickname: currentUser.value?.nickname || currentUser.value?.username || '我的账号',
    signature: '这个人很懒，什么都没留下'
  }
  
  // 加载消息设置
  const savedMessageSettings = localStorage.getItem('messageSettings')
  if (savedMessageSettings) {
    try {
      messageSettings.value = JSON.parse(savedMessageSettings)
    } catch (error) {
      console.error('解析消息设置失败:', error)
      messageSettings.value = {
        notificationsEnabled: true,
        soundEnabled: true,
        desktopNotificationsEnabled: true,
        dndMode: 'none'
      }
    }
  } else {
    messageSettings.value = {
      notificationsEnabled: true,
      soundEnabled: true,
      desktopNotificationsEnabled: true,
      dndMode: 'none'
    }
  }
  
  // 加载外观设置
  const savedAppearanceSettings = localStorage.getItem('appearanceSettings')
  if (savedAppearanceSettings) {
    try {
      appearanceSettings.value = JSON.parse(savedAppearanceSettings)
    } catch (error) {
      console.error('解析外观设置失败:', error)
      appearanceSettings.value = {
        theme: currentTheme.value,
        fontSize: 14
      }
    }
  } else {
    appearanceSettings.value = {
      theme: currentTheme.value,
      fontSize: 14
    }
  }
  
  // 加载文件设置
  const savedFileSettings = localStorage.getItem('fileSettings')
  if (savedFileSettings) {
    try {
      fileSettings.value = JSON.parse(savedFileSettings)
      // 确保新字段存在
      if (fileSettings.value.autoDownload === undefined) {
        fileSettings.value.autoDownload = false
      }
      if (fileSettings.value.maxFileSize === undefined) {
        fileSettings.value.maxFileSize = 50
      }
      if (fileSettings.value.allowedFileTypes === undefined) {
        fileSettings.value.allowedFileTypes = 'jpg,png,gif,pdf,doc,docx,xls,xlsx,ppt,pptx,zip,rar'
      }
      if (fileSettings.value.autoPreviewImages === undefined) {
        fileSettings.value.autoPreviewImages = true
      }
      if (fileSettings.value.enableFileHistory === undefined) {
        fileSettings.value.enableFileHistory = true
      }
    } catch (error) {
      console.error('解析文件设置失败:', error)
      fileSettings.value = {
        defaultSaveDirectory: '',
        autoDownload: false,
        maxFileSize: 50,
        allowedFileTypes: 'jpg,png,gif,pdf,doc,docx,xls,xlsx,ppt,pptx,zip,rar',
        autoPreviewImages: true,
        enableFileHistory: true
      }
    }
  } else {
    fileSettings.value = {
      defaultSaveDirectory: '',
      autoDownload: false,
      maxFileSize: 50,
      allowedFileTypes: 'jpg,png,gif,pdf,doc,docx,xls,xlsx,ppt,pptx,zip,rar',
      autoPreviewImages: true,
      enableFileHistory: true
    }
  }
  closeSettingsMenu()
}

// 监听自动更新事件
if (window.electron) {
  // 检查更新中
  window.electron.ipcRenderer.on('update-checking', () => {
    isCheckingUpdate.value = true
    updateResult.value = '正在检查更新...'
  })
  
  // 发现新版本
  window.electron.ipcRenderer.on('update-available', (event, info) => {
    isCheckingUpdate.value = false
    hasNewVersion.value = true
    updateResult.value = `发现新版本 v${info.version}`
  })
  
  // 当前已是最新版本
  window.electron.ipcRenderer.on('update-not-available', () => {
    isCheckingUpdate.value = false
    hasNewVersion.value = false
    updateResult.value = '当前已是最新版本'
  })
  
  // 更新错误
  window.electron.ipcRenderer.on('update-error', (event, error) => {
    isCheckingUpdate.value = false
    updateResult.value = `更新错误: ${error}`
  })
  
  // 下载进度
  window.electron.ipcRenderer.on('update-progress', (event, progressObj) => {
    isDownloading.value = true
    downloadProgress.value = progressObj.percent
  })
  
  // 更新下载完成
  window.electron.ipcRenderer.on('update-downloaded', (event, info) => {
    isDownloading.value = false
    updateResult.value = '下载完成，正在安装...'
    setTimeout(() => {
      updateResult.value = '升级成功，需要重启应用'
      hasNewVersion.value = false
    }, 1500)
  })
}

const checkForUpdates = () => {
  console.log('检查更新')
  // 显示检查更新对话框
  showUpdateDialog.value = true
  isCheckingUpdate.value = true
  updateResult.value = '正在检查更新...'
  hasNewVersion.value = false
  downloadProgress.value = 0
  isDownloading.value = false
  closeSettingsMenu()
  
  // 向主进程发送检查更新请求
  if (window.electron) {
    window.electron.ipcRenderer.send('check-for-updates')
  } else {
    // 模拟检查更新的过程（开发环境）
    setTimeout(() => {
      isCheckingUpdate.value = false
      // 模拟有新版本
      if (Math.random() > 0.5) {
        hasNewVersion.value = true
        updateResult.value = '发现新版本 v1.0.1'
      } else {
        hasNewVersion.value = false
        updateResult.value = '当前已是最新版本'
      }
    }, 1500)
  }
}

const downloadUpdate = () => {
  console.log('下载升级')
  isDownloading.value = true
  downloadProgress.value = 0
  
  // 向主进程发送下载更新请求
  if (window.electron) {
    window.electron.ipcRenderer.send('download-update')
  } else {
    // 模拟下载过程（开发环境）
    const interval = setInterval(() => {
      downloadProgress.value += 5
      if (downloadProgress.value >= 100) {
        clearInterval(interval)
        isDownloading.value = false
        updateResult.value = '下载完成，正在安装...'
        
        // 模拟安装过程
        setTimeout(() => {
          updateResult.value = '升级成功，需要重启应用'
          hasNewVersion.value = false
        }, 1500)
      }
    }, 100)
  }
}

const aboutApp = () => {
  console.log('关于应用')
  showAboutDialog.value = true
  closeSettingsMenu()
}

const closeAboutDialog = () => {
  showAboutDialog.value = false
}

const emit = defineEmits<{
  (e: 'logout'): void
}>()

const showLogoutDialog = ref(false)
const showUpdateDialog = ref(false)
const isCheckingUpdate = ref(false)
const updateResult = ref('')
const hasNewVersion = ref(false)
const downloadProgress = ref(0)
const isDownloading = ref(false)

const logout = () => {
  console.log('退出登录')
  // 显示退出登录确认弹窗
  showLogoutDialog.value = true
  closeSettingsMenu()
}

const confirmLogout = () => {
  // 清除本地存储的用户信息
  localStorage.removeItem('user')
  // 触发退出登录事件
  emit('logout')
  showLogoutDialog.value = false
}

const cancelLogout = () => {
  showLogoutDialog.value = false
}

const closeUpdateDialog = () => {
  showUpdateDialog.value = false
}

// 主题菜单相关函数
const showThemeMenu = (event: MouseEvent) => {
  event.stopPropagation()
  
  // 关闭设置菜单
  closeSettingsMenu()
  
  // 获取皮肤按钮的DOM元素
  const themeButton = event.currentTarget as HTMLElement
  if (themeButton) {
    // 计算按钮的位置
    const rect = themeButton.getBoundingClientRect()
    
    // 菜单宽度和高度
    const menuWidth = 180
    const menuHeight = 160
    const windowWidth = window.innerWidth
    const windowHeight = window.innerHeight
    
    // 计算菜单位置：按钮右侧2px，底部与鼠标点击位置对齐
    let x = rect.right + 2
    let y = event.clientY - menuHeight
    
    // 调整x坐标，确保菜单不超出屏幕右侧
    if (x + menuWidth > windowWidth) {
      x = rect.left - menuWidth - 10
    }
    
    // 调整y坐标，确保菜单不超出屏幕底部
    if (y + menuHeight > windowHeight) {
      y = windowHeight - menuHeight - 10
    }
    
    // 确保y坐标不小于0
    if (y < 0) {
      y = 10
    }
    
    themeMenuPosition.value = {
      x,
      y
    }
    showThemeMenuFlag.value = true
    
    // 点击其他地方关闭菜单
    setTimeout(() => {
      document.addEventListener('click', closeThemeMenu)
    }, 0)
  }
}

const closeThemeMenu = () => {
  showThemeMenuFlag.value = false
  document.removeEventListener('click', closeThemeMenu)
}

const showMoreMenu = (event: MouseEvent) => {
  event.stopPropagation()
  
  // 关闭其他菜单
  closeSettingsMenu()
  closeThemeMenu()
  
  // 获取更多按钮的DOM元素
  const moreButton = event.currentTarget as HTMLElement
  if (moreButton) {
    // 计算按钮的位置
    const rect = moreButton.getBoundingClientRect()
    
    // 菜单宽度和高度
    const menuWidth = 120
    const menuHeight = 180
    const windowWidth = window.innerWidth
    const windowHeight = window.innerHeight
    
    // 计算菜单位置：默认显示在按钮右侧，与按钮顶部对齐
    let x = rect.right + 10
    let y = rect.top
    
    // 调整x坐标，确保菜单不超出屏幕右侧
    if (x + menuWidth > windowWidth) {
      x = rect.left - menuWidth - 10
    }
    
    // 调整y坐标，确保菜单不超出屏幕底部
    if (y + menuHeight > windowHeight) {
      y = windowHeight - menuHeight - 10
    }
    
    // 确保y坐标不小于0
    if (y < 0) {
      y = 10
    }
    
    moreMenuPosition.value = {
      x,
      y
    }
    showMoreMenuFlag.value = true
    
    // 点击其他地方关闭菜单
    setTimeout(() => {
      document.addEventListener('click', closeMoreMenu)
    }, 0)
  }
}

const closeMoreMenu = () => {
  showMoreMenuFlag.value = false
  document.removeEventListener('click', closeMoreMenu)
}

// 关闭系统设置模态框
const closeSettingsModal = () => {
  showSettingsModal.value = false
}

// 保存系统设置
const saveSettings = async () => {
  try {
    // 保存个人信息
    const profileResponse = await request('/api/v1/users/me', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        nickname: settingsProfile.value.nickname,
        signature: settingsProfile.value.signature
      })
    })
    
    // 保存主题设置
    if (appearanceSettings.value.theme !== currentTheme.value) {
      setTheme(appearanceSettings.value.theme)
    }
    
    // 应用字体大小设置
    localStorage.setItem('fontSize', appearanceSettings.value.fontSize.toString())
    applyFontSize(appearanceSettings.value.fontSize)
    
    // 保存其他设置到本地存储
    localStorage.setItem('messageSettings', JSON.stringify(messageSettings.value))
    localStorage.setItem('appearanceSettings', JSON.stringify(appearanceSettings.value))
    localStorage.setItem('fileSettings', JSON.stringify(fileSettings.value))
    
    if (profileResponse.code === 0) {
      // 更新当前用户信息
      if (currentUser.value) {
        currentUser.value.username = settingsProfile.value.nickname
      }
      ElMessage.success('保存成功')
      closeSettingsModal()
    } else {
      ElMessage.error('保存失败: ' + profileResponse.message)
    }
  } catch (error) {
    console.error('保存设置失败:', error)
    ElMessage.error('保存失败: ' + error.message)
  }
}

// 清除缓存
const clearCache = () => {
  if (confirm('确定要清除缓存吗？')) {
    // 清除本地存储
    localStorage.removeItem('messageSettings')
    localStorage.removeItem('appearanceSettings')
    localStorage.removeItem('fileSettings')
    ElMessage.success('缓存已清除')
  }
}

// 保存双因素认证设置
const saveTwoFactorSetting = async () => {
  try {
    const token = getToken()
    if (!token) {
      showMessage({ message: '请先登录', type: 'error' })
      return
    }
    const response = await axios.put(`${serverUrl.value}/api/v1/users/me`, {
      two_factor_enabled: advancedSettings.value.twoFactorEnabled
    }, {
      headers: {
        'Authorization': `Bearer ${token}`
      }
    })
    if (response.data.code === 0) {
      showMessage({ message: '设置保存成功', type: 'success' })
    }
  } catch (error) {
    console.error('保存双因素认证设置失败:', error)
    showMessage({ message: '保存设置失败', type: 'error' })
  }
}

// 浏览默认保存目录
const browseDefaultSaveDirectory = () => {
  // 在Electron环境中，使用dialog.showOpenDialog
  if (window.electron && window.electron.ipcRenderer) {
    window.electron.ipcRenderer.send('open-file-dialog', { properties: ['openDirectory'] })
    
    // 监听选择结果
    window.electron.ipcRenderer.once('file-dialog-result', (event, result) => {
      if (!result.canceled && result.filePaths && result.filePaths.length > 0) {
        fileSettings.value.defaultSaveDirectory = result.filePaths[0]
        showMessage({ message: '目录已选择', type: 'success' })
      }
    })
  } else {
    // 非Electron环境，使用模拟路径
    fileSettings.value.defaultSaveDirectory = '/Users/yourname/Downloads'
    showMessage({ message: '非Electron环境，已设置默认路径', type: 'info' })
  }
}

// 打开安全设置
const openSecuritySettings = () => {
  ElMessage.info('打开安全设置')
  // 这里可以实现打开安全设置页面的逻辑
}

const setTheme = (theme: string) => {
  currentTheme.value = theme
  localStorage.setItem('theme', theme)
  appearanceSettings.value.theme = theme
  // 应用主题到页面
  document.documentElement.setAttribute('data-theme', theme)
  closeThemeMenu()
}

// 应用字体大小设置
const applyFontSize = (fontSize: number) => {
  const container = document.querySelector('.im-container') as HTMLElement
  if (container) {
    container.style.fontSize = fontSize + 'px'
  }
  localStorage.setItem('fontSize', fontSize.toString())
}

// 初始化主题和字体大小
const initTheme = () => {
  const savedTheme = localStorage.getItem('theme') || 'light'
  const mappedTheme = themeNameMap[savedTheme] || savedTheme
  currentTheme.value = mappedTheme
  appearanceSettings.value.theme = mappedTheme
  document.documentElement.setAttribute('data-theme', mappedTheme)
  
  // 初始化字体大小
  const savedFontSize = localStorage.getItem('fontSize')
  if (savedFontSize) {
    appearanceSettings.value.fontSize = parseInt(savedFontSize)
  }
  applyFontSize(appearanceSettings.value.fontSize)
}

// 初始化主题
initTheme()

// 通知中心相关
const showNotificationCenterFlag = ref(false)
const notificationCenterPosition = ref({ x: 0, y: 0 })
const notificationCenterRef = ref<any>(null)

const showNotificationCenter = (event: MouseEvent) => {
  event.stopPropagation()

  // 切换通知中心显示状态
  if (showNotificationCenterFlag.value) {
    closeNotificationCenter()
    return
  }

  closeActionMenu()
  closeSettingsMenu()
  closeThemeMenu()

  const notificationButton = event.currentTarget as HTMLElement
  if (notificationButton) {
    const rect = notificationButton.getBoundingClientRect()

    const menuWidth = 380
    const menuHeight = 480
    const windowWidth = window.innerWidth
    const windowHeight = window.innerHeight

    let x = rect.right + 2
    let y = rect.top

    if (x + menuWidth > windowWidth) {
      x = rect.left - menuWidth - 10
    }

    if (y + menuHeight > windowHeight) {
      y = windowHeight - menuHeight - 10
    }
    
    if (y < 0) {
      y = 10
    }
    
    notificationCenterPosition.value = {
      x,
      y
    }
    showNotificationCenterFlag.value = true

    // 加载通知列表
    if (notificationCenterRef.value) {
      notificationCenterRef.value.loadNotifications()
    }

    setTimeout(() => {
      document.addEventListener('click', closeNotificationCenter)
    }, 0)
  }
}

const closeNotificationCenter = () => {
  showNotificationCenterFlag.value = false
  document.removeEventListener('click', closeNotificationCenter)
}

const handleNotificationCenterClick = (notification: any) => {
  if (notification.type === 'message' && notification.data?.conversationId) {
    activeOption.value = 'recent'
    currentConversationId.value = notification.data.conversationId
    loadMessages(notification.data.conversationId)
  } else if (notification.type === 'group' && notification.data?.groupId) {
    activeOption.value = 'groups'
  }
}

const handleNewNotification = (notification: any) => {
  console.log('收到新通知:', notification)

  // 显示通知提示
  showMessage({
    message: notification.content || notification.title || '您有一条新通知',
    type: 'info',
    duration: 5000
  })

  // 将通知添加到通知中心
  if (notificationCenterRef.value) {
    const newNotification = {
      id: notification.id || Date.now().toString(),
      title: notification.title || '新通知',
      content: notification.content || '',
      timestamp: notification.timestamp || Date.now(),
      read: false,
      type: notification.type || 'system',
      data: notification.data || {}
    }

    // 获取当前通知列表
    const currentNotifications = notificationCenterRef.value.notifications || []
    // 添加新通知到列表开头
    notificationCenterRef.value.notifications = [newNotification, ...currentNotifications]
    // 重新过滤通知
    notificationCenterRef.value.filterNotifications()

    // 更新未读通知计数
    unreadNotificationCount.value++
  }
}
</script>

<style>
/* ========================================
   导入设计系统 - Import Design System
   ======================================== */
@import url('../assets/styles/design-tokens.css');
@import url('../assets/styles/components.css');

/* 过渡动画 */
.im-container {
  animation: mainWindowFadeIn 0.5s ease-out forwards;
  opacity: 0;
  transform: scale(0.95);
}

@keyframes mainWindowFadeIn {
  0% {
    opacity: 0;
    transform: scale(0.95);
  }
  100% {
    opacity: 1;
    transform: scale(1);
  }
}

/* ========================================
   主题样式 - Theme Styles
   ======================================== */
[data-theme="elegant-dark"] {
  --primary-color: #4b5563;
  --primary-light: #374151;
  --secondary-color: #0a0a0a;
  --text-color: #e5e7eb;
  --border-color: #374151;
  --hover-color: #2d3748;
  --active-color: #6b7280;
  --sidebar-bg: #0f0f0f;
  --window-controls-bg: #0f0f0f;
  --context-menu-bg: #1f2937;
  --context-menu-hover: #374151;
  --accent-color: #4b5563;
  --text-secondary: #9ca3af;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.3);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.4), 0 2px 4px -1px rgba(0, 0, 0, 0.3);
  --panel-bg: #121212;
  --list-bg: #1a1a1a;
  --card-bg: #161616;
  --header-panel-bg: #0f0f0f;
  --content-bg: #0a0a0a;
}

/* 炫酷黑主题 - 侧边栏内容区域 */
[data-theme="elegant-dark"] .sidebar-content {
  background: var(--sidebar-bg) !important;
}

[data-theme="elegant-dark"] .conversation-list {
  background: var(--list-bg) !important;
}

[data-theme="elegant-dark"] .conversation-item {
  background: var(--list-bg) !important;
}

[data-theme="elegant-dark"] .groups-list {
  background: var(--list-bg) !important;
}

[data-theme="elegant-dark"] .group-item {
  background: var(--list-bg) !important;
}

[data-theme="elegant-dark"] .tree-container {
  background: var(--list-bg) !important;
}

[data-theme="elegant-dark"] .apps-container {
  background: var(--list-bg) !important;
}

[data-theme="elegant-dark"] .search-input {
  background: var(--sidebar-bg) !important;
}

/* 炫酷黑主题 - 搜索框背景 */
[data-theme="elegant-dark"] .search-box {
  background: var(--sidebar-bg) !important;
  border-top-color: var(--border-color) !important;
  box-shadow: var(--shadow-sm) !important;
}

/* 天青蓝主题 */
[data-theme="ocean-blue"] {
  --primary-color: #49bccf;
  --primary-light: #e0f2fe;
  --secondary-color: #f0f9ff;
  --text-color: #0c4a6e;
  --border-color: #bae6fd;
  --hover-color: #dbeafe;
  --active-color: #3aa8b9;
  --sidebar-bg: #ffffff;
  --window-controls-bg: #ffffff;
  --context-menu-bg: #ffffff;
  --context-menu-hover: #e0f2fe;
  --accent-color: #68d8e8;
  --text-secondary: #475569;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.1);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.15), 0 2px 4px -1px rgba(0, 0, 0, 0.1);
  --panel-bg: #ffffff;
  --list-bg: #f0f9ff;
  --card-bg: #ffffff;
  --header-panel-bg: #e0f2fe;
}

/* 天青蓝主题 - 左边侧边栏 */
[data-theme="ocean-blue"] .side-options {
  background: linear-gradient(135deg, #49bccf 0%, #68d8e8 100%);
}

/* 天青蓝主题 - 文本颜色 */
[data-theme="ocean-blue"] .window-title,
[data-theme="ocean-blue"] .option-item,
[data-theme="ocean-blue"] .option-item.active {
  color: white;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}

/* 天青蓝主题 - 窗口控制栏左侧 */
[data-theme="ocean-blue"] .window-controls-left {
  background: linear-gradient(135deg, #49bccf 0%, #68d8e8 100%);
}

/* 天青蓝主题 - 侧边栏头部 */
[data-theme="ocean-blue"] .sidebar-header {
  /* background: #ffffff; */
  /* box-shadow: var(--shadow-md); */
   color: var(--text-color);
  text-shadow: none;
}

[data-theme="ocean-blue"] .sidebar-header .user-name {
  color: var(--text-color);
  text-shadow: none;
}

/* 炫酷黑主题 - 侧边栏头部 */
[data-theme="elegant-dark"] .sidebar-header {
  background: var(--sidebar-bg) !important;
  box-shadow: var(--shadow-md) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

[data-theme="elegant-dark"] .sidebar-header .user-name {
  color: var(--text-color) !important;
}

/* 炫酷黑主题 - 窗口控制栏左侧 */
/* [data-theme="elegant-dark"] .window-controls-left {
  background: var(--sidebar-bg) !important;
  border-right: 1px solid var(--border-color) !important;
} */

/* 炫酷黑主题 - 组织架构树 */
[data-theme="elegant-dark"] .org-content {
  background: var(--sidebar-bg) !important;
}

[data-theme="elegant-dark"] .employee-node .tree-node-content {
  background: transparent !important;
  opacity: 1 !important;
}

[data-theme="elegant-dark"] .employee-node .tree-node-content:hover {
  background: var(--hover-color) !important;
  opacity: 1 !important;
}

/* 炫酷黑主题 - 图标颜色 */
[data-theme="elegant-dark"] .option-icon {
  color: rgba(229, 231, 235, 0.7) !important;
}

[data-theme="elegant-dark"] .option-item:hover .option-icon,
[data-theme="elegant-dark"] .option-item.active .option-icon {
  color: rgba(229, 231, 235, 1) !important;
}

[data-theme="elegant-dark"] .app-icon,
[data-theme="elegant-dark"] .recent-app-grid-icon,
[data-theme="elegant-dark"] .category-app-icon,
[data-theme="elegant-dark"] .management-icon,
[data-theme="elegant-dark"] .empty-icon,
[data-theme="elegant-dark"] .file-icon,
[data-theme="elegant-dark"] .context-menu-icon,
[data-theme="elegant-dark"] .action-menu-icon,
[data-theme="elegant-dark"] .user-context-menu-icon,
[data-theme="elegant-dark"] .muted-icon,
[data-theme="elegant-dark"] .toggle-icon,
[data-theme="elegant-dark"] .tab-icon,
[data-theme="elegant-dark"] .category-icon,
[data-theme="elegant-dark"] .recent-app-icon {
  color: rgba(229, 231, 235, 0.7) !important;
}

[data-theme="elegant-dark"] .app-item:hover .app-icon,
[data-theme="elegant-dark"] .recent-app-grid-item:hover .recent-app-grid-icon,
[data-theme="elegant-dark"] .category-app-item:hover .category-app-icon,


[data-theme="elegant-dark"] .app-item:hover .app-icon {
  transform: scale(1.05);
  background: grey!important;
  /* color: #fff; */
}


/* 炫酷黑主题 - 侧边栏按钮样式 */
[data-theme="elegant-dark"] .option-item {
  color: rgba(229, 231, 235, 0.7) !important;
}

[data-theme="elegant-dark"] .option-item:hover {
  background: var(--hover-color) !important;
  color: rgba(229, 231, 235, 1) !important;
  box-shadow: var(--shadow-sm) !important;
}

[data-theme="elegant-dark"] .option-item.active {
  background: var(--hover-color) !important;
  color: rgba(229, 231, 235, 1) !important;
  box-shadow: var(--shadow-md) !important;
}

/* 炫酷黑主题 - 应用管理tab样式 */
[data-theme="elegant-dark"] .app-tab-item {
  color: rgba(229, 231, 235, 0.7) !important;
  border-bottom-color: transparent !important;
}

[data-theme="elegant-dark"] .app-tab-item:hover {
  background: var(--hover-color) !important;
  color: rgba(229, 231, 235, 1) !important;
}

[data-theme="elegant-dark"] .app-tab-item.active {
  border-bottom-color: rgba(229, 231, 235, 1) !important;
  color: rgba(229, 231, 235, 1) !important;
}

/* 炫酷黑主题 - 群聊徽章样式 */
[data-theme="elegant-dark"] .group-badge {
  background: #2d3748 !important;
  color: rgba(229, 231, 235, 1) !important;
  border: 1px solid rgba(229, 231, 235, 0.3) !important;
}

/* 炫酷黑主题 - 右边面板 */
[data-theme="elegant-dark"] .right-content {
  background: var(--secondary-color) !important;
}

[data-theme="elegant-dark"] .right-content-header {
  background: var(--sidebar-bg) !important;
  box-shadow: var(--shadow-md) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

[data-theme="elegant-dark"] .chat-header {
  /* background: var(--sidebar-bg) !important; */
  box-shadow: none !important;
  border-bottom: 1px solid var(--border-color) !important;
}

[data-theme="elegant-dark"] .right-content-body {
  background: var(--secondary-color) !important;
  color: var(--text-color) !important;
}

/* 炫酷黑主题 - 应用内容 */
[data-theme="elegant-dark"] .apps-content {
  background: var(--secondary-color) !important;
}

[data-theme="elegant-dark"] .recent-apps-section,
[data-theme="elegant-dark"] .all-apps-section {
  background: var(--secondary-color) !important;
}





/* 炫酷黑主题 - 应用分类 */
[data-theme="elegant-dark"] .app-category-item {
  background: transparent !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="elegant-dark"] .category-header {
  background: transparent !important;
}

/* Markdown 样式 */
.markdown-heading {
  margin: 20px 0 12px 0;
  font-weight: 600;
  line-height: 1.4;
  color: var(--text-color);
  transition: all 0.3s ease;
}

.markdown-heading:nth-child(1) {
  margin-top: 0;
}

.markdown-heading:nth-of-type(1) {
  font-size: 1.8em;
  border-bottom: 2px solid var(--border-color);
  padding-bottom: 8px;
}

.markdown-heading:nth-of-type(2) {
  font-size: 1.5em;
  border-bottom: 1px solid var(--border-color);
  padding-bottom: 6px;
}

.markdown-heading:nth-of-type(3) {
  font-size: 1.3em;
}

.markdown-heading:nth-of-type(4) {
  font-size: 1.1em;
}

.markdown-heading:nth-of-type(5),
.markdown-heading:nth-of-type(6) {
  font-size: 1em;
  color: var(--text-secondary);
}

.markdown-bold {
  font-weight: 600;
  color: var(--text-color);
}

.markdown-italic {
  font-style: italic;
  color: var(--text-secondary);
}

.markdown-list {
  margin: 12px 0;
  padding-left: 28px;
}

.markdown-list-item {
  margin: 6px 0;
  line-height: 1.5;
  color: var(--text-color);
}

.markdown-code {
  background: #f8f9fa;
  border: 1px solid #e9ecef;
  border-radius: 6px;
  padding: 16px;
  margin: 12px 0;
  overflow-x: auto;
  font-family: 'Fira Code', 'Courier New', Courier, monospace;
  font-size: 13px;
  line-height: 1.5;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.markdown-code:hover {
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}

.markdown-link {
  color: var(--primary-color);
  text-decoration: none;
  transition: all 0.3s ease;
  border-bottom: 1px solid transparent;
  padding-bottom: 2px;
}

.markdown-link:hover {
  color: var(--active-color);
  text-decoration: none;
  border-bottom-color: var(--active-color);
}

.markdown-image {
  max-width: 100%;
  max-height: 400px;
  border-radius: 8px;
  margin: 12px 0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transition: all 0.3s ease;
}

.markdown-image:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transform: scale(1.01);
}

/* 引用样式 */
.markdown-quote {
  border-left: 4px solid var(--primary-color);
  padding: 12px 16px;
  margin: 12px 0;
  background: var(--hover-color);
  border-radius: 0 6px 6px 0;
  color: var(--text-secondary);
  font-style: italic;
}

/* 表格样式 */
.markdown-table {
  width: 100%;
  border-collapse: collapse;
  margin: 12px 0;
  font-size: 14px;
}

.markdown-table th,
.markdown-table td {
  padding: 10px 12px;
  text-align: left;
  border: 1px solid var(--border-color);
}

.markdown-table th {
  background: var(--hover-color);
  font-weight: 600;
  color: var(--text-color);
}

.markdown-table tr:hover {
  background: var(--hover-color);
}



/* 文件管理样式优化 */
.file-item {
  transition: background-color 0.2s ease;
}

.file-item:hover {
  background-color: #f5f5f5;
}

.file-action-btn.delete-btn {
  opacity: 0;
  transition: opacity 0.2s ease;
}

.file-item:hover .file-action-btn {
  opacity: 1;
}

/* 应用管理样式优化 */
.app-item {
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.app-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

/* 组织架构样式优化 */
.tree-node-content {
  transition: background-color 0.2s ease;
}

.tree-node-content:hover {
  background-color: #f5f5f5;
}

/* 会话列表样式优化 */
.conversation-item {
  transition: background-color 0.2s ease;
}

.conversation-item:hover {
  background-color: #f5f5f5;
}

/* 按钮样式优化 */
button {
  transition: all 0.2s ease;
}

button:hover {
  transform: translateY(-1px);
}

button:active {
  transform: translateY(0);
}

/* 模态框样式优化 */
.modal-content {
  animation: modalFadeIn 0.3s ease;
}

@keyframes modalFadeIn {
  from {
    opacity: 0;
    transform: scale(0.95);
  }
  to {
    opacity: 1;
    transform: scale(1);
  }
}

/* 消息提示样式优化 */
@keyframes messageFadeIn {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes messageFadeOut {
  from {
    opacity: 1;
    transform: translateY(0);
  }
  to {
    opacity: 0;
    transform: translateY(-10px);
  }
}

/* 滚动条样式优化 */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

/* 响应式设计优化 */
@media (max-width: 768px) {
  .main-content {
    flex-direction: column;
  }
  
  .sidebar {
    width: 100%;
    height: 200px;
  }
  
  .right-content {
    flex: 1;
  }
}

/* 主题切换样式 - 深色主题下的Markdown样式 */
[data-theme="elegant-dark"] .markdown-code {
  background: #2d2d30;
  border-color: #3e3e42;
  color: #e0e0e0;
}

[data-theme="elegant-dark"] .markdown-link {
  color: #90caf9;
}

[data-theme="elegant-dark"] .markdown-link:hover {
  color: #bbdefb;
}

[data-theme="elegant-dark"] .preview-content {
  color: #e0e0e0;
}

[data-theme="elegant-dark"] .note-content-input {
  background: #2d2d30;
  color: #e0e0e0;
  border-color: #3e3e42;
}

[data-theme="elegant-dark"] .file-item:hover {
  background-color: #2d2d30;
}

[data-theme="elegant-dark"] .app-item:hover {
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.3);
}

[data-theme="elegant-dark"] .tree-node-content:hover {
  background-color: #2d2d30;
}

[data-theme="elegant-dark"] .conversation-item:hover {
  background-color: #2d2d30;
}

/* 主题切换样式 - 天青蓝主题下的Markdown样式 */
[data-theme="ocean-blue"] .markdown-code {
  background: #e0f2fe;
  border-color: #bae6fd;
  color: #0c4a6e;
}

[data-theme="ocean-blue"] .markdown-link {
  color: #0ea5e9;
}

[data-theme="ocean-blue"] .markdown-link:hover {
  color: #0284c7;
}

/* 炫酷黑主题 - 分类标题悬停样式 */
[data-theme="elegant-dark"] .category-header:hover {
  background: var(--hover-color) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

/* 炫酷黑主题 - 分类应用样式 */
[data-theme="elegant-dark"] .category-apps {
  background: transparent !important;
}

[data-theme="elegant-dark"] .category-app-item {
  background: transparent !important;
}

[data-theme="elegant-dark"] .category-app-item:hover {
  background: var(--hover-color) !important;
}



/* 炫酷黑主题 - 应用网格项 */
[data-theme="elegant-dark"] .recent-app-grid-item {
  background: var(--sidebar-bg) !important;
  border: 1px solid var(--border-color) !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2) !important;
}

[data-theme="elegant-dark"] .recent-app-grid-item:hover {
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.3) !important;
}

/* 炫酷黑主题 - 主内容区域 */
[data-theme="elegant-dark"] .main-content {
  background: var(--secondary-color) !important;
}

/* 炫酷黑主题 - 无会话状态 */
[data-theme="elegant-dark"] .no-conversation {
  background: var(--secondary-color) !important;
  color: var(--text-color) !important;
}

/* 炫酷黑主题 - 应用内容区域 */
[data-theme="elegant-dark"] .apps-content {
  background: var(--secondary-color) !important;
}

[data-theme="elegant-dark"] .recent-apps-section,
[data-theme="elegant-dark"] .all-apps-section {
  background: var(--secondary-color) !important;
}

/* 天青蓝主题 - 选项项样式调整 */
[data-theme="ocean-blue"] .option-item {
  color: rgba(255, 255, 255, 0.8);
}

[data-theme="ocean-blue"] .option-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

[data-theme="ocean-blue"] .option-item.active {
  /* background: rgba(255, 255, 255, 0.2); */
  color: white;
}

/* 高雅紫主题 */
[data-theme="elegant-purple"] {
  --primary-color: #7e22ce;
  --primary-light: #f3e8ff;
  --secondary-color: #faf5ff;
  --text-color: #5b21b6;
  --border-color: #e9d5ff;
  --hover-color: #f3e8ff;
  --active-color: #6b21a8;
  --sidebar-bg: #ffffff;
  --window-controls-bg: #ffffff;
  --context-menu-bg: #ffffff;
  --context-menu-hover: #f3e8ff;
  --accent-color: #a855f7;
  --text-secondary: #7e22ce;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  --panel-bg: #ffffff;
  --list-bg: #faf5ff;
  --card-bg: #ffffff;
  --header-panel-bg: #f3e8ff;
}

/* 高雅紫主题 - 左边侧边栏 */
[data-theme="royal-purple"] .side-options {
  background: linear-gradient(135deg, #7e22ce 0%, #a855f7 100%);
}

/* 高雅紫主题 - 文本颜色 */
[data-theme="royal-purple"] .window-title,
[data-theme="royal-purple"] .option-item,
[data-theme="royal-purple"] .option-item.active {
  color: white;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}

/* 高雅紫主题 - 侧边栏头部 */
[data-theme="royal-purple"] .sidebar-header {
  background: #ffffff;
  /* box-shadow: var(--shadow-md); */
}

[data-theme="royal-purple"] .sidebar-header .user-name {
  color: var(--text-color);
  text-shadow: none;
}

/* 高雅紫主题 - 窗口控制栏左侧 */
[data-theme="royal-purple"] .window-controls-left {
  background: linear-gradient(135deg, #7e22ce 0%, #a855f7 100%);
}

/* 月牙黄主题 */
[data-theme="warm-amber"] {
  --primary-color: #d4b85f;
  --primary-light: #fffef8;
  --secondary-color: #fffef8;
  --text-color: #6b5a2f;
  --border-color: #f0e6c8;
  --hover-color: #fffef8;
  --active-color: #c9a85a;
  --sidebar-bg: #ffffff;
  --window-controls-bg: #ffffff;
  --context-menu-bg: #ffffff;
  --context-menu-hover: #fffef8;
  --accent-color: #e8d4a0;
  --text-secondary: #8b7a50;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.03);
  --shadow-md: 0 2px 4px -1px rgba(0, 0, 0, 0.05), 0 1px 2px -1px rgba(0, 0, 0, 0.03);
  --panel-bg: #ffffff;
  --list-bg: #fffef8;
  --card-bg: #ffffff;
  --header-panel-bg: #fffef8;
}

/* 月牙黄主题 - 左边侧边栏 */
[data-theme="warm-amber"] .side-options {
  background: linear-gradient(135deg, #e8d4a0 0%, #f0e2b8 100%);
}

/* 月牙黄主题 - 文本颜色 */
[data-theme="warm-amber"] .window-title,
[data-theme="warm-amber"] .option-item,
[data-theme="warm-amber"] .option-item.active {
  color: #5a4a25;
  text-shadow: 0 1px 1px rgba(255, 255, 255, 0.6);
}

/* 月牙黄主题 - 侧边栏头部 */
[data-theme="warm-amber"] .sidebar-header {
  /* background: #fffef8; */
   color: var(--text-color);
  text-shadow: none;
  /* box-shadow: var(--shadow-md); */
}

[data-theme="warm-amber"] .sidebar-header .user-name {
  color: var(--text-color);
  text-shadow: none;
}

/* 月牙黄主题 - 窗口控制栏左侧 */
[data-theme="warm-amber"] .window-controls-left {
  background: linear-gradient(135deg, #e8d4a0 0%, #f0e2b8 100%);
}

/* 中国红主题 */
[data-theme="crimson-red"] {
  --primary-color: #c41e3a;
  --primary-light: #fff5f5;
  --secondary-color: #fff5f5;
  --text-color: #5c1a1a;
  --border-color: #f5c6c6;
  --hover-color: #fff5f5;
  --active-color: #a01830;
  --sidebar-bg: #ffffff;
  --window-controls-bg: #ffffff;
  --context-menu-bg: #ffffff;
  --context-menu-hover: #fff5f5;
  --accent-color: #e85c6c;
  --text-secondary: #8b3a3a;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.03);
  --shadow-md: 0 2px 4px -1px rgba(0, 0, 0, 0.05), 0 1px 2px -1px rgba(0, 0, 0, 0.03);
  --panel-bg: #ffffff;
  --list-bg: #fff5f5;
  --card-bg: #ffffff;
  --header-panel-bg: #fff5f5;
}

/* 中国红主题 - 左边侧边栏 */
[data-theme="crimson-red"] .side-options {
  background: linear-gradient(135deg, #c41e3a 0%, #e85c6c 100%);
}

/* 中国红主题 - 文本颜色 */
[data-theme="crimson-red"] .window-title,
[data-theme="crimson-red"] .option-item,
[data-theme="crimson-red"] .option-item.active {
  color: #fff5f5;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}

/* 中国红主题 - 侧边栏头部 */
[data-theme="crimson-red"] .sidebar-header {
  /* background: #fff5f5; */
  /* box-shadow: var(--shadow-md); */
   color: var(--text-color);
  text-shadow: none;
}

[data-theme="crimson-red"] .sidebar-header .user-name {
  color: var(--text-color);
  text-shadow: none;
}

/* 中国红主题 - 窗口控制栏左侧 */
[data-theme="crimson-red"] .window-controls-left {
  background: linear-gradient(135deg, #c41e3a 0%, #e85c6c 100%);
}

/* 青草绿主题 */
[data-theme="emerald-green"] {
  --primary-color: #2e8b57;
  --primary-light: #f0fff4;
  --secondary-color: #f0fff4;
  --text-color: #1a4a2e;
  --border-color: #c6e6d5;
  --hover-color: #f0fff4;
  --active-color: #247048;
  --sidebar-bg: #ffffff;
  --window-controls-bg: #ffffff;
  --context-menu-bg: #ffffff;
  --context-menu-hover: #f0fff4;
  --accent-color: #5cb88a;
  --text-secondary: #2e6b4a;
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.03);
  --shadow-md: 0 2px 4px -1px rgba(0, 0, 0, 0.05), 0 1px 2px -1px rgba(0, 0, 0, 0.03);
  --panel-bg: #ffffff;
  --list-bg: #f0fff4;
  --card-bg: #ffffff;
  --header-panel-bg: #f0fff4;
}

/* 青草绿主题 - 左边侧边栏 */
[data-theme="emerald-green"] .side-options {
  background: linear-gradient(135deg, #2e8b57 0%, #5cb88a 100%);
}

/* 青草绿主题 - 文本颜色 */
[data-theme="emerald-green"] .window-title,
[data-theme="emerald-green"] .option-item,
[data-theme="emerald-green"] .option-item.active {
  color: #f0fff4;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
}

/* 青草绿主题 - 侧边栏头部 */
[data-theme="emerald-green"] .sidebar-header {
  /* background: #f0fff4; */
  /* box-shadow: var(--shadow-md); */
   color: var(--text-color);
  text-shadow: none;
}

[data-theme="emerald-green"] .sidebar-header .user-name {
  color: var(--text-color);
  text-shadow: none;
}

/* 青草绿主题 - 窗口控制栏左侧 */
[data-theme="emerald-green"] .window-controls-left {
  background: linear-gradient(135deg, #2e8b57 0%, #5cb88a 100%);
}

/* 语音通话模态框样式 */
.voice-call-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.8);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
}

.voice-call-content {
  width: 360px;
  max-width: 90vw;
  background-color: var(--context-menu-bg);
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  animation: slideIn 0.3s ease;
}

@keyframes slideIn {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.voice-call-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  background-color: var(--list-bg);
  text-align: center;
}

.voice-call-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
}

.voice-call-body {
  flex: 1;
  padding: 40px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.call-status {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.call-status-text {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-color);
  font-size: 16px;
  margin-bottom: 20px;
}

.call-status-text i {
  font-size: 48px;
  color: var(--primary-color);
  margin-bottom: 16px;
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.1);
    opacity: 0.8;
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
}

.call-duration {
  font-size: 24px;
  font-weight: 600;
  color: var(--primary-color);
  margin-top: 12px;
}

.voice-call-footer {
  padding: 20px;
  border-top: 1px solid var(--border-color);
  display: flex;
  justify-content: center;
}

.end-call-btn {
  padding: 12px 32px;
  background-color: #f44336;
  color: white;
  border: none;
  border-radius: 30px;
  cursor: pointer;
  font-size: 16px;
  font-weight: 500;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  box-shadow: 0 2px 8px rgba(244, 67, 54, 0.4);
}

.end-call-btn:hover {
  background-color: #d32f2f;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(244, 67, 54, 0.6);
}

.end-call-btn i {
  margin-right: 8px;
  font-size: 18px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .voice-call-content {
    width: 300px;
  }
  
  .call-status-text i {
    font-size: 36px;
  }
  
  .call-duration {
    font-size: 20px;
  }
  
  .end-call-btn {
    padding: 10px 24px;
    font-size: 14px;
  }
}

/* 网络错误提示 */
.network-error {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: transparent;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  animation: fadeIn 0.3s ease;
}

.network-error-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: rgba(254, 240, 240, 0.9);
  border: none;
  border-radius: 8px;
  padding: 20px;
  max-width: 100%;
  width: 95%;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  min-height: 120px;
  max-height: 200px;
}

.error-icon {
  color: #f56c6c;
  font-size: 32px;
  margin-bottom: 16px;
  flex-shrink: 0;
}

.error-message {
  text-align: center;
  margin-bottom: 20px;
}

.error-message p {
  margin: 0 0 8px 0;
  color: #f56c6c;
  font-size: 16px;
}

.error-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.retry-btn, .login-btn {
  padding: 6px 16px;
  border: none;
  border-radius: 4px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.retry-btn {
  background: rgba(255, 255, 255, 0.8);
  color: #f56c6c;
}

.retry-btn:hover {
  background: #f56c6c;
  color: white;
}

.login-btn {
  background: #f56c6c;
  color: white;
}

.login-btn:hover {
  background: #f5222d;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

/* 响应式设计 */
@media (max-width: 768px) {
  .network-error-content {
    padding: 20px;
    max-width: 90%;
  }
  
  .error-icon {
    font-size: 24px;
  }
  
  .error-message p {
    font-size: 14px;
  }
}
</style>

<style scoped>
.im-container {
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100%;
  background: var(--content-bg);
  color: var(--text-color);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}

/* 自定义窗口控制栏 */
.window-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 40px;
  background: var(--window-controls-bg);
  padding: 0 20px;
  user-select: none;
  -webkit-app-region: drag;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.window-controls-left {
  display: flex;
  align-items: center;
  width: 72px;
  height: 100%;
  justify-content: center;
  background: var(--list-bg);
  margin: 0 -20px;
  padding: 0 20px;
}

.window-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
  letter-spacing: 0.5px;
}

.window-controls-right {
  display: flex;
  align-items: center;
  gap: 12px;
  -webkit-app-region: no-drag;
}

.window-control-btn {
  width: 16px;
  height: 16px;
  border: none;
  border-radius: 50%;
  font-size: 11px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  -webkit-app-region: no-drag;
  line-height: 1;
  box-shadow: var(--shadow-sm);
}

.minimize-btn {
  background: #ffbd2e;
  color: white;
}

.maximize-btn {
  background: #27c93f;
  color: white;
}

.close-btn {
  background: #ff5f56;
  color: white;
}

.window-control-btn:hover {
  transform: scale(1.05);
  opacity: 0.9;
}

/* 主内容区域 */
.main-content-area {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* 左侧垂直选项栏样式 */
.side-options {
  width: 72px;
  background: var(--list-bg);
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 20px 0;
  gap: 20px;
  /* box-shadow: 1px 0 3px rgba(0, 0, 0, 0.08); */
  transition: all 0.3s ease;
  z-index: 10;
}

.side-options:hover {
  box-shadow: var(--shadow-md);
}

.option-spacer {
  flex: 0.9;
}

.settings-option {
  /* top:-20px; */
  /* transform: translateY(-24px); */
  transition: none;
  position: relative;
}

.settings-option:hover {
  transform: translateY(-24px);
  transition: none;
  box-shadow: none;
  background: var(--hover-color);
  color: var(--primary-color);
  position: relative;
}

/* 选项项样式 */
.option-item {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--text-secondary);
}

.option-item:hover {
  background: var(--hover-color);
  color: var(--primary-color);
  box-shadow: var(--shadow-sm);
}

.option-item.active {
  background: var(--primary-color);
  color: white;
  box-shadow: var(--shadow-md);
}

.option-icon {
  font-size: 20px;
}

/* 主题图标样式 */
.theme-icon {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  display: inline-block;
  margin-right: 8px;
  box-shadow: var(--shadow-sm);
}

/* ========================================
   主题图标样式 - Theme Icon Styles
   ======================================== */
.modern-light-theme {
  background: #3b82f6;
  border: 1px solid #2563eb;
}

.elegant-dark-theme {
  background: #1e293b;
  border: 1px solid #334155;
}

.ocean-blue-theme {
  background: #0ea5e9;
  border: 1px solid #0284c7;
}

.elegant-purple-theme {
  background: #8b5cf6;
  border: 1px solid #7c3aed;
}

.warm-amber-theme {
  background: #f59e0b;
  border: 1px solid #d97706;
}

.crimson-red-theme {
  background: #dc2626;
  border: 1px solid #b91c1c;
}

.emerald-green-theme {
  background: #10b981;
  border: 1px solid #059669;
}

.green-theme {
  background: #10b981;
  border: 1px solid #059669;
}

/* 关于对话框样式 */
.about-dialog-overlay {
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
}

.about-dialog {
  background: var(--context-menu-bg);
  border-radius: 12px;
  width: 400px;
  max-width: 90%;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  overflow: hidden;
}

.about-dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: var(--list-bg);
  box-shadow: 0 1px 0 rgba(0, 0, 0, 0.05);
}

.about-dialog-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
}

.about-dialog-close {
  border: none;
  background: transparent;
  font-size: 20px;
  cursor: pointer;
  color: var(--text-secondary);
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.about-dialog-close:hover {
  background: var(--hover-color);
  color: var(--text-color);
  transform: scale(1.05);
}

.about-dialog-content {
  padding: 32px 24px;
  text-align: center;
  background: var(--context-menu-bg);
}

.about-dialog-logo {
  margin-bottom: 16px;
  color: var(--primary-color);
  font-size: 24px;
}

.about-dialog-content h2 {
  margin: 0 0 12px 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-color);
}

.about-dialog-content .version {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.about-dialog-content .date {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.about-dialog-content .author {
  margin: 0 0 8px 0;
  font-size: 14px;
  color: var(--text-secondary);
}

.about-dialog-content .copyright {
  margin: 0 0 16px 0;
  font-size: 14px;
  color: var(--text-secondary);
  opacity: 0.8;
}

.about-dialog-content .description {
  margin: 0;
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.6;
}

.about-dialog-footer {
  padding: 0 24px 24px;
  display: flex;
  justify-content: center;
  background: var(--context-menu-bg);
}

.about-dialog-button {
  padding: 10px 28px;

  background: var(--list-bg);
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  color: var(--text-color);
  box-shadow: var(--shadow-sm);
  border: 1px solid transparent;
}

.about-dialog-button:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

/* 退出登录对话框样式 */
.logout-dialog-overlay {
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
}

.logout-dialog {
  background: white;
  border-radius: 8px;
  width: 360px;
  max-width: 90%;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.logout-dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;

}

.logout-dialog-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #333;
}

.logout-dialog-close {
  border: none;
  background: transparent;
  font-size: 20px;
  cursor: pointer;
  color: var(--text-secondary);
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.logout-dialog-close:hover {
  background: var(--hover-color);
  color: var(--text-color);
  transform: scale(1.05);
}

.logout-dialog-content {
  padding: 32px 20px;
  text-align: center;
}

.logout-dialog-message {
  margin: 0;
  font-size: 16px;
  color: #333;
  line-height: 1.5;
}

.logout-dialog-footer {
  padding: 0 20px 20px;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.logout-dialog-button {
  padding: 10px 28px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  min-width: 80px;
  border: 1px solid transparent;
  box-shadow: var(--shadow-sm);
}

.logout-dialog-button.cancel-button {
  background: var(--list-bg);
  color: var(--text-color);
}

.logout-dialog-button.cancel-button:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.logout-dialog-button.confirm-button {
  background: #ff4d4f;
  border-color: #ff4d4f;
  color: white;
}

.logout-dialog-button.confirm-button:hover {
  background: #ff7875;
  border-color: #ff7875;
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

/* 检查更新对话框样式 */
.update-dialog-overlay {
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
}

.update-dialog {
  background: white;
  border-radius: 8px;
  width: 360px;
  max-width: 90%;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  overflow: hidden;
}

.update-dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;

}

.update-dialog-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #333;
}

.update-dialog-close {
  border: none;
  background: transparent;
  font-size: 20px;
  cursor: pointer;
  color: var(--text-secondary);
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.update-dialog-close:hover {
  background: var(--hover-color);
  color: var(--text-color);
  transform: scale(1.05);
}

.update-dialog-content {
  padding: 32px 20px;
  text-align: center;
}

.update-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #f3f3f3;
  border-top: 3px solid #1890ff;
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.loading-text {
  margin: 0;
  font-size: 14px;
  color: #666;
}

.update-downloading {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.download-icon {
  font-size: 48px;
  color: #1890ff;
  margin-bottom: 8px;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0% {
    transform: scale(1);
  }
  50% {
    transform: scale(1.1);
  }
  100% {
    transform: scale(1);
  }
}

.download-text {
  margin: 0;
  font-size: 14px;
  color: #666;
}

.download-progress {
  width: 100%;
  max-width: 240px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.progress-bar {
  width: 100%;
  height: 8px;
  background: #f0f0f0;
  border-radius: 4px;
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  background: linear-gradient(90deg, #1890ff, #40a9ff);
  border-radius: 4px;
  transition: width 0.3s ease;
}

.progress-text {
  margin: 0;
  font-size: 12px;
  color: #999;
  text-align: center;
}

.update-result {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
}

.result-icon {
  font-size: 48px;
  color: #52c41a;
  margin-bottom: 8px;
  transition: all 0.3s ease;
}

.result-icon.new-version {
  color: #faad14;
  animation: pulse 1.5s ease-in-out infinite;
}

.result-text {
  margin: 0;
  font-size: 16px;
  color: #333;
  font-weight: 500;
}

.version-info {
  margin: 0;
  font-size: 14px;
  color: #999;
}

.update-dialog-footer {
  padding: 0 20px 20px;
  display: flex;
  justify-content: center;
  gap: 12px;
}

.update-dialog-button {
  padding: 10px 28px;
  background: var(--list-bg);
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
  min-width: 80px;
  border: 1px solid transparent;
  color: var(--text-color);
  box-shadow: var(--shadow-sm);
}

.update-dialog-button:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.update-dialog-button.update-button {
  background: #1890ff;
  border-color: #1890ff;
  color: white;
}

.update-dialog-button.update-button:hover {
  background: #40a9ff;
  border-color: #40a9ff;
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.option-item {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s;
  color: var(--text-color);
}

.option-item:hover {
  background: var(--hover-color);
  color: var(--primary-color);
}

.option-item.active {
  background: var(--active-color);
  color: #fff;
}

.option-icon {
  font-size: 18px;
  opacity: 0.8;
  transition: all 0.2s;
}

.option-item:hover .option-icon,
.option-item.active .option-icon {
  opacity: 1;
  transform: scale(1.1);
}

/* 内容区域样式 */
.content-area {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.content-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}





/* 组织架构样式 */
.org-structure {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
}

.org-header {
  padding: 16px 20px;
  background: var(--header-panel-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.org-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
  color: #333;
}

.org-content {
  flex: 1;
  padding: 0;
  overflow-y: auto;
}

.department {
  margin-bottom: 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  overflow: hidden;
}

.department-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  cursor: pointer;
  background: var(--list-bg);
  transition: background 0.2s;
}

.department-header:hover {
  background: var(--hover-color);
}

.department-name {
  font-weight: 500;
  color: #333;
  font-size: 14px;
}

.toggle-icon {
  font-size: 12px;
  color: #666;
  transition: transform 0.2s;
}

.sub-departments {
  padding-left: 24px;
}

.sub-department {
  margin: 8px 0;
}

.sub-department-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 6px;
  transition: background 0.2s;
}

.sub-department-header:hover {
  background: #f5f5f5;
}

.employees {
  padding-left: 24px;
  padding: 12px;
  background: transparent;
  border-radius: 6px;
  margin-top: 8px;
}

.employee-item {
  display: flex;
  align-items: center;
  padding: 8px 0;
  gap: 12px;
}

.employee-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
}


/* 群聊样式 */
.groups-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
}

.groups-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: var(--header-panel-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.groups-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
  color: var(--text-color);
}

.create-group-btn {
  padding: 6px 16px;
  background: var(--primary-color);
  color: #fff;
  border: none;
  border-radius: 16px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.2s;
}

.create-group-btn:hover {
  opacity: 0.9;
}

.groups-list {
  background: #fafafa;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  overflow: hidden;
  margin: 8px 8px;
  max-height: calc(100vh - 200px);
  overflow-y: auto;
}

.group-item {
  display: flex;
  align-items: center;
  padding: 12px 20px;
  background: #fafafa;
  cursor: pointer;
  transition: background 0.2s;
}

.group-item:hover {
  background: var(--hover-color);
}

.group-avatar {
  position: relative;
  margin-right: 12px;
}

.group-avatar img {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  object-fit: cover;
}

.group-badge {
  position: absolute;
  bottom: 0;
  right: 0;
  background: var(--primary-color);
  color: white;
  font-size: 10px;
  padding: 2px 4px;
  border-radius: 4px;
}

.group-info {
  flex: 1;
  min-width: 0;
}

.group-name {
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.group-preview {
  font-size: 13px;
  color: var(--text-color);
  opacity: 0.7;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.group-meta {
  text-align: right;
  margin-left: 8px;
}

.group-time {
  font-size: 12px;
  color: var(--text-color);
  opacity: 0.6;
  margin-bottom: 4px;
}

/* 主内容区域样式 */
.main-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

/* 群聊样式 */
.groups-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
}

.groups-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: var(--header-panel-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.groups-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
  color: var(--text-color);
}

.create-group-btn {
  padding: 6px 16px;
  background: var(--primary-color);
  color: #fff;
  border: none;
  border-radius: 16px;
  font-size: 13px;
  cursor: pointer;
  transition: background 0.2s;
}

.create-group-btn:hover {
  opacity: 0.9;
}

.groups-list {
  background: #fafafa;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  overflow: hidden;
  margin: 8px 8px;
  max-height: calc(100vh - 200px);
  overflow-y: auto;
}

.group-item {
  display: flex;
  align-items: center;
  padding: 12px 20px;
  background: #fafafa;
  cursor: pointer;
  transition: background 0.2s;
}

.group-item:hover {
  background: var(--hover-color);
}

.group-avatar {
  position: relative;
  margin-right: 12px;
}

.group-avatar img {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  object-fit: cover;
}

.group-badge {
  position: absolute;
  bottom: 0;
  right: 0;
  background: var(--primary-color);
  color: white;
  font-size: 10px;
  padding: 2px 4px;
  border-radius: 4px;
}

.group-info {
  flex: 1;
  min-width: 0;
}

.group-name {
  font-weight: 500;
  color: var(--text-color);
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.group-preview {
  font-size: 13px;
  color: var(--text-color);
  opacity: 0.7;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.group-meta {
  text-align: right;
  margin-left: 8px;
}

.group-time {
  font-size: 12px;
  color: var(--text-color);
  opacity: 0.6;
  margin-bottom: 4px;
}

/* 应用样式 */
/* 右边面板应用内容样式 */
.apps-content {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
}

.apps-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--content-bg);
  overflow: hidden;
}

.apps-header {
  padding: 16px 20px;
  background: var(--header-panel-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.apps-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
  color: var(--text-color);
}

.apps-grid {
  flex: 1;
  /* padding: 16px; */
  /* background: var(--content-bg); */
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  overflow-y: auto;
  justify-content: flex-start;
  align-content: flex-start;
}

.app-item {
  width: 120px;
  height: 120px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: var(--list-bg);
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: center;
}

.app-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  border-color: var(--hover-color);
  background: var(--hover-color);
  opacity: 0.9;
}

.app-icon {
  font-size: 28px;
  margin-bottom: 8px;
  width: 48px;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: var(--list-bg);
  color: var(--primary-color);
  transition: all 0.2s ease;
  opacity: 1;
}

.app-item:hover .app-icon {
  transform: scale(1.05);
  background: var(--primary-color);
  color: #fff;
}

.app-name {
  font-size: 12px;
  color: var(--text-color);
  font-weight: 500;
  transition: all 0.2s ease;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.app-item:hover .app-name {
  color: var(--primary-color);
}

/* 最近使用应用样式 */
.recent-apps-section {
  padding: 16px 0 32px 0;
  /* background: var(--content-bg); */
}

.section-header {
  margin-bottom: 16px;
}

.section-header h3 {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
  margin: 0;
}

.recent-apps-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  overflow-y: auto;
}

.recent-app-grid-item {
  width: 100px;
  height: 100px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 12px;
  background: var(--list-bg);
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: center;
}

.recent-app-grid-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
  background: var(--hover-color);
}

.recent-app-grid-icon {
  font-size: 28px;
  margin-bottom: 8px;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  background: var(--list-bg);
  color: var(--primary-color);
  transition: all 0.2s ease;
}

.recent-app-grid-item:hover .recent-app-grid-icon {
  transform: scale(1.05);
  background: var(--primary-color);
  color: #fff;
}

.recent-app-grid-name {
  font-size: 12px;
  color: var(--text-color);
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.empty-recent-apps {
  width: 100%;
  padding: 20px;
  text-align: center;
  color: var(--text-color);
  opacity: 0.6;
}




/* 用户信息弹窗样式 */
.user-profile-modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.user-profile-content {
  background: var(--panel-bg);
  border-radius: 8px;
  width: 500px;
  max-width: 90%;
  max-height: 80vh;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  overflow: hidden;
  border: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
}

.user-profile-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border-color);
}

.user-profile-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: var(--text-color);
}

.user-profile-header .close-btn {
  background: transparent;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-color);
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s;
  opacity: 0.6;
}

.user-profile-header .close-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
  opacity: 1;
}

.user-profile-body {
  padding: 24px;
  flex: 1;
  overflow-y: auto;
  min-height: 0;
}

.profile-avatar {
  display: flex;
  justify-content: center;
  margin-bottom: 24px;
  position: relative;
}

.profile-avatar img {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--border-color);
  transition: all 0.3s ease;
  background: var(--bg-secondary);
}

.avatar-clickable {
  cursor: pointer;
}

.avatar-clickable:hover {
  transform: scale(1.05);
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(var(--primary-rgb), 0.2);
}



.avatar-input {
  display: none;
}

.profile-info {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.info-item label {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}

.profile-input,
.profile-textarea {
  padding: 10px 14px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 14px;
  transition: border-color 0.2s;
  width: 100%;
  box-sizing: border-box;
  background: var(--content-bg);
  color: var(--text-color);
}

.profile-input:focus,
.profile-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(var(--primary-rgb), 0.2);
}

.profile-textarea {
  resize: vertical;
  min-height: 80px;
}

.profile-value {
  font-size: 14px;
  color: var(--text-color);
  font-weight: 500;
  padding: 8px 12px;
  background: var(--bg-secondary);
  border-radius: 4px;
  border: 1px solid var(--border-color);
  transition: all 0.3s ease;
}

.profile-value:hover {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(var(--primary-rgb), 0.1);
}

.profile-actions {
  margin-top: 24px;
  display: flex;
  gap: 12px;
  justify-content: center;
  padding: 0 20px;
}

.action-btn {
  padding: 10px 24px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 4px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.3s ease;
}

.action-btn:hover {
  background: var(--primary-hover);
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(var(--primary-rgb), 0.3);
}

/* 组织架构用户信息样式 */
/* 组织架构用户信息 */
.user-profile-container {
  position: relative;
  padding: 16px;
  margin: 10px 5px;
}

.user-profile-header-bg {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 80px;
  background: linear-gradient(135deg, var(--primary-light), var(--active-color));
  border-radius: 8px 8px 0 0;
  z-index: 1;
}

.user-profile-card {
  position: relative;
  background: var(--card-bg);
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  border: 1px solid var(--border-color);
  z-index: 2;
  margin-top: 40px;
  animation: cardSlideIn 0.4s ease-out;
}

@keyframes cardSlideIn {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

.user-profile-avatar-section {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-color);
}

.user-avatar-container {
  position: relative;
  margin-right: 16px;
}

.user-avatar {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  object-fit: cover;
  border: 3px solid white;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
  transition: transform 0.3s ease;
}

.user-avatar:hover {
  transform: scale(1.05);
}

.online-status-indicator {
  position: absolute;
  bottom: 2px;
  right: 2px;
  width: 14px;
  height: 14px;
  background: #4caf50;
  border-radius: 50%;
  border: 2px solid white;
  box-shadow: 0 2px 8px rgba(76, 175, 80, 0.4);
  animation: onlinePulse 2s ease-in-out infinite;
}

@keyframes onlinePulse {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(76, 175, 80, 0.7);
  }
  50% {
    box-shadow: 0 0 0 8px rgba(76, 175, 80, 0);
  }
}

.user-basic-info {
  flex: 1;
}

.user-full-name {
  margin: 0 0 6px 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--text-color);
  letter-spacing: 0.5px;
}

.user-department {
  margin: 0 0 3px 0;
  font-size: 13px;
  color: var(--primary-color);
  font-weight: 600;
}

.user-position {
  margin: 0;
  font-size: 12px;
  color: var(--text-secondary);
}

.user-info-sections {
  margin-bottom: 20px;
}

.info-section {
  margin-bottom: 16px;
  background: var(--list-bg);
  border-radius: 8px;
  padding: 16px;
  border: 1px solid var(--border-color);
  transition: all 0.3s ease;
}

.info-section:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  transform: translateY(-2px);
}

.section-title {
  display: flex;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-color);
}

.section-title i {
  font-size: 16px;
  color: var(--primary-color);
  margin-right: 8px;
}

.section-title h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-color);
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.info-label {
  font-size: 11px;
  color: #64748b;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.info-value {
  font-size: 13px;
  color: var(--text-color);
  font-weight: 500;
  padding: 5px 8px;
  background: var(--input-bg);
  border-radius: 4px;
  border: 1px solid var(--border-color);
  transition: all 0.2s ease;
}

.info-value:hover {
  border-color: var(--primary-color);
  box-shadow: 0 2px 8px rgba(102, 126, 234, 0.15);
}

.user-action-buttons {
  display: flex;
  gap: 10px;
  justify-content: center;
  padding-top: 16px;
  border-top: 1px solid var(--border-color);
}

.action-btn {
  padding: 10px 20px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 100px;
  justify-content: center;
}

.action-btn.primary {
  background: var(--primary-color);
  border-color: var(--primary-color);
  color: white;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
}

.action-btn.primary:hover {
  background: var(--active-color);
  border-color: var(--active-color);
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(102, 126, 234, 0.4);
}

.action-btn.secondary {
  background: white;
  border-color: var(--border-color);
  color: var(--text-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.action-btn.secondary:hover {
  border-color: var(--primary-color);
  color: var(--primary-color);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.2);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .user-profile-container {
    padding: 10px;
  }
  
  .user-profile-card {
    padding: 20px;
    margin-top: 40px;
  }
  
  .user-profile-avatar-section {
    flex-direction: column;
    text-align: center;
    gap: 15px;
  }
  
  .user-avatar-container {
    margin-right: 0;
  }
  
  .info-grid {
    grid-template-columns: 1fr;
  }
  
  .user-action-buttons {
    flex-direction: column;
  }
  
  .action-btn {
    width: 100%;
  }
}

/* 成员搜索框样式 */
.member-search-box {
  margin-bottom: 8px;
}

.member-search-input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  font-size: 14px;
  outline: none;
  transition: all 0.2s;
  background: var(--panel-bg);
  color: var(--text-color);
}

.member-search-input:focus {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.1);
}

/* 类型选择器 */
.type-selector {
  display: flex;
  gap: 20px;
  margin-top: 8px;
}

.type-option {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.type-option input[type="radio"] {
  width: 16px;
  height: 16px;
}

/* 头像上传 */
.avatar-upload {
  margin-top: 8px;
}

.avatar-preview {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  overflow: hidden;
  border: 2px dashed var(--border-color);
  cursor: pointer;
  transition: all 0.3s ease;
}

.avatar-preview:hover {
  border-color: var(--primary-color);
  transform: scale(1.05);
}

.avatar-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background-color: var(--hover-color);
  color: var(--text-secondary);
}

.avatar-placeholder i {
  font-size: 24px;
  margin-bottom: 8px;
}

.avatar-placeholder span {
  font-size: 12px;
}

/* 成员选择器样式 */
.member-selector {
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 10px;
  background: var(--list-bg);
  margin-top: 10px;
}

.member-item {
  display: flex;
  align-items: center;
  padding: 10px;
  cursor: pointer;
  border-radius: 8px;
  transition: background-color 0.2s ease;
  margin-bottom: 8px;
}

.member-item:hover {
  background-color: var(--hover-color);
}

.member-item input[type="checkbox"] {
  margin-right: 12px;
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.member-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
}

.member-info .member-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  overflow: hidden;
}

.member-info .member-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.member-info span {
  font-size: 14px;
  color: var(--text-primary);
}

/* 成员上下文菜单样式 */
.context-menu-icon {
  margin-right: 10px;
  font-size: 16px;
  width: 20px;
  text-align: center;
}

/* 群聊上下文菜单样式 */
.context-menu-divider {
  height: 1px;
  background: #e0e0e0;
  margin: 4px 0;
  cursor: default;
}

.context-menu-divider:hover {
  background: #e0e0e0;
}

/* 动作菜单样式 */
.action-menu {
  position: fixed;
  background: var(--context-menu-bg);

  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 1000;
  min-width: 160px;
  overflow: hidden;
}

.action-menu-item {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 0.2s;
  font-size: 14px;
  color: var(--text-color);
}

.action-menu-item:hover {
  background: var(--context-menu-hover);
}

.action-menu-icon {
  margin-right: 10px;
  font-size: 16px;
  width: 20px;
  text-align: center;
  color: var(--primary-color);
}

/* 用户右键菜单样式 */
.user-context-menu {
  position: fixed;
  background: var(--context-menu-bg);

  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  z-index: 1000;
  min-width: 160px;
  overflow: hidden;
}

.user-context-menu-item {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 0.2s;
  font-size: 13px;
  color: var(--text-color);
}

.user-context-menu-item:hover {
  background: var(--context-menu-hover);
}

.user-context-menu-icon {
  margin-right: 10px;
  font-size: 16px;
  width: 20px;
  text-align: center;
  color: var(--primary-color);
}

.user-profile-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 24px;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border-color);
}

.cancel-btn,
.save-btn {
  padding: 8px 20px;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid transparent;
}

.cancel-btn {
  background: var(--panel-bg);
  color: var(--text-color);
  border-color: var(--border-color);
}

.cancel-btn:hover {
  background: var(--hover-color);
  border-color: var(--primary-color);
  color: var(--primary-color);
}

.save-btn {
  background: var(--primary-color);
  color: white;
}

.save-btn:hover {
  opacity: 0.85;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
}

/* 右侧内容样式 */
.right-content {
  flex: 1;
  background: var(--right-content-bg);
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
}

.right-content-header {
  padding: 16px 20px;
  background: var(--right-content-header-bg);
  height: 72px;
  box-sizing: border-box;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  /* border-bottom: 1px solid var(--border-color); */
}

.right-content-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  color: var(--text-color);
}

.right-content-body {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-color);
  font-size: 14px;
  opacity: 0.7;
  box-sizing: border-box;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

/* 右键菜单样式 */
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

/* 主题图标样式 */
.theme-icon {
  width: 16px !important;
  height: 16px !important;
  border-radius: 50%;
  display: inline-block;
  margin-right: 8px;
}

.modern-light-theme {
  background: #f5f5f5;
  border: 1px solid #e0e0e0;
}

.elegant-dark-theme {
  background: #333;
  border: 1px solid #555;
}

.ocean-blue-theme {
  background: #1976d2;
  border: 1px solid #1565c0;
}

.elegant-purple-theme {
  background: #8b5cf6;
  border: 1px solid #7c3aed;
}

.warm-amber-theme {
  background: #f59e0b;
  border: 1px solid #d97706;
}

.crimson-red-theme {
  background: #dc2626;
  border: 1px solid #b91c1c;
}

.emerald-green-theme {
  background: #10b981;
  border: 1px solid #059669;
}

.empty-state {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--right-content-bg);
}

.empty-content {
  text-align: center;
  color: var(--text-secondary);
}

.empty-icon {
  font-size: 64px;
  margin-bottom: 16px;
}

/* 添加成员模态框样式 */
.add-members-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.add-members-content {
  background-color: #fff;
  border-radius: 12px;
  width: 600px;
  max-width: 90%;
  max-height: 85vh;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
  display: flex;
  flex-direction: column;
}

.add-members-header {
  padding: 24px;
  border-bottom: 1px solid #e8e8e8;
  display: flex;
  justify-content: space-between;
  align-items: center;
  background-color: #fafafa;
}

.add-members-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #333;
}

.add-members-header .close-btn {
  background: transparent;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-color);
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s;
  opacity: 0.6;
}

.add-members-header .close-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
  opacity: 1;
}

.add-members-body {
  padding: 24px;
  overflow-y: auto;
  flex: 1;
}

.group-info {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
  padding: 16px;
  background-color: #f8f9fa;
  border-radius: 8px;
}

.group-avatar {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
}

.group-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.group-details {
  flex: 1;
}

.group-name {
  font-size: 16px;
  font-weight: 600;
  color: #333;
  margin-bottom: 4px;
}

.group-members-count {
  font-size: 14px;
  color: #666;
}

.search-section {
  margin-bottom: 20px;
}

.search-box {
  position: relative;
}

.search-input {
  width: 100%;
  padding: 10px 16px;
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  font-size: 14px;
  transition: all 0.3s;
}

.search-input:focus {
  outline: none;
  border-color: #1890ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}

.members-section {
  margin-bottom: 16px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e8e8e8;
}

.section-header span:first-child {
  font-size: 14px;
  font-weight: 600;
  color: #333;
}

.selected-count {
  font-size: 14px;
  color: #1890ff;
  font-weight: 500;
}

.members-list {
  max-height: 300px;
  overflow-y: auto;
  border: 1px solid #e8e8e8;
  border-radius: 8px;
}

.member-item {
  padding: 12px 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  transition: all 0.2s;
  border-bottom: 1px solid #f0f0f0;
}

.member-item:last-child {
  border-bottom: none;
}

.member-item:hover {
  background-color: #f5f5f5;
}

.member-item.selected {
  background-color: #e6f7ff;
}

.member-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
}

.member-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.member-info {
  flex: 1;
  min-width: 0;
}

.member-name {
  font-size: 14px;
  font-weight: 500;
  color: #333;
  margin-bottom: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.member-position {
  font-size: 12px;
  color: #999;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.member-checkbox {
  flex-shrink: 0;
}

.checkbox {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.member-actions {
  margin-left: auto;
  align-self: center;
}

.remove-member-btn {
  background: transparent;
  border: none;
  color: #f44336;
  cursor: pointer;
  font-size: 16px;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.3s ease;
}

.remove-member-btn:hover {
  background: rgba(244, 67, 54, 0.1);
  transform: scale(1.1);
}

.empty-state {
  padding: 40px 20px;
  text-align: center;
  color: #999;
  font-size: 14px;
}

.add-members-footer {
  padding: 20px 24px;
  border-top: 1px solid #e8e8e8;
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  background-color: #fafafa;
}

.add-members-footer .cancel-btn {
  padding: 10px 24px;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s;
  background-color: #fff;
  color: #333;
  font-weight: 500;
}

.add-members-footer .cancel-btn:hover {
  border-color: #1890ff;
  color: #1890ff;
  background-color: #f0f9ff;
}

.confirm-btn {
  padding: 10px 24px;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s;
  background-color: #1890ff;
  color: #fff;
  font-weight: 500;
  box-shadow: 0 2px 4px rgba(24, 144, 255, 0.2);
}

.confirm-btn:hover {
  background-color: #40a9ff;
  box-shadow: 0 4px 8px rgba(24, 144, 255, 0.3);
  transform: translateY(-1px);
}

.confirm-btn:disabled {
  background-color: #f0f0f0;
  color: #999;
  cursor: not-allowed;
  box-shadow: none;
  transform: none;
}

.confirm-btn:disabled:hover {
  background-color: #f0f0f0;
  box-shadow: none;
  transform: none;
}

/* 群资料详情样式 */
.group-details-section {
  margin-top: 24px;
  padding: 20px;
  background-color: #f8f9fa;
  border-radius: 8px;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 0;
  border-bottom: 1px solid #e9ecef;
}

.detail-item:last-child {
  border-bottom: none;
}

.detail-label {
  font-size: 14px;
  color: #6c757d;
  font-weight: 500;
}

.detail-value {
  font-size: 14px;
  color: #495057;
  font-weight: 400;
  text-align: right;
  flex: 1;
  margin-left: 20px;
}

/* 机器人徽章 */
.bot-badge {
  position: absolute;
  bottom: 0;
  right: 0;
  background: var(--primary-color);
  color: white;
  font-size: 10px;
  padding: 2px 4px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.bot-badge i {
  font-size: 10px;
}

/* 通知角标样式 */
.notification-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #f5222d;
  color: #fff;
  font-size: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.option-icon {
  position: relative;
}

/* 系统设置页面样式 */
.settings-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  backdrop-filter: blur(10px);
}

.settings-content {
  background: var(--sidebar-bg);
  border-radius: 12px;
  width: 90%;
  max-width: 900px;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.2);
  animation: modalFadeIn 0.3s ease;
}

.settings-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: var(--sidebar-bg);
  border-bottom: 1px solid var(--border-color);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.settings-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--text-color);
}

.settings-header .close-btn {
  background: transparent;
  border: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--text-color);
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s;
  opacity: 0.6;
}

.settings-header .close-btn:hover {
  background: var(--hover-color);
  color: var(--text-color);
  opacity: 1;
}

.settings-body {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.settings-sidebar {
  width: 200px;
  background: var(--sidebar-bg);
  border-right: 1px solid var(--border-color);
  padding: 20px 0;
  overflow-y: auto;
}

.settings-sidebar-item {
  display: flex;
  align-items: center;
  padding: 12px 24px;
  cursor: pointer;
  transition: all 0.3s ease;
  border-left: 3px solid transparent;
}

.settings-sidebar-item:hover {
  background: var(--hover-color);
}

.settings-sidebar-item.active {
  background: var(--hover-color);
  border-left-color: var(--primary-color);
  color: var(--primary-color);
}

.settings-sidebar-item i {
  margin-right: 12px;
  font-size: 16px;
}

.settings-main {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
  background: var(--content-bg);
}

.settings-section {
  margin-bottom: 32px;
}

.settings-section-header {
  margin-bottom: 16px;
}

.settings-section-header h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-color);
}

.settings-item {
  display: flex;
  align-items: center;
  margin-bottom: 16px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border-color);
}

.settings-item label {
  width: 120px;
  font-size: 14px;
  color: var(--text-secondary);
}

.settings-input {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 6px;
  background: var(--panel-bg);
  color: var(--text-color);
  font-size: 14px;
  transition: all 0.3s ease;
}

.settings-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.1);
}

.file-path-setting {
  display: flex;
  gap: 10px;
  align-items: center;
}

.browse-btn {
  padding: 8px 16px;
  background-color: var(--primary-color);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
  transition: background-color 0.3s;
}

.browse-btn:hover {
  background-color: var(--primary-hover);
}

.file-size-setting {
  display: flex;
  gap: 10px;
  align-items: center;
}

.size-unit {
  font-size: 14px;
  color: var(--text-secondary);
  white-space: nowrap;
}

.settings-hint {
  font-size: 12px;
  color: var(--text-secondary);
  margin-top: 5px;
  margin-left: 120px;
}

.settings-value {
  flex: 1;
  padding: 8px 12px;
  background: var(--input-bg);
  color: var(--text-color);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  font-size: 14px;
  pointer-events: none;
}

.settings-textarea {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--input-bg);
  color: var(--text-color);
  font-size: 14px;
  resize: vertical;
  min-height: 80px;
  transition: all 0.3s ease;
}

.settings-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(0, 122, 255, 0.2);
}

.settings-select {
  flex: 1;
  padding: 8px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--input-bg);
  color: var(--text-color);
  font-size: 14px;
  transition: all 0.3s ease;
}

.settings-select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 2px rgba(0, 122, 255, 0.2);
}

/* 开关样式 */
.switch {
  position: relative;
  display: inline-block;
  width: 50px;
  height: 24px;
}

.switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: #ccc;
  transition: .4s;
}

.slider:before {
  position: absolute;
  content: "";
  height: 16px;
  width: 16px;
  left: 4px;
  bottom: 4px;
  background-color: white;
  transition: .4s;
}

input:checked + .slider {
  background-color: var(--primary-color);
}

input:focus + .slider {
  box-shadow: 0 0 1px var(--primary-color);
}

input:checked + .slider:before {
  transform: translateX(26px);
}

.slider.round {
  border-radius: 24px;
}

.slider.round:before {
  border-radius: 50%;
}

/* 头像设置 */
.avatar-setting {
  flex: 1;
}

.current-avatar {
  display: flex;
  align-items: center;
  gap: 12px;
}

.current-avatar img {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--border-color);
}

.change-avatar-btn {
  padding: 6px 12px;
  border: 1px solid var(--primary-color);
  border-radius: 4px;
  background: transparent;
  color: var(--primary-color);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.change-avatar-btn:hover {
  background: var(--primary-color);
  color: white;
}

/* 主题选择器 */
.theme-selector {
  flex: 1;
  display: flex;
  gap: 20px;
}

.theme-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  transition: all 0.3s ease;
  padding: 8px;
  border-radius: 8px;
  border: 2px solid transparent;
}

.theme-option:hover {
  background: var(--hover-color);
}

.theme-option.active {
  border-color: var(--primary-color);
  background: var(--hover-color);
}

.theme-preview {
  width: 56px;
  height: 56px;
  border-radius: 8px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.1);
}

.theme-option span {
  font-size: 12px;
  color: var(--text-color);
  text-align: center;
}

.theme-preview.modern-light-theme {
  background: #ffffff;
  border: 1px solid #e8e8e8;
}

.theme-preview.elegant-dark-theme {
  background: #121212;
  border: 1px solid #333333;
}

.theme-preview.ocean-blue-theme {
  background: #e6f7ff;
  border: 1px solid #91d5ff;
}

.theme-preview.elegant-purple-theme {
  background: #f5f0ff;
  border: 1px solid #d3adf7;
}

.theme-preview.warm-amber-theme {
  background: linear-gradient(135deg, #e8d4a0 0%, #f0e2b8 100%);
  border: 1px solid #d4b85f;
}

.theme-preview.crimson-red-theme {
  background: linear-gradient(135deg, #dc2626 0%, #ef4444 100%);
  border: 1px solid #b91c1c;
}

.theme-preview.emerald-green-theme {
  background: linear-gradient(135deg, #10b981 0%, #34d399 100%);
  border: 1px solid #059669;
}

/* 字体大小滑块 */
.font-size-slider {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 12px;
}

.font-size-slider input[type="range"] {
  flex: 1;
  height: 4px;
  border-radius: 2px;
  background: var(--border-color);
  outline: none;
  -webkit-appearance: none;
}

.font-size-slider input[type="range"]::-webkit-slider-thumb {
  -webkit-appearance: none;
  appearance: none;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--primary-color);
  cursor: pointer;
}

.font-size-slider input[type="range"]::-moz-range-thumb {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: var(--primary-color);
  cursor: pointer;
  border: none;
}

.font-size-value {
  font-size: 14px;
  color: var(--text-secondary);
  min-width: 40px;
}

/* 按钮样式 */
.clear-cache-btn,
.security-btn {
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  border-radius: 4px;
  background: var(--input-bg);
  color: var(--text-color);
  font-size: 14px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.clear-cache-btn:hover,
.security-btn:hover {
  background: var(--hover-color);
}

/* 关于信息 */
.about-info {
  flex: 1;
  font-size: 14px;
  color: var(--text-secondary);
}

/* 设置页脚 */
.settings-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px 24px;
  background: var(--sidebar-bg);
  border-top: 1px solid var(--border-color);
  box-shadow: 0 -2px 4px rgba(0, 0, 0, 0.05);
}

/* 炫酷黑主题 - 系统设置 */
[data-theme="elegant-dark"] .settings-content {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="elegant-dark"] .settings-header {
  background: var(--sidebar-bg) !important;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.3) !important;
  border-bottom: 1px solid var(--border-color) !important;
}

[data-theme="elegant-dark"] .settings-header h3 {
  color: var(--text-color) !important;
}

[data-theme="elegant-dark"] .settings-sidebar {
  background: var(--sidebar-bg) !important;
  border-right: 1px solid var(--border-color) !important;
}

[data-theme="elegant-dark"] .settings-sidebar-item:hover {
  background: var(--hover-color) !important;
}

[data-theme="elegant-dark"] .settings-sidebar-item.active {
  background: var(--hover-color) !important;
  border-left-color: var(--primary-color) !important;
  color: var(--primary-color) !important;
}

[data-theme="elegant-dark"] .save-btn {
  background: var(--hover-color);
  color: white;
}

[data-theme="elegant-dark"] .settings-main {
  background: var(--secondary-color) !important;
}

[data-theme="elegant-dark"] .settings-section-header h4 {
  color: var(--text-color) !important;
}

[data-theme="elegant-dark"] icon-btn:hover {
  background: var(--hover-color)!important;
}

[data-theme="elegant-dark"] .settings-item label {
  color: var(--text-secondary) !important;
}

[data-theme="elegant-dark"] .settings-input,
[data-theme="elegant-dark"] .settings-textarea,
[data-theme="elegant-dark"] .settings-select {
  background: var(--input-bg) !important;
  color: var(--text-color) !important;
  border: 1px solid var(--border-color) !important;
}

[data-theme="elegant-dark"] .settings-input:focus,
[data-theme="elegant-dark"] .settings-textarea:focus,
[data-theme="elegant-dark"] .settings-select:focus {
  border-color: var(--primary-color) !important;
  box-shadow: 0 0 0 2px rgba(0, 122, 255, 0.2) !important;
}

[data-theme="elegant-dark"] .theme-option:hover {
  background: var(--hover-color) !important;
}

[data-theme="elegant-dark"] .theme-option.active {
  border-color: var(--primary-color) !important;
  background: var(--hover-color) !important;
}

[data-theme="elegant-dark"] .clear-cache-btn,
[data-theme="elegant-dark"] .security-btn {
  background: var(--input-bg) !important;
  color: var(--text-color) !important;
}

/* 频道详情样式 */
.channel-detail-content {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--right-content-bg);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  overflow: hidden;
}

.channel-detail-info {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: 20px 24px;
  background: var(--right-content-bg);
  transition: all 0.3s ease;
}

.channel-detail-info:hover {
  background: var(--hover-color);
}

.channel-meta {
  display: flex;
  gap: 16px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--text-tertiary);
  flex-wrap: wrap;
}

.channel-header-info {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  flex: 1;
}

.channel-header-avatar {
  width: 56px;
  height: 56px;
  border-radius: 28px;
  object-fit: cover;
  border: 2px solid var(--border-color);
  flex-shrink: 0;
  transition: all 0.3s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.channel-header-avatar:hover {
  transform: scale(1.05);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
  border-color: var(--primary-color);
}

.channel-header-text {
  flex: 1;
  min-width: 0;
}

.channel-description {
  margin: 0;
  font-size: 14px;
  color: var(--text-secondary);
  line-height: 1.5;
  margin-bottom: 4px;
}

.channel-header-actions {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.channel-header-actions .btn {
  padding: 8px 16px;
  border-radius: 20px;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 6px;
}

.channel-header-actions .btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
}

.channel-header-actions .btn-primary {
  background: var(--primary-color);
  color: white;
  border: none;
}

.channel-header-actions .btn-primary:hover {
  background: var(--primary-hover);
}

.channel-header-actions .btn-secondary {
  background: white;
  color: var(--primary-color);
  border: 1px solid var(--primary-color);
}

.channel-header-actions .btn-secondary:hover {
  background: var(--primary-color);
  color: white;
}

.channel-messages {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  background: var(--bg-color);
}

.channel-messages h3 {
  margin: 0 0 20px 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-color);
}

.channel-messages h3::before {
  content: '';
  width: 3px;
  height: 16px;
  background: var(--primary-color);
  border-radius: 2px;
}

.empty-messages {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: var(--text-secondary);
  text-align: center;
  background: white;
  border-radius: 12px;
  border: 2px dashed var(--border-color);
  transition: all 0.3s ease;
  margin: 20px 0;
}

.empty-messages:hover {
  border-color: var(--primary-color);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.empty-messages i {
  font-size: 40px;
  margin-bottom: 16px;
  color: var(--primary-color);
  opacity: 0.5;
  transition: all 0.3s ease;
}

.empty-messages:hover i {
  opacity: 0.8;
  transform: scale(1.1);
}

.empty-messages p {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
}

.message-list {
  background: var(--right-content-bg);
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  overflow: hidden;
}

.message-item {
  display: flex;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-color);
  transition: all 0.3s ease;
  position: relative;
}

.message-item:hover {
  background: var(--hover-color);
  padding-left: 24px;
}

.message-item:last-child {
  border-bottom: none;
}

.message-avatar {
  width: 36px;
  height: 36px;
  border-radius: 18px;
  object-fit: cover;
  margin-right: 12px;
  border: 2px solid var(--border-color);
  flex-shrink: 0;
  transition: all 0.3s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.message-item:hover .message-avatar {
  border-color: var(--primary-color);
  box-shadow: 0 0 0 4px rgba(0, 123, 255, 0.1);
}

.message-content {
  flex: 1;
  min-width: 0;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.message-sender {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 150px;
}

.message-time {
  font-size: 11px;
  color: var(--text-tertiary);
  white-space: nowrap;
}

.message-text {
  font-size: 14px;
  color: var(--text-primary);
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  padding-right: 10px;
}

.message-input-area {
  display: flex;
  gap: 12px;
  padding: 20px 24px;
  border-top: 1px solid var(--border-color);
  background: var(--hover-color);
  transition: all 0.3s ease;
}

.message-input-area:hover {
  background: var(--bg-color);
}

.message-textarea {
  flex: 1;
  padding: 12px 16px;
  border: 2px solid var(--border-color);
  border-radius: 12px;
  resize: vertical;
  min-height: 70px;
  max-height: 200px;
  font-family: inherit;
  font-size: 14px;
  transition: all 0.3s ease;
  background: white;
  color: var(--text-primary);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.message-textarea:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(0, 123, 255, 0.1);
  transform: translateY(-1px);
}

.send-btn {
  align-self: flex-end;
  padding: 10px 20px;
  border: none;
  border-radius: 20px;
  background: var(--primary-color);
  color: white;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.3s ease;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  white-space: nowrap;
}

.send-btn:hover:not(:disabled) {
  background: var(--primary-hover);
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
}

.send-btn:disabled {
  background: var(--border-color);
  color: var(--text-secondary);
  cursor: not-allowed;
  transform: none;
  box-shadow: none;
}

.channels-content {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--right-content-bg);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
  overflow: hidden;
}

.channels-empty-state {
  text-align: center;
  color: var(--text-secondary);
  padding: 60px 20px;
  transition: all 0.3s ease;
}

.channels-empty-state:hover {
  background: var(--hover-color);
}

.channels-empty-state .empty-icon {
  font-size: 64px;
  margin-bottom: 16px;
  color: var(--primary-color);
  opacity: 0.4;
  transition: all 0.3s ease;
}

.channels-empty-state:hover .empty-icon {
  opacity: 0.6;
  transform: scale(1.05);
}

.channels-empty-state p {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: var(--text-primary);
}

/* 响应式调整 */
@media (max-width: 768px) {
  .channel-detail-info {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }
  
  .channel-header-actions {
    width: 100%;
    justify-content: flex-start;
    flex-wrap: wrap;
  }
  
  .channel-header-actions .btn {
    flex: 1;
    min-width: 120px;
    justify-content: center;
  }
  
  .message-input-area {
    flex-direction: column;
  }
  
  .send-btn {
    align-self: flex-start;
    align-self: stretch;
    justify-content: center;
  }
  
  .message-sender {
    max-width: 100px;
  }
  
  .channel-messages {
    padding: 16px;
  }
  
  .message-item {
    padding: 12px 16px;
  }
  
  .message-item:hover {
    padding-left: 18px;
  }
}

/* 滚动条样式 */
.channel-messages::-webkit-scrollbar,
.message-list::-webkit-scrollbar {
  width: 6px;
}

.channel-messages::-webkit-scrollbar-track,
.message-list::-webkit-scrollbar-track {
  background: var(--bg-color);
  border-radius: 3px;
}

.channel-messages::-webkit-scrollbar-thumb,
.message-list::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 3px;
  transition: all 0.3s ease;
}

.channel-messages::-webkit-scrollbar-thumb:hover,
.message-list::-webkit-scrollbar-thumb:hover {
  background: var(--text-tertiary);
}

/* 动画效果 */
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.channel-detail-content {
  animation: fadeIn 0.3s ease-out;
}

.message-item {
  animation: fadeIn 0.2s ease-out;
}

.message-item:nth-child(1) { animation-delay: 0.05s; }
.message-item:nth-child(2) { animation-delay: 0.1s; }
.message-item:nth-child(3) { animation-delay: 0.15s; }
.message-item:nth-child(4) { animation-delay: 0.2s; }
.message-item:nth-child(5) { animation-delay: 0.25s; }


[data-theme="elegant-dark"] .clear-cache-btn:hover,
[data-theme="elegant-dark"] .security-btn:hover {
  background: var(--hover-color) !important;
}

[data-theme="elegant-dark"] .about-info {
  color: var(--text-secondary) !important;
}

[data-theme="elegant-dark"] .settings-footer {
  background: var(--sidebar-bg) !important;
  border-top: 1px solid var(--border-color) !important;
  box-shadow: 0 -2px 4px rgba(0, 0, 0, 0.3) !important;
}



/* 统计报表应用样式 */


/* 动画效果优化 */
/* 页面过渡动画 */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* 滑动动画 */
.slide-enter-active,
.slide-leave-active {
  transition: all 0.3s ease;
}

.slide-enter-from {
  transform: translateX(-20px);
  opacity: 0;
}

.slide-leave-to {
  transform: translateX(20px);
  opacity: 0;
}

/* 缩放动画 */
.scale-enter-active,
.scale-leave-active {
  transition: all 0.3s ease;
}

.scale-enter-from,
.scale-leave-to {
  transform: scale(0.95);
  opacity: 0;
}

/* 悬停效果优化 */
.option-item {
  transition: all 0.3s ease;
}

.option-item:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.conversation-item {
  transition: all 0.3s ease;
}

.conversation-item:hover {
  background: var(--hover-color);
  transform: translateX(4px);
}

.app-item {
  transition: all 0.3s ease;
}

.app-item:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

/* 按钮动画效果 */
button {
  transition: all 0.3s ease;
}

button:hover {
  transform: translateY(-1px);
}

button:active {
  transform: translateY(0);
}

/* 加载动画 */
.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid var(--border-color);
  border-top: 3px solid var(--primary-color);
  border-radius: 50%;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

/* 脉冲动画 */
@keyframes pulse {
  0% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.05);
    opacity: 0.8;
  }
  100% {
    transform: scale(1);
    opacity: 1;
  }
}

.pulse {
  animation: pulse 2s infinite;
}

/* 渐变背景效果 */
.gradient-bg {
  background: linear-gradient(135deg, var(--primary-color), var(--accent-color));
}

/* 卡片阴影效果优化 */
.card {
  box-shadow: var(--shadow-sm);
  transition: all 0.3s ease;
}

.card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

/* 滚动条样式优化 */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: var(--list-bg);
  border-radius: 3px;
}

::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.1);
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.2);
}

/* 文本选择效果 */
::selection {
  background: var(--primary-color);
  color: white;
}

/* 焦点效果 */
input:focus,
textarea:focus,
select:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  transition: all 0.3s ease;
}

/* 响应式设计优化 */
/* 大屏幕 */
@media (min-width: 1200px) {
  .main-content {
    flex: 1;
    display: flex;
  }
  
  .sidebar {
    width: 320px;
  }
  
  .right-content {
    flex: 1;
  }
}

/* 中等屏幕 */
@media (min-width: 768px) and (max-width: 1199px) {
  .main-content {
    flex: 1;
    display: flex;
  }
  
  .sidebar {
    width: 280px;
  }
  
  .right-content {
    flex: 1;
  }
  

}

/* 小屏幕 */
@media (max-width: 767px) {
  .main-content-area {
    flex-direction: column;
  }
  
  .side-options {
    width: 100%;
    height: 60px;
    flex-direction: row;
    padding: 0 20px;
    gap: 16px;
  }
  
  .option-spacer {
    display: none;
  }
  
  .settings-option {
    transform: none;
  }
  
  .settings-option:hover {
    transform: none;
  }
  
  .main-content {
    flex: 1;
    flex-direction: column;
  }
  
  .sidebar {
    width: 100%;
    max-height: 300px;
    border-right: none;
    border-bottom: 1px solid var(--border-color);
  }
  
  .right-content {
    flex: 1;
  }
  
  .conversation-item {
    padding: 12px 16px;
  }
  

  
  .tasks-board {
    flex-direction: column;
    height: auto;
  }
  
  .task-column {
    min-width: 100%;
    margin-bottom: 16px;
  }
  
  .calendar-grid {
    font-size: 14px;
  }
  
  .calendar-day {
    min-height: 80px;
  }
  
  .day-events {
    font-size: 12px;
  }
  

  
  .modal-content {
    width: 95%;
    margin: 20px;
  }
  
  .window-controls {
    padding: 0 16px;
  }
  
  .window-controls-left {
    width: 60px;
  }
  
  .window-title {
    font-size: 12px;
  }
  
  .window-control-btn {
    width: 14px;
    height: 14px;
    font-size: 10px;
  }
  
  .sidebar-header {
    padding: 12px 16px;
  }
  
  .user-avatar {
    width: 36px;
    height: 36px;
  }
  
  .user-name {
    font-size: 14px;
  }
  
  .search-box {
    padding: 12px 16px;
  }
  
  .search-input {
    padding: 8px 32px 8px 12px;
  }
  
  .right-content-header {
    padding: 16px 20px;
  }
  
  .right-content-header h2 {
    font-size: 16px;
  }
  

  

  
}

/* 用户创建的应用 */
.user-app-content {
  height: calc(100% - 60px);
  padding: 20px;
  overflow: hidden;
}

.user-app-iframe-container {
  height: 100%;
  width: 100%;
  overflow: hidden;
}

.user-app-iframe {
  height: 100%;
  width: 100%;
  border: none;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.empty-user-app {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  color: #666;
}

.empty-user-app .empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
  color: #ccc;
}

.empty-user-app p {
  margin: 8px 0;
}

.empty-user-app .empty-hint {
  font-size: 14px;
  color: #999;
}

/* 应用头部返回按钮 */
.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.back-button {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: var(--hover-color);
  color: var(--primary-color);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
  font-size: 14px;
}

.back-button:hover {
  background: var(--primary-light);
  transform: scale(1.05);
  box-shadow: var(--shadow-sm);
}

.right-content-header h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  transition: color 0.3s ease;
}
</style>