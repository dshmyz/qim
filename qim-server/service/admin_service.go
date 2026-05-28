package service

import (
	"context"
	"time"

	"qim-server/model"

	"gorm.io/gorm"
)

type AdminService struct {
	db *gorm.DB
}

func NewAdminService(db *gorm.DB) *AdminService {
	return &AdminService{db: db}
}

type RoleQuery struct {
	Page     int
	PageSize int
	Keyword  string
}

type RoleInfo struct {
	ID          uint     `json:"id"`
	Name        string   `json:"name"`
	Code        string   `json:"code"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	UserCount   int64    `json:"userCount"`
	CreatedAt   string   `json:"createdAt"`
}

func (s *AdminService) GetRoles(query RoleQuery) ([]RoleInfo, int64, error) {
	ctx := context.Background()

	dbQuery := s.db.WithContext(ctx).Model(&model.UserRole{})
	if query.Keyword != "" {
		dbQuery = dbQuery.Where("role LIKE ?", "%"+query.Keyword+"%")
	}

	var userRoles []model.UserRole
	dbQuery.Find(&userRoles)

	roleMap := make(map[string]int64)
	for _, ur := range userRoles {
		roleMap[ur.Role]++
	}

	roleDefinitions := []struct {
		Code        string
		Name        string
		Description string
	}{
		{"system_admin", "系统管理员", "拥有系统全部权限"},
		{"system_publisher", "系统发布者", "可以发布系统消息"},
		{"user_manager", "用户管理员", "可以管理用户"},
		{"group_manager", "群组管理员", "可以管理群组"},
		{"channel_manager", "频道管理员", "可以管理频道"},
	}

	roles := make([]RoleInfo, 0, len(roleDefinitions))
	for _, rd := range roleDefinitions {
		if query.Keyword != "" && (rd.Code != query.Keyword && rd.Name != query.Keyword) {
			continue
		}
		role := RoleInfo{
			ID:          uint(len(roles) + 1),
			Name:        rd.Name,
			Code:        rd.Code,
			Description: rd.Description,
			Permissions: getPermissionsByRole(rd.Code),
			UserCount:   roleMap[rd.Code],
			CreatedAt:   "",
		}
		roles = append(roles, role)
	}

	return roles, int64(len(roles)), nil
}

func getPermissionsByRole(roleCode string) []string {
	switch roleCode {
	case "system_admin":
		return []string{"user:read", "user:create", "user:update", "user:delete",
			"group:read", "group:create", "group:update", "group:delete",
			"role:read", "role:create", "role:update", "role:delete",
			"system:config", "system:log"}
	case "system_publisher":
		return []string{"message:write", "system:log"}
	case "user_manager":
		return []string{"user:read", "user:create", "user:update"}
	case "group_manager":
		return []string{"group:read", "group:create", "group:update", "group:delete"}
	case "channel_manager":
		return []string{"channel:read", "channel:create", "channel:update", "channel:delete"}
	default:
		return []string{}
	}
}

func (s *AdminService) CreateRole(userID uint, role string) (*model.UserRole, error) {
	ctx := context.Background()

	var existing model.UserRole
	err := s.db.WithContext(ctx).Where("user_id = ? AND role = ?", userID, role).First(&existing).Error
	if err == nil {
		return nil, gorm.ErrDuplicatedKey
	}

	userRole := model.UserRole{
		UserID: userID,
		Role:   role,
	}
	if err := s.db.WithContext(ctx).Create(&userRole).Error; err != nil {
		return nil, err
	}
	return &userRole, nil
}

func (s *AdminService) GetRole(id uint) (*model.UserRole, error) {
	ctx := context.Background()
	var userRole model.UserRole
	err := s.db.WithContext(ctx).First(&userRole, id).Error
	if err != nil {
		return nil, err
	}
	return &userRole, nil
}

func (s *AdminService) UpdateRole(userRole *model.UserRole) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Save(userRole).Error
}

func (s *AdminService) DeleteRole(id uint) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Delete(&model.UserRole{}, id).Error
}

type RoleUserInfo struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

func (s *AdminService) GetRoleUsers(role string, page, pageSize int) ([]RoleUserInfo, int64, error) {
	ctx := context.Background()

	var total int64
	s.db.WithContext(ctx).Model(&model.UserRole{}).Where("role = ?", role).Count(&total)

	var userRoles []model.UserRole
	offset := (page - 1) * pageSize
	s.db.WithContext(ctx).Where("role = ?", role).Offset(offset).Limit(pageSize).Find(&userRoles)

	users := make([]RoleUserInfo, 0, len(userRoles))
	for _, ur := range userRoles {
		var user model.User
		if err := s.db.WithContext(ctx).First(&user, ur.UserID).Error; err == nil {
			users = append(users, RoleUserInfo{
				ID:       user.ID,
				Username: user.Username,
				Nickname: user.Nickname,
				Avatar:   user.Avatar,
			})
		}
	}

	return users, total, nil
}

func (s *AdminService) DeleteGroup(convID uint) error {
	ctx := context.Background()

	var conversation model.Conversation
	if err := s.db.WithContext(ctx).First(&conversation, convID).Error; err != nil {
		return err
	}

	conversation.IsDeleted = true
	s.db.WithContext(ctx).Save(&conversation)
	s.db.WithContext(ctx).Where("conversation_id = ?", convID).Delete(&model.ConversationMember{})

	return nil
}

type AdminUserInfo struct {
	model.User
	Roles []string `json:"roles"`
}

func (s *AdminService) GetUsers(page, pageSize int, keyword string) ([]AdminUserInfo, int64, error) {
	ctx := context.Background()

	query := s.db.WithContext(ctx).Model(&model.User{})
	if keyword != "" {
		query = query.Where("username LIKE ? OR nickname LIKE ? OR email LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var users []model.User
	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&users)

	responseUsers := make([]AdminUserInfo, 0, len(users))
	for _, user := range users {
		var roles []model.UserRole
		s.db.WithContext(ctx).Where("user_id = ?", user.ID).Find(&roles)
		roleNames := make([]string, 0, len(roles))
		for _, role := range roles {
			roleNames = append(roleNames, role.Role)
		}
		responseUsers = append(responseUsers, AdminUserInfo{
			User:  user,
			Roles: roleNames,
		})
	}

	return responseUsers, total, nil
}

type AdminChannelInfo struct {
	ID                uint   `json:"id"`
	Name              string `json:"name"`
	Avatar            string `json:"avatar"`
	Description       string `json:"description"`
	Status            string `json:"status"`
	PublishPermission string `json:"publish_permission"`
	CreatorName       string `json:"creatorName"`
	MemberCount       int64  `json:"memberCount"`
	CreatedAt         string `json:"createdAt"`
}

func (s *AdminService) GetChannels(page, pageSize int, keyword string) ([]AdminChannelInfo, int64, error) {
	ctx := context.Background()

	query := s.db.WithContext(ctx).Model(&model.Channel{})
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var channels []model.Channel
	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Preload("Creator").Order("id DESC").Find(&channels)

	channelInfos := make([]AdminChannelInfo, 0, len(channels))
	for _, ch := range channels {
		var memberCount int64
		s.db.WithContext(ctx).Model(&model.ChannelSubscriber{}).Where("channel_id = ?", ch.ID).Count(&memberCount)

		creatorName := ""
		if ch.Creator.ID > 0 {
			creatorName = ch.Creator.Nickname
			if creatorName == "" {
				creatorName = ch.Creator.Username
			}
		}

		channelInfos = append(channelInfos, AdminChannelInfo{
			ID:                ch.ID,
			Name:              ch.Name,
			Avatar:            ch.Avatar,
			Description:       ch.Description,
			Status:            ch.Status,
			PublishPermission: ch.PublishPermission,
			CreatorName:       creatorName,
			MemberCount:       memberCount,
			CreatedAt:         ch.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return channelInfos, total, nil
}

func (s *AdminService) GetChannel(id uint) (*model.Channel, error) {
	ctx := context.Background()
	var channel model.Channel
	err := s.db.WithContext(ctx).Preload("Creator").First(&channel, id).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

func (s *AdminService) UpdateChannel(id uint, updates map[string]interface{}) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Model(&model.Channel{}).Where("id = ?", id).Updates(updates).Error
}

func (s *AdminService) DeleteChannel(id uint) error {
	ctx := context.Background()

	var channel model.Channel
	if err := s.db.WithContext(ctx).First(&channel, id).Error; err != nil {
		return err
	}

	s.db.WithContext(ctx).Delete(&channel)
	s.db.WithContext(ctx).Where("channel_id = ?", id).Delete(&model.ChannelSubscriber{})
	s.db.WithContext(ctx).Where("channel_id = ?", id).Delete(&model.ChannelMessage{})

	return nil
}

type AdminGroupInfo struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	MemberCount int64  `json:"memberCount"`
	CreatedAt   string `json:"createdAt"`
}

func (s *AdminService) GetGroups(page, pageSize int, keyword string) ([]AdminGroupInfo, int64, error) {
	ctx := context.Background()

	query := s.db.WithContext(ctx).Model(&model.Conversation{}).Where("type IN ?", []string{"group", "discussion"})
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var conversations []model.Conversation
	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&conversations)

	groups := make([]AdminGroupInfo, 0, len(conversations))
	for _, conv := range conversations {
		var memberCount int64
		s.db.WithContext(ctx).Model(&model.ConversationMember{}).Where("conversation_id = ?", conv.ID).Count(&memberCount)

		var group model.Group
		s.db.WithContext(ctx).Where("conversation_id = ?", conv.ID).First(&group)

		status := "active"
		if conv.IsDeleted {
			status = "inactive"
		}

		groups = append(groups, AdminGroupInfo{
			ID:          conv.ID,
			Name:        group.Name,
			Avatar:      group.Avatar,
			Description: group.Announcement,
			Type:        conv.Type,
			Status:      status,
			MemberCount: memberCount,
			CreatedAt:   conv.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return groups, total, nil
}

type GrowthRate struct {
	Users    float64 `json:"users"`
	Groups   float64 `json:"groups"`
	Messages float64 `json:"messages"`
}

type AdminStatistics struct {
	TotalUsers    int64      `json:"totalUsers"`
	OnlineUsers   int64      `json:"onlineUsers"`
	TotalGroups   int64      `json:"totalGroups"`
	TotalChannels int64      `json:"totalChannels"`
	TotalMessages int64      `json:"totalMessages"`
	ActiveUsers   int64      `json:"activeUsers"`
	MessagesToday int64      `json:"messagesToday"`
	GrowthRate    GrowthRate `json:"growthRate"`
}

func (s *AdminService) GetStatistics() (*AdminStatistics, error) {
	ctx := context.Background()

	var stats AdminStatistics

	s.db.WithContext(ctx).Model(&model.User{}).Count(&stats.TotalUsers)
	s.db.WithContext(ctx).Model(&model.User{}).Where("status = ?", "online").Count(&stats.OnlineUsers)
	s.db.WithContext(ctx).Model(&model.Conversation{}).Where("type = ? AND is_deleted = ?", "group", false).Count(&stats.TotalGroups)
	s.db.WithContext(ctx).Model(&model.Message{}).Count(&stats.TotalMessages)
	s.db.WithContext(ctx).Model(&model.Channel{}).Where("status = ?", "active").Count(&stats.TotalChannels)
	s.db.WithContext(ctx).Model(&model.Message{}).Where("created_at >= CURRENT_DATE").Count(&stats.MessagesToday)
	s.db.WithContext(ctx).Model(&model.Message{}).Distinct("sender_id").Where("created_at >= CURRENT_DATE").Count(&stats.ActiveUsers)

	// 计算增长率：对比最近 7 天与前 7 天
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	twoWeeksAgo := now.AddDate(0, 0, -14)

	var currentWeekUsers, prevWeekUsers int64
	s.db.WithContext(ctx).Model(&model.User{}).Where("created_at >= ?", weekAgo).Count(&currentWeekUsers)
	s.db.WithContext(ctx).Model(&model.User{}).Where("created_at >= ? AND created_at < ?", twoWeeksAgo, weekAgo).Count(&prevWeekUsers)
	stats.GrowthRate.Users = calcGrowthRate(currentWeekUsers, prevWeekUsers)

	var currentWeekGroups, prevWeekGroups int64
	s.db.WithContext(ctx).Model(&model.Conversation{}).Where("type = ? AND created_at >= ?", "group", weekAgo).Count(&currentWeekGroups)
	s.db.WithContext(ctx).Model(&model.Conversation{}).Where("type = ? AND created_at >= ? AND created_at < ?", "group", twoWeeksAgo, weekAgo).Count(&prevWeekGroups)
	stats.GrowthRate.Groups = calcGrowthRate(currentWeekGroups, prevWeekGroups)

	var currentWeekMsgs, prevWeekMsgs int64
	s.db.WithContext(ctx).Model(&model.Message{}).Where("created_at >= ?", weekAgo).Count(&currentWeekMsgs)
	s.db.WithContext(ctx).Model(&model.Message{}).Where("created_at >= ? AND created_at < ?", twoWeeksAgo, weekAgo).Count(&prevWeekMsgs)
	stats.GrowthRate.Messages = calcGrowthRate(currentWeekMsgs, prevWeekMsgs)

	return &stats, nil
}

func calcGrowthRate(current, previous int64) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100
	}
	return float64(current-previous) / float64(previous) * 100
}

type RegistrationInfo struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Avatar    string `json:"avatar"`
	CreatedAt string `json:"createdAt"`
}

func (s *AdminService) GetRecentRegistrations(page, pageSize int) ([]RegistrationInfo, int64, error) {
	ctx := context.Background()

	var total int64
	s.db.WithContext(ctx).Model(&model.User{}).Count(&total)

	var users []model.User
	offset := (page - 1) * pageSize
	s.db.WithContext(ctx).Order("id DESC").Offset(offset).Limit(pageSize).Find(&users)

	registrations := make([]RegistrationInfo, 0, len(users))
	for _, user := range users {
		registrations = append(registrations, RegistrationInfo{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Avatar:    user.Avatar,
			CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return registrations, total, nil
}
