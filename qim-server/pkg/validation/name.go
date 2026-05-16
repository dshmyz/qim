package validation

import (
	"fmt"
	"strings"
	"unicode"
)

func countAliasWeight(name string) int {
	weight := 0
	for _, r := range name {
		if unicode.Is(unicode.Han, r) {
			weight += 2
		} else {
			weight += 1
		}
	}
	return weight
}

const MaxAliasWeight = 8

func ValidateAliasName(name string) error {
	trimmed := strings.TrimSpace(name)
	if len(trimmed) == 0 {
		return fmt.Errorf("名称不能为空")
	}

	weight := countAliasWeight(trimmed)
	if weight > MaxAliasWeight {
		return fmt.Errorf("别名最多允许4个中文字或8个字母")
	}

	return nil
}