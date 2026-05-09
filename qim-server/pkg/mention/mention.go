package mention

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var atUserRegex = regexp.MustCompile(`@(\d+)`)

type AtMention struct {
	UserID   uint
	Username string
	Start    int
	End      int
}

func ExtractMentions(content string) []AtMention {
	var mentions []AtMention
	matches := atUserRegex.FindAllStringSubmatchIndex(content, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			userIDStr := content[match[2]:match[3]]
			userID, err := strconv.ParseUint(userIDStr, 10, 32)
			if err == nil {
				mentions = append(mentions, AtMention{
					UserID: uint(userID),
					Start:  match[0],
					End:    match[1],
				})
			}
		}
	}
	return mentions
}

func IsMentioned(content string, userID uint) bool {
	mentions := ExtractMentions(content)
	for _, m := range mentions {
		if m.UserID == userID {
			return true
		}
	}
	return false
}

func FormatMention(userID uint, username string) string {
	return fmt.Sprintf("@%d(%s)", userID, username)
}

func StripMentionTags(content string) string {
	return atUserRegex.ReplaceAllStringFunc(content, func(match string) string {
		parts := strings.SplitN(match[1:], "(", 2)
		if len(parts) == 2 {
			username := strings.TrimSuffix(parts[1], ")")
			return "@" + username
		}
		return match
	})
}
