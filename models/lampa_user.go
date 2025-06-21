package models

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

var _ core.RecordProxy = (*LampaUser)(nil)

type LampaUser struct {
	core.BaseRecordProxy
}

func (a *LampaUser) UserId() string {
	return a.GetString("user")
}

func (a *LampaUser) SetUserId(id string) {
	a.Set("user", id)
}

func (a *LampaUser) AuthKey() string {
	return a.GetString("authKey")
}

func (a *LampaUser) GenerateAuthKey() {
	a.Set("authKey", security.RandomString(32))
}

func (a *LampaUser) Disabled() bool {
	return a.GetBool("disabled")
}

func (a *LampaUser) SetDisabled(disabled bool) {
	a.Set("disabled", disabled)
}
