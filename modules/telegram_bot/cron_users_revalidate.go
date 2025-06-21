package telegram_bot

import (
	"strings"
	"time"

	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tools/types"
	tele "gopkg.in/telebot.v4"
)

func (m *TelegramBotModule) useUsersRevalidateCron() {
	m.Ctx.App.Cron().MustAdd("telegram_bot_channel_users", m.Config.CronUserSyncExpression, func() {
		if m.Bot == nil {
			m.Logger.Warn("Telegram bot is not initialized, skipping user sync")
			return
		}

		users, err := m.users.FindUsersByFilter("synced < {:synced}", "+synced", m.Config.CronUsersPerSync, 0, dbx.Params{
			"synced": time.Now().Add(-m.Config.CronUserSyncInterval).Format(time.RFC3339),
		})

		if err != nil {
			m.Logger.Warn("Failed to fetch users for Telegram bot channel update:", "Err", err)
			return
		}

		// Check if has access to the main channel
		mainChannel, err := m.Bot.ChatByID(m.appConfig.AppConfig().TelegramChannelId())
		if err != nil {
			m.Logger.Warn("Telegram bot is not a member of the main channel, skipping user sync")
			return
		}

		// Check if has access to the premium channel
		premiumChannel, err := m.Bot.ChatByID(m.appConfig.AppConfig().TelegramPremiumChannelId())
		if err != nil && premiumChannel != nil {
			m.Logger.Warn("Telegram bot is not a member of the premium channel, skipping premium user sync")
		}

		// Fetch and update user records
		for _, user := range users {
			user.SetSynced(types.NowDateTime())
			userQuery := &tele.User{ID: user.TelegramId()}

			// Main channel
			mainMember, err := m.Bot.ChatMemberOf(mainChannel, userQuery)
			if err == nil {
				updatedUser, err := m.handleChatMember(mainMember, mainChannel.ID)
				if err == nil {
					user = updatedUser
				}
			} else if strings.Contains(err.Error(), "PARTICIPANT_ID_INVALID") {
				user.SetRole(models.RoleGuest)
			}

			// Premium channel
			if premiumChannel != nil {
				premiumMember, err := m.Bot.ChatMemberOf(premiumChannel, userQuery)
				if err == nil {
					updatedUser, err := m.handleChatMember(premiumMember, premiumChannel.ID)
					if err == nil {
						user = updatedUser
					}
				} else if strings.Contains(err.Error(), "PARTICIPANT_ID_INVALID") {
					user.SetPremium(false)
				}
			}

			// Save user record if needed
			if err := m.Ctx.App.Save(user); err != nil {
				m.Logger.Error(
					"Failed to save user record after Telegram bot channel update",
					"Error", err,
					"UserId", user.Id,
				)
			}
		}
	})
}
