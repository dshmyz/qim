package orgsync

import (
	"context"
)

type OrgData struct {
	Users         []UserInfo
	Departments   []DepartmentInfo
	Groups        []GroupInfo
	UserDeptRelations []UserDeptRelation
	UserGroupRelations []UserGroupRelation
}

type UserInfo struct {
	ID          string
	Username    string
	Nickname    string
	Email       string
	Phone       string
	Avatar      string
	DepartmentID string
	Position    string
}

type DepartmentInfo struct {
	ID       string
	Name     string
	ParentID string
	Level    int
}

type GroupInfo struct {
	ID          string
	Name        string
	Description string
}

type UserDeptRelation struct {
	UserID       string
	DepartmentID string
	IsLeader     bool
}

type UserGroupRelation struct {
	UserID  string
	GroupID string
}

type SyncResult struct {
	Success       bool
	UsersCreated  int
	UsersUpdated  int
	UsersDeleted  int
	DeptsCreated  int
	DeptsUpdated  int
	DeptsDeleted  int
	GroupsCreated int
	GroupsUpdated int
	GroupsDeleted int
	Message       string
}

type OrgSyncProvider interface {
	Name() string
	Fetch(ctx context.Context, config string) (*OrgData, error)
	Sync(ctx context.Context, data *OrgData) (*SyncResult, error)
}
