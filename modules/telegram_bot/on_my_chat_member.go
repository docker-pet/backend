package telegram_bot

import (
	tele "gopkg.in/telebot.v4"
)

func (m *TelegramBotModule) useOnMyChatMember() {
	m.Bot.Handle(tele.OnMyChatMember, func(c tele.Context) error {
		// No role change, do nothing
		if c.ChatMember().NewChatMember.Role == c.ChatMember().OldChatMember.Role {
			return nil
		}

		switch c.ChatMember().NewChatMember.Role {
		case tele.Administrator, tele.Creator:
			m.Logger.Info(
				"Telegram bot joined chat as administrator:",
				"ChatId", c.Chat().ID,
				"Title", c.Chat().Title,
			)

		case tele.Left:
		case tele.Kicked:
			m.Logger.Info(
				"Telegram bot left chat:",
				"ChatId", c.Chat().ID,
				"Title", c.Chat().Title,
			)

		case tele.Restricted:
			m.Logger.Info(
				"Telegram bot restricted in chat & left:",
				"ChatId", c.Chat().ID,
				"Title", c.Chat().Title,
			)
			return c.Bot().Leave(c.Chat())

		default:
			m.Logger.Info(
				"Telegram bot received unknown chat member update:",
				"ChatId", c.Chat().ID,
				"Title", c.Chat().Title,
			)
			return c.Bot().Leave(c.Chat())
		}

		return nil
	})
}
