package syncer

import (
	"fmt"

	"qim-server/model"
	"qim-server/orgsync"
)

func NewProvider(config *model.OrgSyncConfig) (orgsync.OrgSyncProvider, error) {
	switch config.SyncType {
	case "ldap":
		return NewLDAPSyncer(config)
	case "api":
		return NewAPISyncer(config)
	default:
		return nil, fmt.Errorf("不支持的同步类型: %s", config.SyncType)
	}
}
