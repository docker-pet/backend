package outline

import (
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/tools/security"
)

type Token struct {
	UserId     string
	Token      string
	CreatedAt  time.Time
	LastUsedAt time.Time
}

type TokenStore struct {
	// tokens[serverId][userId] = *Token
	tokens map[string]map[string]*Token
	mu     sync.RWMutex

	// Configuration
	slidingTTL   time.Duration // e.g., 6h
	absoluteTTL  time.Duration // e.g., 24h
	cleanupIntvl time.Duration // how often to scan

	// onChange listeners
	onChangeListeners []func(serverId string)

	TechnicalKeyName   string
	TechnicalKeySecret string
}

// NewTokenStore initializes a token store with cleanup.
func NewTokenStore(slidingTTL, absoluteTTL, cleanupInterval time.Duration, TechnicalKeyName, TechnicalKeySecret string) *TokenStore {
	s := &TokenStore{
		tokens:       make(map[string]map[string]*Token),
		slidingTTL:   slidingTTL,
		absoluteTTL:  absoluteTTL,
		cleanupIntvl: cleanupInterval,

		TechnicalKeyName:   TechnicalKeyName,
		TechnicalKeySecret: TechnicalKeySecret,
	}
	go s.cleanupLoop()
	return s
}

// SubscribeOnChange adds a listener for token add/remove events (not for updates).
func (s *TokenStore) SubscribeOnChange(listener func(serverId string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onChangeListeners = append(s.onChangeListeners, listener)
}

// callOnChange calls all listeners for the given serverId.
func (s *TokenStore) callOnChange(serverId string) {
	for _, listener := range s.onChangeListeners {
		go listener(serverId)
	}
}

func (s *TokenStore) GetOrGenerate(userId string, serverId string) (*Token, error) {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure map for serverId exists
	if s.tokens[serverId] == nil {
		s.tokens[serverId] = make(map[string]*Token)
	}

	found := s.tokens[serverId][userId]
	if found != nil {
		// Check absolute lifetime
		if now.Sub(found.CreatedAt) > s.absoluteTTL {
			delete(s.tokens[serverId], userId)
			s.callOnChange(serverId)
		} else if now.Sub(found.LastUsedAt) > s.slidingTTL {
			delete(s.tokens[serverId], userId)
			s.callOnChange(serverId)
		} else {
			// Renew sliding window
			found.LastUsedAt = now
			return found, nil
		}
	}

	// Generate new token
	token := &Token{
		UserId:     userId,
		Token:      security.RandomString(32),
		CreatedAt:  now,
		LastUsedAt: now,
	}
	s.tokens[serverId][userId] = token
	s.callOnChange(serverId)
	return token, nil
}

// GetAllByServer returns all tokens for a given serverId
func (s *TokenStore) GetAllByServer(serverId string) []*Token {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := []*Token{}
	for _, t := range s.tokens[serverId] {
		result = append(result, t)
	}

	// Add technical token for serverId (Without a token, Outline won’t start)
	result = append(result, &Token{
		UserId:     s.TechnicalKeyName,
		Token:      s.TechnicalKeySecret,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	})

	return result
}

// cleanupLoop periodically removes expired tokens and calls onChange once per server if needed.
func (s *TokenStore) cleanupLoop() {
	ticker := time.NewTicker(s.cleanupIntvl)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		changedServers := make(map[string]struct{})
		for serverId, userMap := range s.tokens {
			initialLen := len(userMap)
			for userId, t := range userMap {
				if now.Sub(t.CreatedAt) > s.absoluteTTL || now.Sub(t.LastUsedAt) > s.slidingTTL {
					delete(userMap, userId)
				}
			}
			if len(userMap) != initialLen {
				changedServers[serverId] = struct{}{}
			}
			if len(userMap) == 0 {
				delete(s.tokens, serverId)
			}
		}
		s.mu.Unlock()
		// Call onChange for each changed serverId (outside lock)
		for serverId := range changedServers {
			s.callOnChange(serverId)
		}
	}
}
