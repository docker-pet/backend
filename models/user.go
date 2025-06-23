package models

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/pocketbase/pocketbase/tools/types"
)

var _ core.RecordProxy = (*User)(nil)

type User struct {
	core.BaseRecordProxy
}

func (a *User) TelegramId() int64 {
	return int64(a.GetInt("telegramId"))
}

func (a *User) SetTelegramId(id int64) {
	a.Set("telegramId", id)
	a.Set("email", fmt.Sprintf("%d@telegram.internal", id))
	a.Set("password", security.RandomString(30))
	a.Set("outlineToken", security.RandomString(32))
}

func (a *User) TelegramUsername() string {
	return a.GetString("telegramUsername")
}

func (a *User) SetTelegramUsername(username string) {
	a.Set("telegramUsername", username)
}

func (a *User) Name() string {
	return a.GetString("name")
}

func (a *User) SetName(firstName string, lastName string) {
	a.Set("name", strings.TrimSpace(firstName+" "+lastName))
}

func (a *User) Language() string {
	return a.GetString("language")
}

func (a *User) SetLanguage(language string) {
	a.Set("language", language)
}

func (a *User) Role() UserRole {
	role := UserRole(a.GetString("role"))
	switch role {
	case RoleUser, RoleAdmin, RoleGuest:
		return role
	default:
		return RoleGuest
	}
}

func (a *User) SetRole(role UserRole) {
	a.Set("role", string(role))

	if role != RoleGuest {
		a.SetJoinPending(false)
	}
}

func (a *User) IsActive() bool {
	return a.Role() != RoleGuest
}

func (a *User) Premium() bool {
	return a.GetBool("premium")
}

func (a *User) SetPremium(premium bool) {
	a.Set("premium", premium)
}

func (a *User) JoinPending() bool {
	return a.GetBool("joinPending")
}

func (a *User) SetJoinPending(pending bool) {
	a.Set("joinPending", pending)
}

func (a *User) AvatarHash() string {
	return a.GetString("avatarHash")
}

func (a *User) SetAvatar(file *filesystem.File, hash string) {
	a.Set("avatarHash", hash)
	a.Set("avatar", file)
}

func (a *User) OutlineToken() string {
	return a.GetString("outlineToken")
}

func (a *User) GenerateOutlineToken() {
	a.Set("outlineToken", security.RandomString(32))
}

func (a *User) OutlinePrefixEnabled() bool {
	return a.GetBool("outlinePrefixEnabled")
}

func (a *User) SetOutlinePrefixEnabled(enabled bool) {
	a.Set("outlinePrefixEnabled", enabled)
}

func (a *User) OutlineReverseServerEnabled() bool {
	return a.GetBool("outlineReverseServerEnabled")
}

func (a *User) SetOutlineReverseServerEnabled(enabled bool) {
	a.Set("outlineReverseServerEnabled", enabled)
}

func (a *User) OutlineServer() string {
	return a.GetString("outlineServer")
}

func (a *User) SetOutlineServer(serverId string) {
	a.Set("outlineServer", serverId)
}

func (a *User) Synced() types.DateTime {
	return a.GetDateTime("synced")
}

func (a *User) SetSynced(date types.DateTime) {
	a.Set("synced", date)
}

func (a *User) Created() types.DateTime {
	return a.GetDateTime("created")
}

func (a *User) Updated() types.DateTime {
	return a.GetDateTime("updated")
}
