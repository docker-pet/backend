package telegram_bot

import (
	tele "gopkg.in/telebot.v4"
)

func (m *TelegramBotModule) useOnChatMember() {
	m.Bot.Handle(tele.OnChatMember, func(c tele.Context) error {
		member := c.ChatMember().NewChatMember
		_, err := m.handleChatMember(member, c.Chat().ID)
		if err != nil {
			m.Logger.Error(
				"Failed to handle chat member event",
				"Error", err,
				"ChatId", c.Chat().ID,
				"UserId", member.User.ID,
			)
		}

		return nil
	})
}
