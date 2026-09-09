package service

import "testing"

func TestUseFulltextForMessageKeyword(t *testing.T) {
	tests := []struct {
		name    string
		keyword string
		want    bool
	}{
		{name: "english keyword", keyword: "golang", want: true},
		{name: "english phrase", keyword: "hello world", want: true},
		{name: "chinese keyword", keyword: "项目", want: false},
		{name: "single chinese character", keyword: "你", want: false},
		{name: "mixed keyword", keyword: "hello项目", want: false},
		{name: "empty keyword", keyword: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := useFulltextForMessageKeyword(tt.keyword); got != tt.want {
				t.Fatalf("useFulltextForMessageKeyword(%q) = %v, want %v", tt.keyword, got, tt.want)
			}
		})
	}
}
