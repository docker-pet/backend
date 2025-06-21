package otp_auth

import (
	"time"

	"github.com/docker-pet/backend/models"
	"github.com/patrickmn/go-cache"
)

type KeyChainOptions struct {
	Expiration      time.Duration
	CleanupInterval time.Duration
}

type KeyChain struct {
	keychain *cache.Cache
	options  *KeyChainOptions
}

type KeyChainUser struct {
	UserId   string          `json:"userId"`
	UserRole models.UserRole `json:"userRole"`
}

func NewKeyChain(options *KeyChainOptions) *KeyChain {
	return &KeyChain{
		keychain: cache.New(options.Expiration, options.CleanupInterval),
		options:  options,
	}
}

func (kc *KeyChain) Reserve(code string) bool {
	err := kc.keychain.Add(code, nil, kc.options.Expiration)
	return err == nil
}

func (kc *KeyChain) Exists(code string) bool {
	_, found := kc.keychain.Get(code)
	return found
}

func (kc *KeyChain) Confirm(code string, userId string, role models.UserRole) {
	value := &KeyChainUser{
		UserId:   userId,
		UserRole: role,
	}

	kc.keychain.Set(code, value, kc.options.Expiration)
}

func (kc *KeyChain) IsConfirmed(code string) (*KeyChainUser, bool) {
	rawValue, found := kc.keychain.Get(code)
	if !found {
		return nil, false
	}

	value, ok := rawValue.(*KeyChainUser)
	if !ok || value.UserId == "" || value.UserRole == "" {
		return nil, false
	}

	return value, true
}
