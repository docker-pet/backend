package otp_auth

import (
	"net/http"
	"time"

	"github.com/docker-pet/backend/models"
	"github.com/golang-jwt/jwt/v4"
	"github.com/pocketbase/pocketbase/core"
)

type CookieClaims struct {
	Pin            string          `json:"pin"`
	UserId         string          `json:"userId"`
	UserRole       models.UserRole `json:"userRole"`
	DeviceName     string          `json:"deviceName"`
	ValidationDate time.Time       `json:"validationDate"`
}

func (m *OtpAuthModule) parseCooke(e *core.RequestEvent) *CookieClaims {
	claims := &CookieClaims{
		Pin:        "",
		UserId:     "",
		UserRole:   "",
		DeviceName: "",
	}

	// Parse cookie
	cookie, err := e.Request.Cookie(m.appConfig.AppConfig().AuthCookieName())
	if err != nil {
		return claims
	}

	// Parse JWT token
	token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.appConfig.AppConfig().AuthSecret()), nil
	})
	if err != nil || !token.Valid {
		return claims
	}

	// Extract claims from the token
	if jwtClaims, ok := token.Claims.(jwt.MapClaims); ok {
		if v, ok := jwtClaims["pin"].(string); ok {
			if len(v) == m.appConfig.AppConfig().AuthPinLength() {
				claims.Pin = v
			}
		}
		if v, ok := jwtClaims["userId"].(string); ok {
			claims.UserId = v
		}
		if v, ok := jwtClaims["userRole"].(string); ok {
			role := models.UserRole(v)
			switch role {
			case models.RoleUser, models.RoleAdmin, models.RoleGuest:
				claims.UserRole = role
			default:
				claims.UserRole = models.RoleGuest
			}
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
		if found := m.keychain.Exists(claims.Pin); !found {
			claims.Pin = ""
			m.Logger.Info("Pin not found", "Pin", claims.Pin)
		}
	}

	// Validate claims
	if claims.UserId != "" {
		if claims.UserRole == "" || claims.ValidationDate.Add(m.Config.SessionVerifyInterval).Before(time.Now()) {
			user, err := m.users.GetUserById(claims.UserId)

			// Save
			if err == nil && user.Role() != models.RoleGuest {
				claims.ValidationDate = time.Now()
				claims.UserRole = user.Role()
			} else {
				claims.Pin = ""
				claims.UserId = ""
				claims.UserRole = ""
				claims.DeviceName = ""
				claims.ValidationDate = time.Time{}
				m.Logger.Debug("Unauthenticated user", "UserId", claims.UserId)
			}

			m.fillCookie(e, *claims)
		}
	}

	return claims
}

func (m *OtpAuthModule) fillCookie(e *core.RequestEvent, claims CookieClaims) {
	domain := m.getAppDomain(e)
	expires := time.Now().Add(10 * 365 * 24 * time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"pin":            claims.Pin,
		"userId":         claims.UserId,
		"deviceName":     claims.DeviceName,
		"validationDate": claims.ValidationDate.Format(time.RFC3339),
		"exp":            expires.Unix(),
	})

	tokenStr, err := token.SignedString([]byte(m.appConfig.AppConfig().AuthSecret()))
	if err != nil {
		m.Logger.Error("Failed to sign JWT token", "Err", err)
		return
	}

	// Set response cookie
	e.SetCookie(&http.Cookie{
		Name:     m.appConfig.AppConfig().AuthCookieName(),
		Value:    tokenStr,
		Path:     "/",
		HttpOnly: true,
		Expires:  expires,
		Domain:   "." + domain,
	})
}
