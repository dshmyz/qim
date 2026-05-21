package provider

import (
	"context"
)

type Credentials struct {
	Username string
	Password string
	Token    string
	Extra    map[string]interface{}
}

type AuthResult struct {
	Success  bool
	UserID   string
	UserInfo map[string]interface{}
	Token    string
	Message  string
}

type AuthProvider interface {
	Name() string
	Authenticate(ctx context.Context, creds *Credentials) (*AuthResult, error)
	IsEnabled() bool
	Priority() int
	GetType() string
}
