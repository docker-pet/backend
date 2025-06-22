package telegram_bot

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v4"
)

func (m *TelegramBotModule) useStartCommand() {
	m.Bot.Handle("/start", func(c tele.Context) error {
		if c.Chat().Type != tele.ChatPrivate {
			return nil
		}

		m.handleSender(c.Sender())

		messageText := ""
		buttonText := ""

		switch strings.ToLower((c.Sender().LanguageCode + "xx")[:2]) {
		case "uk":
			messageText = "👋 Привіт! Щоб продовжити, запусти застосунок за кнопкою нижче:"
			buttonText = "Запустити"
		case "en":
			messageText = "👋 Hi! To continue, launch the app using the button below:"
			buttonText = "Launch"
		default:
			messageText = "👋 Привет! Для продолжения запусти приложение по кнопке ниже:"
			buttonText = "Запустить"
		}

		btn := tele.InlineButton{
			Text: buttonText,
			WebApp: &tele.WebApp{
				URL: fmt.Sprintf("https://%s", m.appConfig.AppConfig().AppDomain()),
			},
		}

		return c.Send(messageText, &tele.SendOptions{
			ReplyMarkup: &tele.ReplyMarkup{
				InlineKeyboard: [][]tele.InlineButton{{btn}},
			},
		})
	})
}
