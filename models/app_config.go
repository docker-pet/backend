package models

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

var _ core.RecordProxy = (*AppConfig)(nil)

type AppConfig struct {
	core.BaseRecordProxy
}

func (a *AppConfig) AppDomain() string {
	return a.GetString("appDomain")
}

func (a *AppConfig) SetAppDomain(value string) {
	a.Set("appDomain", value)
}

func (a *AppConfig) TelegramBotToken() string {
	return a.GetString("telegramBotToken")
}

func (a *AppConfig) SetTelegramBotToken(value string) {
	a.Set("telegramBotToken", value)
}

func (a *AppConfig) TelegramChannelId() int64 {
	return int64(a.GetInt("telegramChannelId"))
}

func (a *AppConfig) SetTelegramChannelId(value int64) {
	a.Set("telegramChannelId", value)
}

func (a *AppConfig) TelegramChannelInviteLink() string {
	return a.GetString("telegramChannelInviteLink")
}

func (a *AppConfig) SetTelegramChannelInviteLink(value string) {
	a.Set("telegramChannelInviteLink", value)
}

func (a *AppConfig) TelegramPremiumChannelId() int64 {
	return int64(a.GetInt("telegramPremiumChannelId"))
}

func (a *AppConfig) SetTelegramPremiumChannelId(value int64) {
	a.Set("telegramPremiumChannelId", value)
}

func (a *AppConfig) TelegramPremiumChannelInviteLink() string {
	return a.GetString("telegramPremiumChannelInviteLink")
}

func (a *AppConfig) SetTelegramPremiumChannelInviteLink(value string) {
	a.Set("telegramPremiumChannelInviteLink", value)
}

func (a *AppConfig) AuthSecret() string {
	return a.GetString("authSecret")
}

func (a *AppConfig) GenerateAuthSecret(value string) {
	a.Set("authSecret", security.RandomString(32))
}

func (a *AppConfig) AuthCookieName() string {
	return a.GetString("authCookieName")
}

func (a *AppConfig) GenerateAuthCookieName(value string) {
	a.Set("authCookieName", "auth_" + security.RandomString(12))
}

func (a *AppConfig) AuthPinLength() int {
	return a.GetInt("authPinLength")
}

func (a *AppConfig) SetAuthPinLength(value int) {
	a.Set("authPinLength", value)
}