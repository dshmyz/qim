package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
)

func TestAdminToolsRejectAnonymousCaller(t *testing.T) {
	tests := []struct {
		name string
		tool ai.Tool
	}{
		{name: "user management", tool: &UserManagementTool{}},
		{name: "group management", tool: &GroupManagementTool{}},
		{name: "system notification", tool: &SystemNotificationTool{}},
		{name: "create task", tool: &CreateTaskTool{}},
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

func TestCallerIsGroupMemberRejectsAnonymousCaller(t *testing.T) {
	ok, err := callerIsGroupMember(1, &ai.CallerContext{})
	if err == nil {
		t.Fatal("expected anonymous caller to be rejected")
	}
	if ok {
		t.Fatal("anonymous caller should not be treated as a group member")
	}
}
