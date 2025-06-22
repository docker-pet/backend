package telegram_bot

import (
	"fmt"

	"github.com/docker-pet/backend/models"
	"github.com/pocketbase/pocketbase/tools/types"
	tele "gopkg.in/telebot.v4"
)

func (m *TelegramBotModule) handleChatMember(member *tele.ChatMember, channelId int64) (*models.User, error) {
	user, err := m.users.GetUserByTelegramId(member.User.ID)
	needToSave := false
	if err != nil {
		user, err = m.users.NewUser(member.User.ID)
		if err != nil {
			return nil, err
		}
		needToSave = true
	}

	// Sync data
	user.SetSynced(types.NowDateTime())

	// Role detection
	if channelId == m.appConfig.AppConfig().TelegramChannelId() {
		role := models.RoleGuest
		switch member.Role {
		case tele.Creator, tele.Administrator:
			role = models.RoleAdmin
		case tele.Member:
			role = models.RoleUser
		default:
			role = models.RoleGuest
		}

		if user.Role() != role {
			user.SetRole(role)
			needToSave = true
		}
	}

	// Premium detection
	if channelId == m.appConfig.AppConfig().TelegramPremiumChannelId() {
		premium := false
		switch member.Role {
		case tele.Administrator, tele.Creator, tele.Member:
			premium = true
		default:
			premium = false
		}

		if user.Premium() != premium {
			user.SetPremium(premium)
			needToSave = true
		}
	}

	// Username
	if user.TelegramUsername() != member.User.Username {
		user.SetTelegramUsername(member.User.Username)
		needToSave = true
	}

	// Name
	oldName := user.Name()
	user.SetName(member.User.FirstName, member.User.LastName)
	if user.Name() != oldName {
		needToSave = true
	}

	// Language code
	if user.Language() != member.User.LanguageCode {
		user.SetLanguage(member.User.LanguageCode)
		needToSave = true
	}

	// Need to save user?
	if needToSave {
		if err := m.Ctx.App.Save(user); err != nil {
			return nil, fmt.Errorf("failed to save user: %w", err)
		}
	}

	return user, nil
}

func (m *TelegramBotModule) handleSender(sender *tele.User) (*models.User, error) {
	user, err := m.users.GetUserByTelegramId(sender.ID)
	needToSave := false
	if err != nil {
		user, err = m.users.NewUser(sender.ID)
		if err != nil {
			return nil, err
		}
		needToSave = true
	}


	// Username
	if user.TelegramUsername() != sender.Username {
		user.SetTelegramUsername(sender.Username)
		needToSave = true
	}

	// Name
	oldName := user.Name()
	user.SetName(sender.FirstName, sender.LastName)
	if user.Name() != oldName {
		needToSave = true
	}

	// Language code
	if user.Language() != sender.LanguageCode {
		user.SetLanguage(sender.LanguageCode)
		needToSave = true
	}

	// Need to save user?
	if needToSave {
		if err := m.Ctx.App.Save(user); err != nil {
			return nil, fmt.Errorf("failed to save user: %w", err)
		}
	}

	return user, nil
}
