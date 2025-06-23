package app_config

func (m *AppConfigModule) SetBotUsername(botUsername string) error {
	if m.AppConfig().BotUsername() == botUsername || botUsername == "" {
		return nil // No change needed
	}

	m.AppConfig().SetBotUsername(botUsername)
	return m.Ctx.App.Save(m.AppConfig())
}