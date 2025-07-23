package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	customError "main/internal/error"
)

// SessionManager interface defines session management operations
type SessionManager interface {
	// Session lifecycle
	CreateSession(userID string, tokenData *OAuth2TokenResponse) (*Session, error)
	GetSession(sessionID string) (*Session, error)
	UpdateSession(sessionID string, tokenData *OAuth2TokenResponse) error
	DeleteSession(sessionID string) error
	CleanupExpiredSessions() int

	// Session validation
	ValidateSession(sessionID string) (*Session, error)
	RefreshSession(sessionID string) (*Session, error)
	ExtendSession(sessionID string, duration time.Duration) error

	// Session queries
	GetUserSessions(userID string) ([]*Session, error)
	GetActiveSessionsCount() int
	DeleteUserSessions(userID string) error
}

// Session represents an authenticated user session
type Session struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	AccessToken     string                 `json:"access_token"`
	RefreshToken    string                 `json:"refresh_token"`
	IDToken         string                 `json:"id_token"`
	TokenType       string                 `json:"token_type"`
	ExpiresAt       time.Time              `json:"expires_at"`
	CreatedAt       time.Time              `json:"created_at"`
	LastAccessedAt  time.Time              `json:"last_accessed_at"`
	IPAddress       string                 `json:"ip_address"`
	UserAgent       string                 `json:"user_agent"`
	IsActive        bool                   `json:"is_active"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// SessionConfig holds session management configuration
type SessionConfig struct {
	DefaultTTL       time.Duration
	MaxTTL           time.Duration
	CleanupInterval  time.Duration
	MaxSessionsPerUser int
	SecureCookies    bool
	SameSite         string
}

// sessionManagerImpl implements SessionManager
type sessionManagerImpl struct {
	config   *SessionConfig
	sessions map[string]*Session
	userSessions map[string][]string // userID -> sessionIDs
	mu       sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// NewSessionManager creates a new session manager instance
func NewSessionManager(config *SessionConfig) SessionManager {
	if config == nil {
		config = &SessionConfig{
			DefaultTTL:         24 * time.Hour,
			MaxTTL:             7 * 24 * time.Hour,
			CleanupInterval:    time.Hour,
			MaxSessionsPerUser: 5,
			SecureCookies:      true,
			SameSite:           "Strict",
		}
	}

	sm := &sessionManagerImpl{
		config:       config,
		sessions:     make(map[string]*Session),
		userSessions: make(map[string][]string),
		stopCleanup:  make(chan bool),
	}

	// Start cleanup routine
	sm.startCleanupRoutine()

	return sm
}

// CreateSession creates a new session for a user
func (sm *sessionManagerImpl) CreateSession(userID string, tokenData *OAuth2TokenResponse) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Generate unique session ID
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %v", err)
	}

	// Calculate expiration time
	expiresAt := time.Now().Add(sm.config.DefaultTTL)
	if tokenData.ExpiresIn > 0 {
		tokenExpiry := time.Now().Add(time.Duration(tokenData.ExpiresIn) * time.Second)
		if tokenExpiry.Before(expiresAt) {
			expiresAt = tokenExpiry
		}
	}

	// Create session
	session := &Session{
		ID:             sessionID,
		UserID:         userID,
		AccessToken:    tokenData.AccessToken,
		RefreshToken:   tokenData.RefreshToken,
		IDToken:        tokenData.IDToken,
		TokenType:      tokenData.TokenType,
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now(),
		LastAccessedAt: time.Now(),
		IsActive:       true,
		Metadata:       make(map[string]interface{}),
	}

	// Check session limits per user
	if err := sm.enforceSessionLimits(userID); err != nil {
		return nil, err
	}

	// Store session
	sm.sessions[sessionID] = session
	sm.addUserSession(userID, sessionID)

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *sessionManagerImpl) GetSession(sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, customError.ErrTokenInvalid
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, customError.ErrTokenExpired
	}

	// Check if session is active
	if !session.IsActive {
		return nil, customError.ErrTokenInvalid
	}

	return session, nil
}

// UpdateSession updates session token data
func (sm *sessionManagerImpl) UpdateSession(sessionID string, tokenData *OAuth2TokenResponse) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return customError.ErrTokenInvalid
	}

	// Update token data
	session.AccessToken = tokenData.AccessToken
	if tokenData.RefreshToken != "" {
		session.RefreshToken = tokenData.RefreshToken
	}
	if tokenData.IDToken != "" {
		session.IDToken = tokenData.IDToken
	}
	session.TokenType = tokenData.TokenType
	session.LastAccessedAt = time.Now()

	// Update expiration if provided
	if tokenData.ExpiresIn > 0 {
		newExpiry := time.Now().Add(time.Duration(tokenData.ExpiresIn) * time.Second)
		if newExpiry.Before(session.ExpiresAt.Add(sm.config.MaxTTL)) {
			session.ExpiresAt = newExpiry
		}
	}

	return nil
}

// DeleteSession removes a session
func (sm *sessionManagerImpl) DeleteSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil // Session doesn't exist, consider it deleted
	}

	// Remove from user sessions
	sm.removeUserSession(session.UserID, sessionID)

	// Delete session
	delete(sm.sessions, sessionID)

	return nil
}

// ValidateSession validates and updates last accessed time
func (sm *sessionManagerImpl) ValidateSession(sessionID string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, customError.ErrTokenInvalid
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// Clean up expired session
		sm.removeUserSession(session.UserID, sessionID)
		delete(sm.sessions, sessionID)
		return nil, customError.ErrTokenExpired
	}

	// Check if session is active
	if !session.IsActive {
		return nil, customError.ErrTokenInvalid
	}

	// Update last accessed time
	session.LastAccessedAt = time.Now()

	return session, nil
}

// RefreshSession refreshes a session's tokens
func (sm *sessionManagerImpl) RefreshSession(sessionID string) (*Session, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	if session.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	// This would typically call the OAuth2 service to refresh tokens
	// For now, we'll just update the last accessed time
	sm.mu.Lock()
	session.LastAccessedAt = time.Now()
	sm.mu.Unlock()

	return session, nil
}

// ExtendSession extends a session's expiration time
func (sm *sessionManagerImpl) ExtendSession(sessionID string, duration time.Duration) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return customError.ErrTokenInvalid
	}

	// Calculate new expiration time
	newExpiry := session.ExpiresAt.Add(duration)
	maxExpiry := session.CreatedAt.Add(sm.config.MaxTTL)

	// Don't exceed maximum TTL
	if newExpiry.After(maxExpiry) {
		newExpiry = maxExpiry
	}

	session.ExpiresAt = newExpiry
	session.LastAccessedAt = time.Now()

	return nil
}

// GetUserSessions returns all active sessions for a user
func (sm *sessionManagerImpl) GetUserSessions(userID string) ([]*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessionIDs, exists := sm.userSessions[userID]
	if !exists {
		return []*Session{}, nil
	}

	var sessions []*Session
	for _, sessionID := range sessionIDs {
		if session, exists := sm.sessions[sessionID]; exists && session.IsActive {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// GetActiveSessionsCount returns the number of active sessions
func (sm *sessionManagerImpl) GetActiveSessionsCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	count := 0
	for _, session := range sm.sessions {
		if session.IsActive && time.Now().Before(session.ExpiresAt) {
			count++
		}
	}

	return count
}

// DeleteUserSessions removes all sessions for a user
func (sm *sessionManagerImpl) DeleteUserSessions(userID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessionIDs, exists := sm.userSessions[userID]
	if !exists {
		return nil
	}

	// Delete all user sessions
	for _, sessionID := range sessionIDs {
		delete(sm.sessions, sessionID)
	}

	// Clear user session list
	delete(sm.userSessions, userID)

	return nil
}

// CleanupExpiredSessions removes expired sessions
func (sm *sessionManagerImpl) CleanupExpiredSessions() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	cleanedCount := 0

	// Find expired sessions
	var expiredSessions []string
	for sessionID, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			expiredSessions = append(expiredSessions, sessionID)
			cleanedCount++
		}
	}

	// Remove expired sessions
	for _, sessionID := range expiredSessions {
		session := sm.sessions[sessionID]
		sm.removeUserSession(session.UserID, sessionID)
		delete(sm.sessions, sessionID)
	}

	return cleanedCount
}

// Helper methods

// generateSessionID generates a cryptographically secure session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// enforceSessionLimits ensures user doesn't exceed session limits
func (sm *sessionManagerImpl) enforceSessionLimits(userID string) error {
	sessionIDs, exists := sm.userSessions[userID]
	if !exists {
		return nil
	}

	// Count active sessions
	activeCount := 0
	var activeSessions []string
	for _, sessionID := range sessionIDs {
		if session, exists := sm.sessions[sessionID]; exists && session.IsActive && time.Now().Before(session.ExpiresAt) {
			activeSessions = append(activeSessions, sessionID)
			activeCount++
		}
	}

	// Remove oldest sessions if limit exceeded
	if activeCount >= sm.config.MaxSessionsPerUser {
		// Sort by creation time and remove oldest
	oldestSessionID := activeSessions[0]
	oldestTime := sm.sessions[activeSessions[0]].CreatedAt
		for _, sessionID := range activeSessions[1:] {
			if sm.sessions[sessionID].CreatedAt.Before(oldestTime) {
				oldestSessionID = sessionID
				oldestTime = sm.sessions[sessionID].CreatedAt
			}
		}

		// Remove oldest session
		sm.removeUserSession(userID, oldestSessionID)
		delete(sm.sessions, oldestSessionID)
	}

	return nil
}

// addUserSession adds a session to user's session list
func (sm *sessionManagerImpl) addUserSession(userID, sessionID string) {
	if _, exists := sm.userSessions[userID]; !exists {
		sm.userSessions[userID] = []string{}
	}
	sm.userSessions[userID] = append(sm.userSessions[userID], sessionID)
}

// removeUserSession removes a session from user's session list
func (sm *sessionManagerImpl) removeUserSession(userID, sessionID string) {
	sessionIDs, exists := sm.userSessions[userID]
	if !exists {
		return
	}

	// Find and remove session ID
	for i, id := range sessionIDs {
		if id == sessionID {
			sm.userSessions[userID] = append(sessionIDs[:i], sessionIDs[i+1:]...)
			break
		}
	}

	// Clean up empty user session list
	if len(sm.userSessions[userID]) == 0 {
		delete(sm.userSessions, userID)
	}
}

// startCleanupRoutine starts the background cleanup routine
func (sm *sessionManagerImpl) startCleanupRoutine() {
	sm.cleanupTicker = time.NewTicker(sm.config.CleanupInterval)
	go func() {
		for {
			select {
			case <-sm.cleanupTicker.C:
				sm.CleanupExpiredSessions()
			case <-sm.stopCleanup:
				sm.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// Stop stops the session manager and cleanup routine
func (sm *sessionManagerImpl) Stop() {
	if sm.cleanupTicker != nil {
		sm.stopCleanup <- true
	}
}