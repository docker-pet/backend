package telegram_bot

import (
	tele "gopkg.in/telebot.v4"
)

func (m *TelegramBotModule) useOnChatJoinRequest() {
	m.Bot.Handle(tele.OnChatJoinRequest, func(c tele.Context) error {
		if c.Chat().ID != m.appConfig.AppConfig().TelegramChannelId() {
			return nil
		}

		sender := c.ChatJoinRequest().Sender
		user, err := m.handleSender(sender)

		// Set join pending status
		if err != nil {
			user.SetJoinPending(true)
			err = m.Ctx.App.Save(user)
		}

		if err != nil {
			m.Logger.Error(
				"Failed to handle chat join request",
				"Error", err,
				"UserId", sender.ID,
			)
		}

		return nil
	})
}
