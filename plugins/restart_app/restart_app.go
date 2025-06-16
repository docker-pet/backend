package telegram_auth

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type Options struct {
}

type Plugin struct {
	app        core.App
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

	app.OnRecordAfterUpdateSuccess("app").BindFunc(func(e *core.RecordEvent) error {
      	p.app.Logger().Info("Restarting application in 3 seconds due to app configuration update")

		// Wait for 3 seconds before restarting the app
		// This is to ensure that any ongoing requests are completed before the restart
		go func() {
			time.Sleep(3 * time.Second)
			err := app.Restart()
			if err != nil {
				panic(fmt.Errorf("failed to restart app: %w", err))
			}
		}()

		return e.Next()
	})

	return p, nil
}
