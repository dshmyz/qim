package handler

import (
	"encoding/json"
	"testing"
)

func TestBuildRemindResultIncludesConfiguredSystemName(t *testing.T) {
	payload, err := buildRemindResult(42, true, "", "企业微信")
	if err != nil {
		t.Fatalf("buildRemindResult returned error: %v", err)
	}

	var result struct {
		Type string `json:"type"`
		Data struct {
			MessageID  uint   `json:"message_id"`
			Success    bool   `json:"success"`
			SystemName string `json:"system_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("unmarshal reminder result: %v", err)
	}

	if result.Type != "remind_result" {
		t.Errorf("expected type remind_result, got %q", result.Type)
	}
	if result.Data.MessageID != 42 {
		t.Errorf("expected message ID 42, got %d", result.Data.MessageID)
	}
	if !result.Data.Success {
		t.Error("expected success result")
	}
	if result.Data.SystemName != "企业微信" {
		t.Errorf("expected system name 企业微信, got %q", result.Data.SystemName)
	}
}
