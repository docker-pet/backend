package lampa

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

type Options struct {
}

type Plugin struct {
	app        core.App
	appConfig  *core.Record
	options    *Options
}

// Validate plugin options.
func (p *Plugin) Validate() error {
	if p.options == nil {
		return fmt.Errorf("options is required")
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
	}

	p.app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		appConfig, err := app.FindFirstRecordByFilter("app", "id != ''")
    	if err != nil {
      		return err
    	}
		p.appConfig = appConfig

		// Watch for app configuration changes
		app.OnRecordAfterUpdateSuccess("lampa").BindFunc(func(e *core.RecordEvent) error {
			p.app.Logger().Info("Got lampa config update event")
			p.UpdateConfigs()
			return e.Next()
		})

		// Initialize first time
		p.UpdateConfigs()
		return se.Next()
	})

	return p, nil
}

func (p *Plugin) UpdateConfigs() {
	lampaConfig, err := p.app.FindFirstRecordByFilter("lampa", "id != ''")
	if err != nil {
		p.app.Logger().Error("Failed to find lampa config", "Err", err)
		return
	}

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
