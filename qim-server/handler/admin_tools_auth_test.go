package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/database"
	"github.com/dshmyz/qim/qim-server/model"
)

func TestAdminToolsRejectAnonymousCaller(t *testing.T) {
	tests := []struct {
		name string
		tool ai.Tool
	}{
		{name: "user management", tool: &UserManagementTool{}},
		{name: "group management", tool: &GroupManagementTool{}},
		{name: "system notification", tool: &SystemNotificationTool{}},
		{name: "create task", tool: &CreateGroupTaskTool{}},
		{name: "search messages", tool: &SearchMessagesTool{}},
		{name: "group summary", tool: &GroupSummaryTool{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tt.tool.Execute(map[string]interface{}{}, &ai.CallerContext{}); err == nil {
				t.Fatalf("%s should reject anonymous caller", tt.tool.Name())
			}
		})
	}
}

func TestRequireGroupMemberRejectsAnonymousCaller(t *testing.T) {
	err := requireGroupMember(1, &ai.CallerContext{})
	if err == nil {
		t.Fatal("expected anonymous caller to be rejected")
	}
}

func TestUserManagementToolUpdatesAccountStatus(t *testing.T) {
	db := setupHandlerTestDB(t)
	database.DB = db

	admin := &model.User{Username: "system-admin", Status: "online"}
	target := &model.User{Username: "managed-user", Status: "online"}
	if err := db.Create(admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(target).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.UserRole{UserID: admin.ID, Role: "system_admin"}).Error; err != nil {
		t.Fatal(err)
	}

	result, err := (&UserManagementTool{}).Execute(map[string]interface{}{
		"action":          "disable",
		"user_identifier": target.Username,
	}, &ai.CallerContext{UserID: admin.ID})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected successful disable result")
	}

	var updated model.User
	if err := db.First(&updated, target.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.AccountStatus != "disabled" {
		t.Fatalf("expected account status disabled, got %q", updated.AccountStatus)
	}
	if updated.Status != "online" {
		t.Fatalf("expected connection status unchanged, got %q", updated.Status)
	}
}
