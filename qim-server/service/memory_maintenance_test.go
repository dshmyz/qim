package service

import "testing"

func TestParseMemoryBucket(t *testing.T) {
	cases := []struct {
		in        string
		userID    string
		namespace string
		ok        bool
	}{
		{"memory:user:123:group_assistant", "123", "group_assistant", true},
		{"memory:user:42:avatar", "42", "avatar", true},
		{"memory:user::avatar", "", "", false},
		{"memory:user:123", "", "", false},
		{"other:user:123:avatar", "", "", false},
	}
	for _, c := range cases {
		user, ns, ok := parseMemoryBucket(c.in)
		if ok != c.ok || user != c.userID || ns != c.namespace {
			t.Errorf("parseMemoryBucket(%q)=(%q,%q,%v) want (%q,%q,%v)",
				c.in, user, ns, ok, c.userID, c.namespace, c.ok)
		}
	}
}
