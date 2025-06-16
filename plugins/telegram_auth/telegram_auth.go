package telegram_auth

import (
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	initdata "github.com/telegram-mini-apps/init-data-golang"
	tele "gopkg.in/telebot.v4"
	"resty.dev/v3"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/security"
)

type Options struct {
	// BotToken is a Telegram bot token.
	// You can get it from @BotFather.
	BotToken string

	// CollectionKey is a collection key (name or id) for PocketBase auth collection.
	CollectionKey string
}

type Plugin struct {
	app        core.App
	options    *Options
	collection *core.Collection
	httpClient *resty.Client
}

type ChatMemberOrUser interface {
    tele.ChatMember | tele.User
}

// Validate plugin options.
func (p *Plugin) Validate() error {
	if p.options == nil {
		return fmt.Errorf("options is required")
	}

	if p.options.CollectionKey == "" {
		return fmt.Errorf("collection key is required")
	}

	return nil
}

// Get users collection by key
func (p *Plugin) GetCollection() (*core.Collection, error) {
	// If collection object stored in plugin - return it
	if p.collection != nil {
		return p.collection, nil
	}

	// If no collection object - find it, store and return
	if collection, err := p.app.FindCollectionByNameOrId(p.options.CollectionKey); err != nil {
		return nil, err
	} else {
		p.collection = collection
		return collection, nil
	}
}

// Register the register plugin and panic if error occurred
func Register(app core.App, options *Options) *Plugin {
	if p, err := RegisterWrapper(app, options); err != nil {
		panic(err)
	} else {
		return p
	}
}

// Plugin registration
func RegisterWrapper(app core.App, options *Options) (*Plugin, error) {
	p := &Plugin{
		app:        app,
		options:    options,
		httpClient: resty.New(),
	}

	// Validate options
	if err := p.Validate(); err != nil {
		return p, err
	}

	// Http Client
	p.httpClient.SetTimeout(3 * time.Second)
	defer p.httpClient.Close()

	// Route
	p.app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Get app configuration
		appConfig, err := app.FindFirstRecordByFilter("app", "id != ''")
    	if err != nil {
      		return err
    	}
		botToken := appConfig.GetString("telegramBotToken")

		// Auth endpoint
		se.Router.POST("/api/telegram/auth", func(e *core.RequestEvent) error {
			// Collection
			collection, err := p.GetCollection()
			if err != nil {
				return e.InternalServerError("The server is not ready yet", err)
			}

			// Validate request body
			data := struct {
				InitData string `json:"initData" form:"initData"`
			}{}
			if err := e.BindBody(&data); err != nil {
				return e.BadRequestError("Failed to read request data", err)
			}

			// Token expiration time
			expIn := 12 * time.Hour

			// Will return error in case, init data is invalid.
			if err := initdata.Validate(data.InitData, botToken, expIn); err != nil {
				return e.BadRequestError("Invalid init data", err)
			}

			// Parse init data
			tgUser, err := initdata.Parse(data.InitData)
			if err != nil {
				return e.BadRequestError("Failed to parse data", err)
			}

			// Exist user
			user, err := app.FindFirstRecordByData(collection.Id, "telegramId", tgUser.User.ID)

			// Need to create a new user
			if err != nil {
				user = core.NewRecord(collection)
				user.Set("telegramId", tgUser.User.ID)
				user.Set("premium", false)
				user.Set("role", "guest")
				user.Set("email", fmt.Sprintf("%d@telegram.internal", tgUser.User.ID))
				user.Set("language", tgUser.User.LanguageCode)
				user.Set("password", security.RandomString(30))
				user.Set("synced", "2000-01-01T00:00:00Z")

				p.app.Logger().Info("Creating new user from Telegram auth", "userId", user.Id, "telegramId", tgUser.User.ID)
			}

			// Update user data
			user.Set("telegramUsername", tgUser.User.Username)
			user.Set("name", strings.TrimSpace(tgUser.User.FirstName+" "+tgUser.User.LastName))

			// New avatar
			avatarHash := fmt.Sprintf("%x", md5.Sum([]byte(tgUser.User.PhotoURL)))
			if avatarHash != user.GetString("avatarHash") && tgUser.User.PhotoURL != "" {
				// New avatar, save it
				avatarResponse, err := p.httpClient.R().
					Get(tgUser.User.PhotoURL)

				if err != nil {
					p.app.Logger().Warn("Failed to download user avatar", "error", err, "userId", user.Id, "avatarUrl", tgUser.User.PhotoURL)
				} else {
					// Make file from bytes
					avatar, avatarErr := filesystem.NewFileFromBytes(avatarResponse.Bytes(), "avatar.jpg")
					if avatarErr != nil {
						p.app.Logger().Warn("Failed to create avatar file from bytes", "error", avatarErr, "userId", user.Id)
					} else {
						// Save avatar to user
						p.app.Logger().Info("User avatar downloaded", "userId", user.Id, "avatarUrl", tgUser.User.PhotoURL)
						user.Set("avatarHash", avatarHash)
						user.Set("avatar", avatar)
					}
				}
			}

			// Save
			if err := p.app.Save(user); err != nil {
				return e.InternalServerError("Failed to save user", err)
			}

			return apis.RecordAuthResponse(e, user, "", tgUser)
		})

		return se.Next()
	})

	return p, nil
}

// Auth bot user
// TODO: Join http endpoint and this method
func (p *Plugin) AuthBotUser(u tele.User, status *tele.MemberStatus, premium *bool) (*core.Record, error) {
	// Collection
	collection, err := p.GetCollection()
	if err != nil {
		return nil, fmt.Errorf("the server is not ready yet: %w", err)
	}

	// Exist user
	user, err := p.app.FindFirstRecordByData(collection.Id, "telegramId", u.ID)

	// Need to create a new user
	if err != nil {
		user = core.NewRecord(collection)
		user.Set("telegramId", u.ID)
		user.Set("premium", false)
		user.Set("role", "guest")
		user.Set("email", fmt.Sprintf("%d@telegram.internal", u.ID))
		user.Set("language", u.LanguageCode)
		user.Set("password", security.RandomString(30))
		user.Set("synced", "2000-01-01T00:00:00Z")
		p.app.Logger().Info("Creating new user from Telegram bot", "userId", user.Id, "telegramId", u.ID)
	}

	// Update user data
	user.Set("telegramUsername", u.Username)
	user.Set("name", strings.TrimSpace(u.FirstName+" "+u.LastName))

	// Premium status changed
	if premium != nil && user.GetBool("premium") != *premium {
		user.Set("synced", time.Now().Format(time.RFC3339))
		user.Set("premium", *premium)
	}

	// Role update
	if status != nil {
		oldRole := user.GetString("role")
		user.Set("synced", time.Now().Format(time.RFC3339))
		
		switch *status {
		case tele.Creator, tele.Administrator:
			user.Set("role", "admin")
		case tele.Member:
			user.Set("role", "user")
		default:
			user.Set("role", "guest")
		}

		if user.GetString("role") != oldRole {
			p.app.Logger().Info("User role updated from Telegram bot", "userId", user.Id, "telegramId", u.ID, "oldRole", oldRole, "newRole", user.GetString("role"))
		}
	}

	// Join pending
	if user.GetString("role") != "guest" {
		user.Set("joinPending", false)
	}

	// Save
	if err := p.app.Save(user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return user, nil
}

func (p *Plugin) WhenUserLeft(user *core.Record) error {
	user.Set("role", "guest")
	user.Set("premium", false)
	user.Set("synced", time.Now().Format(time.RFC3339))
	return p.app.Save(user)
}

func (p *Plugin) UpdateUserPremium(user *core.Record, premium bool) error {
	if user.GetBool("premium") != premium {
		user.Set("premium", premium)
		user.Set("synced", time.Now().Format(time.RFC3339))
		return p.app.Save(user)
	}

	return nil
}

func (p *Plugin) UpdateUserJoinPending(user *core.Record, pending bool) error {
	if user.GetBool("joinPending") != pending {
		user.Set("joinPending", pending)
		user.Set("synced", time.Now().Format(time.RFC3339))
		return p.app.Save(user)
	}

	return nil
}