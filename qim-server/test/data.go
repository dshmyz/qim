package test

import (
	"fmt"
	"log"
	"qim-server/database"
	"qim-server/model"
	"time"

	"gorm.io/gorm"
)

// AddTestData 添加测试数据
func AddTestData() {
	db := database.GetDB()

	// 检查是否已有测试数据
	var count int64
	db.Model(&model.Channel{}).Count(&count)
	if count > 0 {
		fmt.Println("测试数据已存在，跳过添加")
		return
	}

	// 添加测试频道
	channels := []model.Channel{
		{
			Name:        "公司公告",
			Description: "公司内部公告和通知",
			Avatar:      "https://api.dicebear.com/7.x/avataaars/svg?seed=company",
			CreatorID:   1, // 假设用户ID为1的是系统管理员
			Status:      "active",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "技术分享",
			Description: "技术相关的分享和讨论",
			Avatar:      "https://api.dicebear.com/7.x/avataaars/svg?seed=tech",
			CreatorID:   1,
			Status:      "active",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			Name:        "产品动态",
			Description: "产品相关的动态和更新",
			Avatar:      "https://api.dicebear.com/7.x/avataaars/svg?seed=product",
			CreatorID:   1,
			Status:      "active",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	for _, channel := range channels {
		db.Create(&channel)
	}

	// 获取创建的频道
	var createdChannels []model.Channel
	db.Find(&createdChannels)

	// 添加测试频道订阅
	subscribers := []struct {
		channelID uint
		userID    uint
	}{
		{createdChannels[0].ID, 1},
		{createdChannels[0].ID, 2},
		{createdChannels[0].ID, 3},
		{createdChannels[1].ID, 1},
		{createdChannels[1].ID, 2},
		{createdChannels[2].ID, 1},
		{createdChannels[2].ID, 3},
	}

	for _, sub := range subscribers {
		channelSubscriber := model.ChannelSubscriber{
			ChannelID: sub.channelID,
			UserID:    sub.userID,
			JoinedAt:  time.Now(),
		}
		db.Create(&channelSubscriber)
	}

	// 添加测试频道消息
	messages := []struct {
		channelID uint
		senderID  uint
		content   string
	}{
		{createdChannels[0].ID, 1, "欢迎大家加入公司公告频道，这里会发布公司的重要通知。"},
		{createdChannels[0].ID, 1, "下周一开始，公司将实行新的考勤制度，请大家注意查看邮件。"},
		{createdChannels[1].ID, 1, "欢迎来到技术分享频道，这里可以分享技术文章和问题。"},
		{createdChannels[1].ID, 2, "推荐一个不错的前端框架，大家可以了解一下。"},
		{createdChannels[2].ID, 1, "产品动态频道正式开通，欢迎大家关注产品的最新进展。"},
	}

	for _, msg := range messages {
		channelMessage := model.ChannelMessage{
			ChannelID: msg.channelID,
			SenderID:  msg.senderID,
			Content:   msg.content,
			Type:      "text",
			CreatedAt: time.Now(),
		}
		db.Create(&channelMessage)
	}

	// 添加测试用户角色
	roles := []model.UserRole{
		{UserID: 1, Role: "system_admin", CreatedAt: time.Now()},
		{UserID: 2, Role: "system_publisher", CreatedAt: time.Now()},
	}

	for _, role := range roles {
		db.Create(&role)
	}

	fmt.Println("测试数据添加成功")
}

// InitTestData 初始化测试数据
func InitTestData(db *gorm.DB) {
	// 检查是否已有用户数据
	var userCount int64
	db.Model(&model.User{}).Count(&userCount)

	// 确保有足够的用户数据
	var users []model.User
	if userCount < 4 {
		log.Println("初始化测试数据...")

		// 清空现有用户数据
		db.Where("1=1").Delete(&model.User{})

		// 创建测试用户
		users = []model.User{
			{
				Username:     "test",
				PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // 123456
				Nickname:     "测试用户",
				Avatar:       "https://api.dicebear.com/7.x/avataaars/svg?seed=test",
				Status:       "online",
			},
			{
				Username:     "user1",
				PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // 123456
				Nickname:     "用户一",
				Avatar:       "https://api.dicebear.com/7.x/avataaars/svg?seed=user1",
				Status:       "online",
			},
			{
				Username:     "user2",
				PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // 123456
				Nickname:     "用户二",
				Avatar:       "https://api.dicebear.com/7.x/avataaars/svg?seed=user2",
				Status:       "online",
			},
			{
				Username:     "user3",
				PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // 123456
				Nickname:     "用户三",
				Avatar:       "https://api.dicebear.com/7.x/avataaars/svg?seed=user3",
				Status:       "online",
			},
		}

		for _, user := range users {
			db.Create(&user)
		}
	} else {
		// 从数据库中查询用户数据
		db.Find(&users)
	}

	// 检查是否已有部门数据
	var deptCount int64
	db.Model(&model.Department{}).Count(&deptCount)
	if deptCount == 0 {
		log.Println("初始化部门数据...")

		// 创建部门
		departments := []model.Department{
			{
				Name:      "技术部",
				ParentID:  nil,
				Level:     1,
				Path:      "1",
				SortOrder: 1,
			},
			{
				Name:      "产品部",
				ParentID:  nil,
				Level:     1,
				Path:      "2",
				SortOrder: 2,
			},
			{
				Name:      "前端组",
				ParentID:  new(uint),
				Level:     2,
				Path:      "1-1",
				SortOrder: 1,
			},
			{
				Name:      "后端组",
				ParentID:  new(uint),
				Level:     2,
				Path:      "1-2",
				SortOrder: 2,
			},
		}

		// 先创建顶级部门
		db.Create(&departments[0])
		db.Create(&departments[1])

		// 设置子部门的父ID
		*departments[2].ParentID = departments[0].ID
		*departments[3].ParentID = departments[0].ID

		// 创建子部门
		db.Create(&departments[2])
		db.Create(&departments[3])

		// 关联用户和部门
		departmentEmployees := []model.DepartmentEmployee{
			{DepartmentID: departments[0].ID, UserID: users[0].ID},
			{DepartmentID: departments[2].ID, UserID: users[1].ID},
			{DepartmentID: departments[3].ID, UserID: users[2].ID},
			{DepartmentID: departments[1].ID, UserID: users[3].ID},
		}

		for _, de := range departmentEmployees {
			db.Create(&de)
		}

		// 清空现有会话数据
		db.Where("1=1").Delete(&model.Message{})
		db.Where("1=1").Delete(&model.ConversationMember{})
		db.Where("1=1").Delete(&model.Conversation{})

		// 创建会话
		conversation1 := model.Conversation{
			Type:      "single",
			Name:      users[1].Nickname,
			Avatar:    users[1].Avatar,
			CreatorID: users[0].ID,
		}
		db.Create(&conversation1)

		conversation2 := model.Conversation{
			Type:      "single",
			Name:      users[2].Nickname,
			Avatar:    users[2].Avatar,
			CreatorID: users[0].ID,
		}
		db.Create(&conversation2)

		conversation3 := model.Conversation{
			Type:      "group",
			Name:      "技术交流群",
			Avatar:    "https://api.dicebear.com/7.x/identicon/svg?seed=group",
			CreatorID: users[0].ID,
		}
		db.Create(&conversation3)

		// 添加会话成员
		conversationMembers := []model.ConversationMember{
			{ConversationID: conversation1.ID, UserID: users[0].ID, Role: "member"},
			{ConversationID: conversation1.ID, UserID: users[1].ID, Role: "member"},
			{ConversationID: conversation2.ID, UserID: users[0].ID, Role: "member"},
			{ConversationID: conversation2.ID, UserID: users[2].ID, Role: "member"},
			{ConversationID: conversation3.ID, UserID: users[0].ID, Role: "owner"},
			{ConversationID: conversation3.ID, UserID: users[1].ID, Role: "member"},
			{ConversationID: conversation3.ID, UserID: users[2].ID, Role: "member"},
			{ConversationID: conversation3.ID, UserID: users[3].ID, Role: "member"},
		}

		for _, cm := range conversationMembers {
			db.Create(&cm)
		}

		// 创建消息
		now := time.Now()
		messages := []model.Message{
			{
				ConversationID: conversation1.ID,
				SenderID:       users[1].ID,
				Type:           "text",
				Content:        "你好，最近怎么样？",
				CreatedAt:      now.Add(-24 * time.Hour),
			},
			{
				ConversationID: conversation1.ID,
				SenderID:       users[0].ID,
				Type:           "text",
				Content:        "挺好的，你呢？",
				CreatedAt:      now.Add(-23 * time.Hour),
			},
			{
				ConversationID: conversation2.ID,
				SenderID:       users[2].ID,
				Type:           "text",
				Content:        "项目进展如何？",
				CreatedAt:      now.Add(-12 * time.Hour),
			},
			{
				ConversationID: conversation3.ID,
				SenderID:       users[3].ID,
				Type:           "text",
				Content:        "大家好，今天我们来讨论一下项目的进展",
				CreatedAt:      now.Add(-6 * time.Hour),
			},
			{
				ConversationID: conversation3.ID,
				SenderID:       users[1].ID,
				Type:           "text",
				Content:        "前端部分已经完成了80%",
				CreatedAt:      now.Add(-5 * time.Hour),
			},
			{
				ConversationID: conversation3.ID,
				SenderID:       users[2].ID,
				Type:           "text",
				Content:        "后端API也基本完成了",
				CreatedAt:      now.Add(-4 * time.Hour),
			},
		}

		for _, msg := range messages {
			db.Create(&msg)
		}

		// 更新会话最后消息
		updateLastMessage := func(convID uint, msgID uint) {
			var conv model.Conversation
			db.First(&conv, convID)
			conv.LastMessageID = &msgID
			db.Save(&conv)
		}

		updateLastMessage(conversation1.ID, messages[1].ID)
		updateLastMessage(conversation2.ID, messages[2].ID)
		updateLastMessage(conversation3.ID, messages[5].ID)

		log.Println("测试数据初始化完成")
	}

	// 初始化机器人数据（无论会话数据是否存在）
	var botCount int64
	db.Model(&model.Bot{}).Count(&botCount)
	if botCount == 0 {
		log.Println("初始化机器人数据...")

		// 创建系统机器人
		systemBot := model.Bot{
			Name:        "系统助手",
			Avatar:      "https://api.dicebear.com/7.x/avataaars/svg?seed=system",
			Description: "提供系统相关的帮助和信息",
			Type:        "system",
			Config:      `{"responses":{"greeting":"你好！我是系统助手，有什么可以帮你的吗？","help":"我可以帮助你了解系统功能，解答常见问题。"}}`,
			IsActive:    true,
		}
		db.Create(&systemBot)

		// 创建AI机器人
		aiBot := model.Bot{
			Name:        "AI助手",
			Avatar:      "https://api.dicebear.com/7.x/avataaars/svg?seed=ai",
			Description: "基于大模型的智能助手，能回答各种问题",
			Type:        "ai",
			Config:      `{"api_key":"your-api-key", "model":"gpt-3.5-turbo", "temperature":0.7}`,
			IsActive:    true,
		}
		db.Create(&aiBot)

		// 为每个用户创建与机器人的会话
		for _, user := range users {
			// 检查用户是否已有机器人会话
			var userBotConvCount int64
			db.Model(&model.BotConversation{}).Where("user_id = ?", user.ID).Count(&userBotConvCount)
			if userBotConvCount == 0 {
				// 系统助手会话
				systemConv := model.Conversation{
					Type:      "bot",
					Name:      systemBot.Name,
					Avatar:    systemBot.Avatar,
					CreatorID: user.ID,
				}
				db.Create(&systemConv)

				// 添加成员
				db.Create(&model.ConversationMember{
					ConversationID: systemConv.ID,
					UserID:         user.ID,
					Role:           "member",
				})

				// 创建机器人会话关联
				db.Create(&model.BotConversation{
					BotID:          systemBot.ID,
					UserID:         user.ID,
					ConversationID: systemConv.ID,
				})

				// AI助手会话
				aiConv := model.Conversation{
					Type:      "bot",
					Name:      aiBot.Name,
					Avatar:    aiBot.Avatar,
					CreatorID: user.ID,
				}
				db.Create(&aiConv)

				// 添加成员
				db.Create(&model.ConversationMember{
					ConversationID: aiConv.ID,
					UserID:         user.ID,
					Role:           "member",
				})

				// 创建机器人会话关联
				db.Create(&model.BotConversation{
					BotID:          aiBot.ID,
					UserID:         user.ID,
					ConversationID: aiConv.ID,
				})

				// 发送欢迎消息
				welcomeMsg := model.Message{
					ConversationID: aiConv.ID,
					SenderID:       0, // 0表示系统/机器人
					Type:           "text",
					Content:        "你好！我是AI助手，有什么可以帮你的吗？",
				}
				db.Create(&welcomeMsg)

				// 更新会话最后消息
				var conv model.Conversation
				db.First(&conv, aiConv.ID)
				conv.LastMessageID = &welcomeMsg.ID
				db.Save(&conv)
			}
		}
	}

	// 初始化小程序数据
	var miniAppCount int64
	db.Model(&model.MiniApp{}).Count(&miniAppCount)
	if miniAppCount == 0 {
		log.Println("初始化小程序数据...")

		// 创建计算器小程序
		calculatorApp := model.MiniApp{
			AppID:       "calculator",
			Name:        "计算器",
			Description: "简单易用的计算器",
			Icon:        "https://api.dicebear.com/7.x/avataaars/svg?seed=calculator",
			Path:        "/calculator",
			Status:      "active",
		}
		db.Create(&calculatorApp)

		// 创建记事本小程序
		noteApp := model.MiniApp{
			AppID:       "notepad",
			Name:        "记事本",
			Description: "简单的文本编辑器",
			Icon:        "https://api.dicebear.com/7.x/avataaars/svg?seed=notepad",
			Path:        "/notepad",
			Status:      "active",
		}
		db.Create(&noteApp)

		// 创建待办事项小程序
		todoApp := model.MiniApp{
			AppID:       "todo",
			Name:        "待办事项",
			Description: "任务管理工具",
			Icon:        "https://api.dicebear.com/7.x/avataaars/svg?seed=todo",
			Path:        "/todo",
			Status:      "active",
		}
		db.Create(&todoApp)

		// 创建单位转换器小程序
		converterApp := model.MiniApp{
			AppID:       "unit-converter",
			Name:        "单位转换",
			Description: "多种单位之间的转换",
			Icon:        "https://api.dicebear.com/7.x/avataaars/svg?seed=unit",
			Path:        "/unit-converter",
			Status:      "active",
		}
		db.Create(&converterApp)

		// 创建密码生成器小程序
		passwordApp := model.MiniApp{
			AppID:       "password-generator",
			Name:        "密码生成器",
			Description: "生成强密码",
			Icon:        "https://api.dicebear.com/7.x/avataaars/svg?seed=password",
			Path:        "/password-generator",
			Status:      "active",
		}
		db.Create(&passwordApp)

		log.Println("小程序数据初始化完成")
	}
}
