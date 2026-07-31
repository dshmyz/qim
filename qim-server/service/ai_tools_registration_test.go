package service

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/ai"
)

func TestRegisterUserToolsIncludesCreateUserTask(t *testing.T) {
	registry := ai.NewToolRegistry(nil)

	RegisterUserTools(registry, &TaskService{}, nil, nil, nil)

	if _, ok := registry.GetTool("create_user_task"); !ok {
		t.Fatal("create_user_task should be registered when TaskService is available")
	}
}
