package telegram_bot

import (
	"fmt"

	tele "gopkg.in/telebot.v4"
)

func (m *TelegramBotModule) useAccessMiddleware() {
	m.Bot.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			// Private chat (between user and bot)
			if c.Chat().Type == tele.ChatPrivate {
				return next(c)
			}

			// Unauthorized chat check
			if c.Chat().ID != m.appConfig.AppConfig().TelegramChannelId() && c.Chat().ID != m.appConfig.AppConfig().TelegramPremiumChannelId() {
				m.Logger.Info(
					"Telegram bot received message from unauthorized chat.",
					"ChatId", c.Chat().ID,
					"Title", c.Chat().Title,
				)

				c.Send(
					"This bot is not authorized to work in this chat (<code>"+fmt.Sprint(c.Chat().ID)+"</code>).",
					&tele.SendOptions{
						ParseMode:           tele.ModeHTML,
						DisableNotification: true,
					},
				)

				return c.Bot().Leave(c.Chat())
			}

			return next(c)
		}
	})
}
