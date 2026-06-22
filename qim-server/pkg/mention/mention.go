// Package mention defines the wire format for semantic @ mentions in message
// content. It deliberately does not infer mentions from a naked '@' character.
package mention

import (
	"net/url"
	"regexp"
	"strconv"
)

// Targets are the semantic recipients encoded in a message.
type Targets struct {
	UserIDs []uint
	All     bool
}

var tokenPattern = regexp.MustCompile(`@\{mention:(all|[1-9][0-9]*)(?:\|([^}]*))?\}`)

// Parse extracts only complete, well-formed mention tokens. Text such as a Go
// module URL ending in /@v1.0.20 is ordinary text and produces no target.
func Parse(content string) Targets {
	targets := Targets{UserIDs: []uint{}}
	seen := make(map[uint]struct{})

	for _, match := range tokenPattern.FindAllStringSubmatch(content, -1) {
		target, encodedName := match[1], match[2]
		if target == "all" {
			if encodedName == "" {
				targets.All = true
			}
			continue
		}
		if encodedName == "" {
			continue
		}
		name, err := url.PathUnescape(encodedName)
		if err != nil || name == "" {
			continue
		}
		id, err := strconv.ParseUint(target, 10, 0)
		if err != nil || id == 0 {
			continue
		}
		userID := uint(id)
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		targets.UserIDs = append(targets.UserIDs, userID)
	}

	return targets
}

// IsMentioned checks whether the user is a direct target or the message uses
// the explicit @all token.
func IsMentioned(targets Targets, userID uint) bool {
	if targets.All {
		return true
	}
	for _, id := range targets.UserIDs {
		if id == userID {
			return true
		}
	}
	return false
}
