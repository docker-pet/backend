package telegram_miniapp

import (
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	initdata "github.com/telegram-mini-apps/init-data-golang"
)

func (m *TelegramMiniappModule) registerAuthVerifyEndpoint() {
	m.Ctx.App.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/api/telegram_miniapp/auth", func(e *core.RequestEvent) error {
			// Validate request body
			data := struct {
				InitData string `json:"initData" form:"initData"`
			}{}
			if err := e.BindBody(&data); err != nil {
				return e.BadRequestError("Failed to read request data", err)
			}

			// Will return error in case, init data is invalid.
			if err := initdata.Validate(data.InitData, m.appConfig.AppConfig().TelegramBotToken(), m.Config.AuthTokenLifetime); err != nil {
				return e.BadRequestError("Invalid init data", err)
			}

			// Parse init data
			tgUser, err := initdata.Parse(data.InitData)
			if err != nil {
				return e.BadRequestError("Failed to parse data", err)
			}

			// Get user by Telegram ID
			user, err := m.users.GetUserByTelegramId(tgUser.User.ID)
			needToSave := false
			if err != nil {
				newUser, err := m.users.NewUser(tgUser.User.ID)
				if err != nil {
					return e.InternalServerError("Failed to create new user", err)
				}
				user = newUser
				user.SetSynced(types.NowDateTime().AddDate(-20, 0, 0))
				needToSave = true
			}

			// Telegram Username
			if user.TelegramUsername() != tgUser.User.Username {
				user.SetTelegramUsername(tgUser.User.Username)
				needToSave = true
			}

			// Name
			oldName := user.Name()
			user.SetName(tgUser.User.FirstName, tgUser.User.LastName)
			if user.Name() != oldName {
				needToSave = true
			}

			// Language
			if user.Language() != tgUser.User.LanguageCode {
				user.SetLanguage(tgUser.User.LanguageCode)
				needToSave = true
			}

			// New avatar
			avatarChanged := m.users.UploadAvatar(user, tgUser.User.PhotoURL)
			if avatarChanged {
				needToSave = true
			}

			// Save user if needed
			if needToSave {
				if err := m.Ctx.App.Save(user); err != nil {
					return e.InternalServerError("Failed to save user", err)
				}
			}

			return apis.RecordAuthResponse(e, user.Record, "", tgUser)
		})

		return se.Next()
	})
}
