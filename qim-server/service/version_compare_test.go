package service

import (
	"errors"
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
)

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"2.1.0", true},
		{"1.0.0", true},
		{"10.20.30", true},
		{"2.1.0-beta", false},  // 与前端一致，不接受预发布后缀
		{"2.1.0+build", false}, // 与前端一致，不接受构建元数据
		{"v2.1.0", false},      // 不接受 v 前缀
		{"", false},
		{"abc", false},
		{"2.1", false}, // 与前端一致，要求三段数字
	}
	for _, tt := range tests {
		got := IsValidVersion(tt.version)
		if got != tt.want {
			t.Errorf("IsValidVersion(%q) = %v, want %v", tt.version, got, tt.want)
		}
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"2.1.0", "2.0.0", 1},
		{"2.0.0", "2.1.0", -1},
		{"2.1.0", "2.1.0", 0},
		{"3.0.0", "2.99.99", 1},
		{"1.0.0", "1.0.1", -1},
		{"10.0.0", "9.0.0", 1},
	}
	for _, tt := range tests {
		got := CompareVersions(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestLatestVersion(t *testing.T) {
	versions := []model.ClientVersion{
		{Version: "2.0.0"},
		{Version: "2.1.0"},
		{Version: "1.9.0"},
		{Version: "2.0.1"},
	}
	latest, err := LatestVersion(versions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest.Version != "2.1.0" {
		t.Errorf("expected 2.1.0, got %s", latest.Version)
	}
}

func TestLatestVersion_SupplementaryOldVersion(t *testing.T) {
	// 补录旧版本场景：创建时间晚但版本号低，不应被误判为最新
	versions := []model.ClientVersion{
		{Version: "2.1.0"}, // 先创建（旧记录）
		{Version: "1.0.0"}, // 后创建（补录旧版本）
	}
	latest, err := LatestVersion(versions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest.Version != "2.1.0" {
		t.Errorf("expected 2.1.0 (semver), got %s (might be created_at sort)", latest.Version)
	}
}

func TestLatestVersion_Empty(t *testing.T) {
	_, err := LatestVersion([]model.ClientVersion{})
	if err == nil {
		t.Error("expected error for empty list")
	}
}

func TestLatestVersion_Single(t *testing.T) {
	versions := []model.ClientVersion{
		{Version: "3.0.0"},
	}
	latest, err := LatestVersion(versions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if latest.Version != "3.0.0" {
		t.Errorf("expected 3.0.0, got %s", latest.Version)
	}
}

func TestNormalizePlatform(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"win", "windows"},
		{"win7", "windows"},
		{"win10", "windows"},
		{"windows", "windows"},
		{"WIN", "windows"},
		{"Windows", "windows"},
		{"mac", "macos"},
		{"macos", "macos"},
		{"Mac", "macos"},
		{"linux", "linux"},
		{"LINUX", "linux"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := NormalizePlatform(tt.input)
		if got != tt.want {
			t.Errorf("NormalizePlatform(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestValidateVersionFormat(t *testing.T) {
	if err := ValidateVersionFormat(""); err == nil {
		t.Error("expected error for empty version")
	}
	if err := ValidateVersionFormat("abc"); err == nil {
		t.Error("expected error for invalid version")
	}
	if err := ValidateVersionFormat("2.1.0"); err != nil {
		t.Errorf("expected no error for valid version, got %v", err)
	}
}

func TestNormalizeRolloutPercentage(t *testing.T) {
	tests := []struct {
		name  string
		input *int
		want  int
	}{
		{name: "missing defaults to full rollout", input: nil, want: 100},
		{name: "zero means disabled", input: intPtr(0), want: 0},
		{name: "partial rollout is preserved", input: intPtr(30), want: 30},
		{name: "full rollout is preserved", input: intPtr(100), want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeRolloutPercentage(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil || *got != tt.want {
				t.Fatalf("expected %d, got %v", tt.want, got)
			}
		})
	}
}

func TestNormalizeRolloutPercentageRejectsOutOfRange(t *testing.T) {
	for _, value := range []int{-1, 101} {
		_, err := NormalizeRolloutPercentage(intPtr(value))
		if !errors.Is(err, ErrInvalidRolloutPercentage) {
			t.Fatalf("expected ErrInvalidRolloutPercentage for %d, got %v", value, err)
		}
	}
}

func TestApplyRolloutPercentageUpdate(t *testing.T) {
	version := model.ClientVersion{RolloutPercentage: intPtr(25)}

	if err := ApplyRolloutPercentageUpdate(&version, nil); err != nil {
		t.Fatalf("unexpected error for missing rollout percentage: %v", err)
	}
	if version.GetRolloutPercentage() != 25 {
		t.Fatalf("expected missing rollout percentage to leave value unchanged, got %d", version.GetRolloutPercentage())
	}

	if err := ApplyRolloutPercentageUpdate(&version, intPtr(0)); err != nil {
		t.Fatalf("unexpected error for zero rollout percentage: %v", err)
	}
	if version.GetRolloutPercentage() != 0 {
		t.Fatalf("expected zero rollout percentage to be preserved, got %d", version.GetRolloutPercentage())
	}

	if err := ApplyRolloutPercentageUpdate(&version, intPtr(101)); !errors.Is(err, ErrInvalidRolloutPercentage) {
		t.Fatalf("expected ErrInvalidRolloutPercentage, got %v", err)
	}
}

func intPtr(v int) *int {
	return &v
}
