package telegram_auth

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/patrickmn/go-cache"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type Options struct {
	Expiration	      time.Duration  // Duration after which the OTP code expires
	CleanupInterval   time.Duration  // Interval at which expired OTP codes are cleaned up
	MaxPinGenerationAttempts int     // Maximum attempts to generate a unique PIN code

	AuthVerifyInterval time.Duration // Interval for verifying the OTP auth
}

type CookieClaims struct {
	Pin            string `json:"pin"`
	UserId         string `json:"userId"`
	UserRole	   string `json:"userRole"`
	DeviceName 	   string `json:"deviceName"`
	ValidationDate time.Time `json:"validationDate"`
}

type Plugin struct {
	app        core.App
	options    *Options
	keychain   *cache.Cache
	appConfig  *core.Record
}

// Validate plugin options.
func (p *Plugin) Validate() error {
	if p.options == nil {
		return fmt.Errorf("options is required")
	}

	if p.options.Expiration <= 0 {
		return fmt.Errorf("options.Expiration must be greater than 0")
	}

	if p.options.CleanupInterval <= 0 {
		return fmt.Errorf("options.CleanupInterval must be greater than 0")
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
	keychain := cache.New(options.Expiration, options.CleanupInterval)
	p := &Plugin{
		app:        app,
		keychain:   keychain,
		options:    options,
	}


	p.app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// Get app configuration
		appConfig, err := app.FindFirstRecordByFilter("app", "id != ''")
		p.appConfig = appConfig
    	if err != nil {
      		return err
    	}

		authPinLength := appConfig.GetInt("authPinLength")

		se.Router.POST("/api/otp/pin", func(e *core.RequestEvent) error {
            claims := p.ParseCooke(e)

			// Device name is required
			claims.DeviceName = e.Request.Header.Get("X-Device-Name")
			if claims.DeviceName == "" || len(claims.DeviceName) > 86 {
				return e.JSON(http.StatusOK, map[string]string{"type": "invalid_device_name"})
			}

			// Already authenticated
			if claims.UserId != "" {
				return e.JSON(http.StatusOK, map[string]string{"type": "already_authenticated"})
			}

			// Generate a new PIN code
			if claims.Pin == "" {
				// Generate unique PIN code
				for i := 0; i < options.MaxPinGenerationAttempts; i++ {
					pin, err := GenCode(authPinLength)
					if err != nil {
						return e.JSON(http.StatusOK, map[string]string{"type": "failed_to_generate_pin"})
					}
					if _, found := keychain.Get(pin); !found {
						claims.Pin = pin
						break
					}
					// If we reach the maximum attempts, return an error
					if i == options.MaxPinGenerationAttempts-1 {
						return e.JSON(http.StatusOK, map[string]string{"type": "failed_to_generate_pin"})
					}
				}

				// Reserve the PIN code in the keychain
				keychain.Set(claims.Pin, nil, cache.DefaultExpiration)

				// Save cookie
				p.FillCookie(e, *claims)
			}

			// If pin confirmed
			if pinStatus, found := keychain.Get(claims.Pin); found && pinStatus != nil {
				claims.Pin = ""
				claims.UserId = pinStatus.(string)
				claims.ValidationDate = time.Now()
				p.FillCookie(e, *claims)

				return e.JSON(http.StatusOK, map[string]string{"type": "confirmed"})
			}

			return e.JSON(http.StatusOK, map[string]string{"type": "pin", "pin": claims.Pin})
		})

		se.Router.POST("/api/otp/confirm", func(e *core.RequestEvent) error {
			// Unauhorized
			if e.Auth == nil || e.Auth.GetString("role") == "guest" || e.Auth.GetString("role") == "" {
				return e.JSON(http.StatusOK, map[string]string{"type": "unauthorized"})
			}

			// Invalid pin
			pinCode := e.Request.Header.Get("X-Auth-Pin")
			if _, found := p.keychain.Get(pinCode); !found {
				return e.JSON(http.StatusOK, map[string]string{"type": "not_found"})
			}


			// Confirm pin
			p.keychain.Set(pinCode, e.Auth.Id, cache.DefaultExpiration)
			return e.JSON(http.StatusOK, map[string]string{"type": "confirmed"})
		}).Bind(apis.RequireAuth("users"))

		se.Router.Any("/api/otp/verify", func(e *core.RequestEvent) error {
			claims := p.ParseCooke(e)

			// Already authenticated
			if claims.UserId != "" {
				return e.JSON(http.StatusOK, map[string]string{"type": "already_authenticated"})
			}

			// Redirect to auth page
			redirectUrl := "https://" + p.appConfig.GetString("appDomain")
			if e.Request.Header.Get("Remote-Addr") != "" {
				redirectUrl = "https://" + e.Request.Header.Get("Remote-Addr") + e.Request.Header.Get("Original-URI")
			}

			return e.Redirect(302, fmt.Sprintf(
				"https://%s/auth?redirect=%s",
				p.appConfig.GetString("appDomain"),
				url.QueryEscape(redirectUrl),
			))
		})

		return se.Next()
	})

	return p, nil
}

// TODO: Optimize this function
func (p *Plugin) ParseCooke(e *core.RequestEvent) (*CookieClaims) {
	claims := &CookieClaims{
		Pin:        "",
		UserId:     "",
		UserRole:   "",
		DeviceName: "",
	}

	// Parse cookie
	cookie, err := e.Request.Cookie(p.appConfig.GetString("authCookieName"));
	if err != nil {
		return claims
	}

	// Parse JWT token
	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		return []byte(p.appConfig.GetString("authSecret")), nil
	})
	if err != nil || !token.Valid {
		return claims
	}

	// Extract claims from the token
	if jwtClaims, ok := token.Claims.(jwt.MapClaims); ok {
		if v, ok := jwtClaims["pin"].(string); ok {
			if len(v) == p.appConfig.GetInt("authPinLength") {
				claims.Pin = v
			}
		}
		if v, ok := jwtClaims["userId"].(string); ok {
			claims.UserId = v
		}
		if v, ok := jwtClaims["userRole"].(string); ok {
			claims.UserRole = v
		}
		if v, ok := jwtClaims["deviceName"].(string); ok {
			claims.DeviceName = v
		}
		if v, ok := jwtClaims["validationDate"].(string); ok {
			validationDate, err := time.Parse(time.RFC3339, v)
			if err == nil {
				claims.ValidationDate = validationDate
			} else {
				// If parsing fails, set to zero value
				claims.ValidationDate = time.Time{}
			}
		}
	}

	// Pin not exists
	if claims.Pin != "" {
		if _, found := p.keychain.Get(claims.Pin); !found {
			claims.Pin = ""
			p.app.Logger().Info("Pin not found", "Pin", claims.Pin)
		}
	}

	// Validate claims
	if claims.UserId != "" {
		if claims.UserRole == "" || claims.ValidationDate.Add(p.options.AuthVerifyInterval).Before(time.Now()) {
			record, err := p.app.FindRecordById("users", claims.UserId)

			// Save
			if err == nil && record.GetString("role") != "guest" {
				claims.ValidationDate = time.Now()
				claims.UserRole = record.GetString("role")
			} else {
				claims.Pin = ""
				claims.UserId = ""
				claims.UserRole = ""
				claims.DeviceName = ""
				claims.ValidationDate = time.Time{}

				p.app.Logger().Info("Unauthenticated user", "UserId", claims.UserId)
			}

			p.FillCookie(e, *claims)
		}
	}

	return claims
}

func (p *Plugin) FillCookie(e *core.RequestEvent, claims CookieClaims) {
	expires := time.Now().Add(10 * 365 * 24 * time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"pin":            claims.Pin,
		"userId":         claims.UserId,
		"deviceName":     claims.DeviceName,
		"validationDate": claims.ValidationDate.Format(time.RFC3339),
		"exp": expires.Unix(),
	})

	tokenStr, err := token.SignedString([]byte(p.appConfig.GetString("authSecret")))
	if err != nil {
		p.app.Logger().Error("Failed to sign JWT token", "Err", err)
		return
	}

	// Set response cookie
	e.SetCookie(&http.Cookie{
		Name:     p.appConfig.GetString("authCookieName"),
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		Expires:  expires,
		Domain:   "." + p.appConfig.GetString("appDomain"),
	})
}

func GenCode(n int) (string, error) {
    const digits = "0123456789"
    b := make([]byte, n)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    for i := range b {
        b[i] = digits[int(b[i])%len(digits)]
    }
    return string(b), nil
}