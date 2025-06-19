package lampa

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"time"

	otpAuthPlugin "github.com/docker-pet/backend/plugins/otp_auth"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
	"github.com/zmwangx/debounce"
)

type Options struct {
  	AuthPlugin *otpAuthPlugin.Plugin
}

type Plugin struct {
	app        core.App
	appConfig  *core.Record
	options    *Options
	lampaUsersCollection *core.Collection
}

// Validate plugin options.
func (p *Plugin) Validate() error {
	if p.options == nil {
		return fmt.Errorf("options is required")
	}

	if p.options.AuthPlugin == nil {
		return fmt.Errorf("AuthPlugin is required")
	}

	return nil
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
		options:   options,
	}

	updateConfigs, _ := debounce.Debounce(
        func() { p.doUpdateConfigs() },
        10 * time.Second,
        debounce.WithLeading(true),
        debounce.WithTrailing(true),
    )

	p.app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// App Configuration
		appConfig, err := app.FindFirstRecordByFilter("app", "id != ''")
    	if err != nil {
      		return err
    	}
		p.appConfig = appConfig

		// Users collection
		lampaUsersCollection, err := app.FindCollectionByNameOrId("lampa_users")
		p.lampaUsersCollection = lampaUsersCollection
		if err != nil {
			return err
		}

		// Get current lampa user
		se.Router.GET("/api/lampa/user", func(e *core.RequestEvent) error {
			claims := p.options.AuthPlugin.ParseCooke(e)

			// Not authenticated
			if claims.UserId == "" {
				return e.JSON(http.StatusUnauthorized, map[string]string{
					"error": "You are not authenticated",
				})
			}

			// Find lampa user
			lampaUser, err := p.UpdateLampaUser(claims.UserId)
			if err != nil {
				return e.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Failed to get lampa user",
				})
			}

			return e.JSON(http.StatusOK, map[string]interface{}{
				"authKey": lampaUser.GetString("authKey"),
				"deviceName": claims.DeviceName,
			})
		})

		// Watch for app configuration changes
		app.OnRecordAfterUpdateSuccess("lampa").BindFunc(func(e *core.RecordEvent) error {
			p.app.Logger().Info("Got lampa config update event")
			updateConfigs()
			return e.Next()
		})

		// Users collection events
		app.OnRecordAfterCreateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
			p.UpdateLampaUser(e.Record.Id)
			return e.Next()
		});

		app.OnRecordAfterUpdateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
			p.UpdateLampaUser(e.Record.Id)
			return e.Next()
		})

		// Lampa users collection events
		app.OnRecordAfterCreateSuccess(p.lampaUsersCollection.Name).BindFunc(func(e *core.RecordEvent) error {
			updateConfigs()
			return e.Next()
		})

		app.OnRecordAfterUpdateSuccess(p.lampaUsersCollection.Name).BindFunc(func(e *core.RecordEvent) error {
			updateConfigs()
			return e.Next()
		})

		app.OnRecordAfterDeleteSuccess(p.lampaUsersCollection.Name).BindFunc(func(e *core.RecordEvent) error {
			updateConfigs()
			return e.Next()
		})

		// Initialize first time
		p.SyncLampaUsers()
		updateConfigs()
		return se.Next()
	})

	return p, nil
}

func (p *Plugin) doUpdateConfigs() {
	lampaConfig, err := p.app.FindFirstRecordByFilter("lampa", "id != ''")
	if err != nil {
		p.app.Logger().Error("Failed to find lampa config", "Err", err)
		return
	}

	// Lampa users
	lampaUsers, err := p.app.FindAllRecords(p.lampaUsersCollection)
	if err != nil {
		p.app.Logger().Error("Failed to find lampa users", "Err", err)
		return
	}
	sort.Slice(lampaUsers, func(i, j int) bool {
		return lampaUsers[i].GetString("authKey") < lampaUsers[j].GetString("authKey")
	})

	// Config folder
	err = os.MkdirAll("./lampa", 0755)
	if err != nil {
		p.app.Logger().Error("Failed to create lampa config folder", "Err", err)
		return
	}

	// Flags
	dlnaEnabled := lampaConfig.GetBool("dlnaEnabled")
	sisiEnabled := lampaConfig.GetBool("sisiEnabled")
	cubEnabled := lampaConfig.GetBool("cubEnabled")
	tmdbProxyEnabled := lampaConfig.GetBool("tmdbProxyEnabled")
	torrServerEnabled := lampaConfig.GetBool("torrServerEnabled")
	serverProxyEnabled := lampaConfig.GetBool("serverProxyEnabled")
	tracksEnabled := lampaConfig.GetBool("tracksEnabled")
	onlineEnabled := lampaConfig.GetBool("onlineEnabled")

	// Manifest config
	manifestConfig := []map[string]interface{}{
		{"enable": onlineEnabled, "dll": "Online.dll"},
		{"enable": sisiEnabled, "dll": "SISI.dll"},
		{"enable": dlnaEnabled, "dll": "DLNA.dll"},
		{"enable": tracksEnabled, "dll": "Tracks.dll", "initspace": "Tracks.ModInit"},
		{"enable": torrServerEnabled, "dll": "TorrServer.dll", "initspace": "TorrServer.ModInit"},
	}
	manifestConfigJson, err := json.MarshalIndent(manifestConfig, "", "  ")
	if err != nil {
		p.app.Logger().Error("Failed to marshal manifest config JSON", "Err", err)
		return
	}
	err = os.WriteFile("./lampa/manifest.json", []byte(string(manifestConfigJson)), 0644)
	if err != nil {
		p.app.Logger().Error("Failed to write manifest.json", "Err", err)
	}

	// Admin password
	adminPassword := lampaConfig.GetString("adminPassword")
	if adminPassword == "" {
		adminPassword = security.RandomString(30)
		p.app.Logger().Warn("Admin password is not set, generating a temporary", "Password", adminPassword)
	}
	err = os.WriteFile("./lampa/passwd", []byte(adminPassword), 0644)
	if err != nil {
		p.app.Logger().Error("Failed to write passwd", "Err", err)
	}

	// Init config
	configInit := lampaConfig.GetString("configInit")
	if configInit == "" {
		configInit = "{}"
	}
	var configInitObj map[string]interface{}
	if err := json.Unmarshal([]byte(configInit), &configInitObj); err != nil {
		p.app.Logger().Error("Failed to parse configInit JSON", "Err", err)
		return
	}

	configInitObj["listenscheme"] = "https"
	configInitObj["listenport"] = 80
	configInitObj["listenhost"] = "lampa." + p.appConfig.GetString("appDomain")
	configInitObj["multiaccess"] = true
	configInitObj["compression"] = false
	configInitObj["chromium"] = map[string]interface{}{
		"enable": true,
	}
	configInitObj["firefox"] = map[string]interface{}{
		"enable": true,
	}
	configInitObj["dlna"] = map[string]interface{}{
		"enable": dlnaEnabled,
		"autoupdatetrackers": dlnaEnabled,
	}
	configInitObj["LampaWeb"] = map[string]interface{}{
		"initPlugins": map[string]bool{
			"dlna": dlnaEnabled,
			"tracks": tracksEnabled,
			"tmdbProxy": tmdbProxyEnabled,
			"online": onlineEnabled,
			"sisi": sisiEnabled,
			"timecode": true,
			"torrserver": torrServerEnabled,
			"backup": true,
			"sync": true,
		},
	}
	configInitObj["tmdb"] = map[string]interface{}{
		"enable": tmdbProxyEnabled,
		"useproxy": true,
		"useproxystream": true,
	}
	configInitObj["serverproxy"] = map[string]interface{}{
		"enable": serverProxyEnabled,
		"verifyip": false,
		"allow_tmdb": true,
		"image": map[string]interface{}{
			"cache": true,
			"cache_rsize": true,
		},
		"buffering": map[string]interface{}{
			"enable": true,
			"rent": 8192,
			"length": 3906,
			"millisecondsTimeout": 5,
		},
	}
	configInitObj["cub"] = map[string]interface{}{
		"enable": cubEnabled,
	}

	// Users
	users := make([]map[string]interface{}, len(lampaUsers))
	for i, u := range lampaUsers {
		users[i] = map[string]interface{}{
			"id":  u.GetString("authKey"),
			"ban": u.GetBool("disabled"),
			"expires": "2040-01-01T00:00:00",
		}
	}
	configInitObj["accsdb"] = map[string]interface{}{
		"enable": true,
		"users":  users,
	}

	// Save config init
	configInitJson, err := json.MarshalIndent(configInitObj, "", "  ")
	if err != nil {
		p.app.Logger().Error("Failed to marshal configInit JSON", "Err", err)
		return
	}
	err = os.WriteFile("./lampa/init.conf", []byte(string(configInitJson)), 0644)
	if err != nil {
		p.app.Logger().Error("Failed to write init.conf", "Err", err)
	}
}

func (p *Plugin) SyncLampaUsers() error {
	// Get all users
	users, err := p.app.FindAllRecords("users")
	if err != nil {
		return err
	}

	// Update each user
	for _, user := range users {
		p.UpdateLampaUser(user.Id)
	}
	p.app.Logger().Info("Lampa users synced successfully")
	return nil
}

func (p *Plugin) UpdateLampaUser(userId string) (*core.Record, error) {
	hasChanges := false

	// Find user record
	user, err := p.app.FindRecordById("users", userId)
	if err != nil {
		p.app.Logger().Error("Failed to find user record", "Err", err)
		return nil, err
	}

	// Find lampa user record
	lampaUser, err := p.app.FindFirstRecordByFilter(p.lampaUsersCollection.Name, fmt.Sprintf("user = '%s'", userId))
	if err != nil {
		p.app.Logger().Info("Failed to find lampa user record, creating new one", "Err", err)
		lampaUser = core.NewRecord(p.lampaUsersCollection)
		lampaUser.Set("user", user.Id)
		lampaUser.Set("authKey", security.RandomString(32))
		hasChanges = true
	}

	// Check if the user role has changed
	userRole := user.GetString("role")
	userDisabled := userRole != "admin" && userRole != "user"
	if lampaUser.GetBool("disabled") != userDisabled {
		lampaUser.Set("disabled", userDisabled)
		hasChanges = true
	}
	
	// Save changes if needed
	if hasChanges {
		if err := p.app.Save(lampaUser); err != nil {
			p.app.Logger().Error("Failed to save lampa user record", "Err", err)
		} else {
			p.app.Logger().Info("Lampa user record updated successfully", "UserId", userId)
		}
	}

	return lampaUser, nil
}
