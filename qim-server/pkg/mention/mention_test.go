package mention

import (
	"reflect"
	"testing"
)

func TestParseOnlyRecognizesStructuredMentionTokens(t *testing.T) {
	content := `go get gitee.com/xxx/xxxx/xxx/@v1.0.20
@{mention:42|%E5%BC%A0%E4%B8%89} 请看 @{mention:all}`

	targets := Parse(content)

	if !reflect.DeepEqual(targets.UserIDs, []uint{42}) {
		t.Fatalf("UserIDs = %v, want [42]", targets.UserIDs)
	}
	if !targets.All {
		t.Fatal("All = false, want true")
	}
	if !IsMentioned(targets, 42) || !IsMentioned(targets, 7) {
		t.Fatal("structured targets should notify their direct target and @all recipients")
	}
}

func TestParseIgnoresMalformedTokens(t *testing.T) {
	targets := Parse("@张三 @{mention:0|bad} @{mention:42} @{mention:all|bad}")
	if len(targets.UserIDs) != 0 || targets.All {
		t.Fatalf("Parse() = %#v, want no targets", targets)
	}
}
