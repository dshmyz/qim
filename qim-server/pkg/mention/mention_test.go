package mention

import "testing"

func TestParseOnlyRecognizesStructuredMentionTokens(t *testing.T) {
	content := `go get gitee.com/xxx/xxxx/xxx/@v1.0.20
@{mention:42|%E5%BC%A0%E4%B8%89} 请看 @{mention:all}`

	mentions := Parse(content)

	if len(mentions) != 2 || mentions[0].UserID != 42 || !mentions[1].All {
		t.Fatalf("Parse() = %#v, want direct mention 42 and @all", mentions)
	}
	if !IsMentioned(mentions, 42) || IsMentioned(mentions, 7) {
		t.Fatal("only direct structured targets should match without expanding @all")
	}
	userIDs := ExtractUserIDs(mentions, []uint{7, 42}, 0)
	if len(userIDs) != 2 || userIDs[0] != 42 || userIDs[1] != 7 {
		t.Fatalf("ExtractUserIDs() = %v, want [42 7]", userIDs)
	}
}

func TestParseAcceptsTokensWithOptionalNames(t *testing.T) {
	mentions := Parse("@张三 @{mention:0|bad} @{mention:42} @{mention:all|bad}")
	if len(mentions) != 2 || mentions[0].UserID != 42 || !mentions[1].All {
		t.Fatalf("Parse() = %#v, want user 42 and @all", mentions)
	}
}
