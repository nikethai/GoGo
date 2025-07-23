package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	customError "main/internal/error"
)

// TokenCache interface defines token caching operations
type TokenCache interface {
	// Token storage
	StoreToken(userID string, token *CachedToken) error
	GetToken(userID string) (*CachedToken, error)
	DeleteToken(userID string) error
	UpdateToken(userID string, token *CachedToken) error

	// Token validation
	IsTokenValid(userID string) bool
	IsTokenExpired(userID string) bool
	GetTokenTTL(userID string) (time.Duration, error)

	// Cache management
	CleanupExpiredTokens() int
	GetCacheStats() *CacheStats
	ClearCache() error
	GetCacheSize() int

	// Batch operations
	StoreTokens(tokens map[string]*CachedToken) error
	GetTokens(userIDs []string) (map[string]*CachedToken, error)
	DeleteTokens(userIDs []string) error
}

// CachedToken represents a cached token with metadata
type CachedToken struct {
	UserID          string                 `json:"user_id"`
	AccessToken     string                 `json:"access_token"`
	RefreshToken    string                 `json:"refresh_token"`
	IDToken         string                 `json:"id_token"`
	TokenType       string                 `json:"token_type"`
	Scope           string                 `json:"scope"`
	ExpiresAt       time.Time              `json:"expires_at"`
	RefreshExpiresAt time.Time             `json:"refresh_expires_at"`
	CachedAt        time.Time              `json:"cached_at"`
	LastAccessedAt  time.Time              `json:"last_accessed_at"`
	AccessCount     int64                  `json:"access_count"`
	Metadata        map[string]interface{} `json:"metadata"`
	Encrypted       bool                   `json:"encrypted"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalTokens     int           `json:"total_tokens"`
	ActiveTokens    int           `json:"active_tokens"`
	ExpiredTokens   int           `json:"expired_tokens"`
	HitRate         float64       `json:"hit_rate"`
	MissRate        float64       `json:"miss_rate"`
	AverageAge      time.Duration `json:"average_age"`
	OldestToken     time.Time     `json:"oldest_token"`
	NewestToken     time.Time     `json:"newest_token"`
	CacheSize       int64         `json:"cache_size_bytes"`
	LastCleanup     time.Time     `json:"last_cleanup"`
}

// TokenCacheConfig holds token cache configuration
type TokenCacheConfig struct {
	DefaultTTL       time.Duration
	MaxTTL           time.Duration
	CleanupInterval  time.Duration
	MaxCacheSize     int
	EncryptTokens    bool
	CompressionLevel int
	PersistToDisk    bool
	CacheFilePath    string
}

// tokenCacheImpl implements TokenCache
type tokenCacheImpl struct {
	config        *TokenCacheConfig
	tokens        map[string]*CachedToken
	stats         *CacheStats
	mu            sync.RWMutex
	encryptionKey []byte
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
	hits          int64
	misses        int64
}

// NewTokenCache creates a new token cache instance
func NewTokenCache(config *TokenCacheConfig) (TokenCache, error) {
	if config == nil {
		config = &TokenCacheConfig{
			DefaultTTL:       time.Hour,
			MaxTTL:           24 * time.Hour,
			CleanupInterval:  15 * time.Minute,
			MaxCacheSize:     1000,
			EncryptTokens:    true,
			CompressionLevel: 6,
			PersistToDisk:    false,
			CacheFilePath:    "/tmp/token_cache.json",
		}
	}

	tc := &tokenCacheImpl{
		config:      config,
		tokens:      make(map[string]*CachedToken),
		stats:       &CacheStats{},
		stopCleanup: make(chan bool),
	}

	// Initialize encryption key if encryption is enabled
	if config.EncryptTokens {
		key, err := getOrCreateCacheEncryptionKey()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize encryption key: %v", err)
		}
		tc.encryptionKey = key
	}

	// Load cache from disk if persistence is enabled
	if config.PersistToDisk {
		if err := tc.loadFromDisk(); err != nil {
			// Log error but don't fail initialization
			fmt.Printf("Warning: failed to load cache from disk: %v\n", err)
		}
	}

	// Start cleanup routine
	tc.startCleanupRoutine()

	return tc, nil
}

// StoreToken stores a token in the cache
func (tc *tokenCacheImpl) StoreToken(userID string, token *CachedToken) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Set cache metadata
	token.UserID = userID
	token.CachedAt = time.Now()
	token.LastAccessedAt = time.Now()
	token.AccessCount = 0

	// Encrypt token if encryption is enabled
	if tc.config.EncryptTokens {
		encryptedToken, err := tc.encryptToken(token)
		if err != nil {
			return fmt.Errorf("failed to encrypt token: %v", err)
		}
		token = encryptedToken
		token.Encrypted = true
	}

	// Check cache size limits
	if len(tc.tokens) >= tc.config.MaxCacheSize {
		if err := tc.evictOldestToken(); err != nil {
			return fmt.Errorf("failed to evict old token: %v", err)
		}
	}

	// Store token
	tc.tokens[userID] = token

	// Update stats
	tc.updateStats()

	// Persist to disk if enabled
	if tc.config.PersistToDisk {
		go tc.saveToDisk() // Async save
	}

	return nil
}

// GetToken retrieves a token from the cache
func (tc *tokenCacheImpl) GetToken(userID string) (*CachedToken, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	token, exists := tc.tokens[userID]
	if !exists {
		tc.misses++
		return nil, customError.ErrTokenInvalid
	}

	// Check if token is expired
	if time.Now().After(token.ExpiresAt) {
		// Remove expired token
		delete(tc.tokens, userID)
		tc.misses++
		return nil, customError.ErrTokenExpired
	}

	// Decrypt token if encrypted
	if token.Encrypted && tc.config.EncryptTokens {
		decryptedToken, err := tc.decryptToken(token)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt token: %v", err)
		}
		token = decryptedToken
	}

	// Update access metadata
	token.LastAccessedAt = time.Now()
	token.AccessCount++
	tc.hits++

	// Store updated token back (for access tracking)
	if token.Encrypted && tc.config.EncryptTokens {
		encryptedToken, _ := tc.encryptToken(token)
		tc.tokens[userID] = encryptedToken
	} else {
		tc.tokens[userID] = token
	}

	return token, nil
}

// DeleteToken removes a token from the cache
func (tc *tokenCacheImpl) DeleteToken(userID string) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	delete(tc.tokens, userID)
	tc.updateStats()

	// Persist to disk if enabled
	if tc.config.PersistToDisk {
		go tc.saveToDisk() // Async save
	}

	return nil
}

// UpdateToken updates an existing token in the cache
func (tc *tokenCacheImpl) UpdateToken(userID string, token *CachedToken) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	existingToken, exists := tc.tokens[userID]
	if !exists {
		return customError.ErrTokenInvalid
	}

	// Preserve cache metadata
	token.UserID = userID
	token.CachedAt = existingToken.CachedAt
	token.LastAccessedAt = time.Now()
	token.AccessCount = existingToken.AccessCount

	// Encrypt token if encryption is enabled
	if tc.config.EncryptTokens {
		encryptedToken, err := tc.encryptToken(token)
		if err != nil {
			return fmt.Errorf("failed to encrypt token: %v", err)
		}
		token = encryptedToken
		token.Encrypted = true
	}

	// Update token
	tc.tokens[userID] = token

	// Update stats
	tc.updateStats()

	// Persist to disk if enabled
	if tc.config.PersistToDisk {
		go tc.saveToDisk() // Async save
	}

	return nil
}

// IsTokenValid checks if a token exists and is not expired
func (tc *tokenCacheImpl) IsTokenValid(userID string) bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	token, exists := tc.tokens[userID]
	if !exists {
		return false
	}

	return time.Now().Before(token.ExpiresAt)
}

// IsTokenExpired checks if a token is expired
func (tc *tokenCacheImpl) IsTokenExpired(userID string) bool {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	token, exists := tc.tokens[userID]
	if !exists {
		return true
	}

	return time.Now().After(token.ExpiresAt)
}

// GetTokenTTL returns the time until token expiration
func (tc *tokenCacheImpl) GetTokenTTL(userID string) (time.Duration, error) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	token, exists := tc.tokens[userID]
	if !exists {
		return 0, customError.ErrTokenInvalid
	}

	ttl := time.Until(token.ExpiresAt)
	if ttl < 0 {
		return 0, customError.ErrTokenExpired
	}

	return ttl, nil
}

// CleanupExpiredTokens removes expired tokens from the cache
func (tc *tokenCacheImpl) CleanupExpiredTokens() int {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	now := time.Now()
	cleanedCount := 0

	// Find expired tokens
	var expiredUsers []string
	for userID, token := range tc.tokens {
		if now.After(token.ExpiresAt) {
			expiredUsers = append(expiredUsers, userID)
			cleanedCount++
		}
	}

	// Remove expired tokens
	for _, userID := range expiredUsers {
		delete(tc.tokens, userID)
	}

	// Update stats
	tc.updateStats()
	tc.stats.LastCleanup = now

	// Persist to disk if enabled
	if tc.config.PersistToDisk && cleanedCount > 0 {
		go tc.saveToDisk() // Async save
	}

	return cleanedCount
}

// GetCacheStats returns cache statistics
func (tc *tokenCacheImpl) GetCacheStats() *CacheStats {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	// Update stats before returning
	tc.updateStats()

	// Calculate hit rate
	total := tc.hits + tc.misses
	if total > 0 {
		tc.stats.HitRate = float64(tc.hits) / float64(total)
		tc.stats.MissRate = float64(tc.misses) / float64(total)
	}

	return tc.stats
}

// ClearCache removes all tokens from the cache
func (tc *tokenCacheImpl) ClearCache() error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.tokens = make(map[string]*CachedToken)
	tc.hits = 0
	tc.misses = 0
	tc.updateStats()

	// Clear disk cache if enabled
	if tc.config.PersistToDisk {
		go tc.saveToDisk() // Async save
	}

	return nil
}

// GetCacheSize returns the number of tokens in the cache
func (tc *tokenCacheImpl) GetCacheSize() int {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	return len(tc.tokens)
}

// StoreTokens stores multiple tokens in batch
func (tc *tokenCacheImpl) StoreTokens(tokens map[string]*CachedToken) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	for userID, token := range tokens {
		// Set cache metadata
		token.UserID = userID
		token.CachedAt = time.Now()
		token.LastAccessedAt = time.Now()
		token.AccessCount = 0

		// Encrypt token if encryption is enabled
		if tc.config.EncryptTokens {
			encryptedToken, err := tc.encryptToken(token)
			if err != nil {
				return fmt.Errorf("failed to encrypt token for user %s: %v", userID, err)
			}
			token = encryptedToken
			token.Encrypted = true
		}

		// Check cache size limits
		if len(tc.tokens) >= tc.config.MaxCacheSize {
			if err := tc.evictOldestToken(); err != nil {
				return fmt.Errorf("failed to evict old token: %v", err)
			}
		}

		// Store token
		tc.tokens[userID] = token
	}

	// Update stats
	tc.updateStats()

	// Persist to disk if enabled
	if tc.config.PersistToDisk {
		go tc.saveToDisk() // Async save
	}

	return nil
}

// GetTokens retrieves multiple tokens in batch
func (tc *tokenCacheImpl) GetTokens(userIDs []string) (map[string]*CachedToken, error) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	result := make(map[string]*CachedToken)
	now := time.Now()

	for _, userID := range userIDs {
		token, exists := tc.tokens[userID]
		if !exists {
			tc.misses++
			continue
		}

		// Check if token is expired
		if now.After(token.ExpiresAt) {
			// Remove expired token
			delete(tc.tokens, userID)
			tc.misses++
			continue
		}

		// Decrypt token if encrypted
		if token.Encrypted && tc.config.EncryptTokens {
			decryptedToken, err := tc.decryptToken(token)
			if err != nil {
				continue // Skip this token
			}
			token = decryptedToken
		}

		// Update access metadata
		token.LastAccessedAt = now
		token.AccessCount++
		tc.hits++

		result[userID] = token
	}

	return result, nil
}

// DeleteTokens removes multiple tokens in batch
func (tc *tokenCacheImpl) DeleteTokens(userIDs []string) error {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	for _, userID := range userIDs {
		delete(tc.tokens, userID)
	}

	tc.updateStats()

	// Persist to disk if enabled
	if tc.config.PersistToDisk {
		go tc.saveToDisk() // Async save
	}

	return nil
}

// Helper methods

// encryptToken encrypts a cached token
func (tc *tokenCacheImpl) encryptToken(token *CachedToken) (*CachedToken, error) {
	// Create a copy to avoid modifying the original
	encryptedToken := *token

	// Encrypt sensitive fields
	encryptedAccessToken, err := tc.encrypt(token.AccessToken)
	if err != nil {
		return nil, err
	}
	encryptedToken.AccessToken = encryptedAccessToken

	encryptedRefreshToken, err := tc.encrypt(token.RefreshToken)
	if err != nil {
		return nil, err
	}
	encryptedToken.RefreshToken = encryptedRefreshToken

	encryptedIDToken, err := tc.encrypt(token.IDToken)
	if err != nil {
		return nil, err
	}
	encryptedToken.IDToken = encryptedIDToken

	return &encryptedToken, nil
}

// decryptToken decrypts a cached token
func (tc *tokenCacheImpl) decryptToken(encryptedToken *CachedToken) (*CachedToken, error) {
	// Create a copy to avoid modifying the original
	decryptedToken := *encryptedToken

	// Decrypt sensitive fields
	accessToken, err := tc.decrypt(encryptedToken.AccessToken)
	if err != nil {
		return nil, err
	}
	decryptedToken.AccessToken = accessToken

	refreshToken, err := tc.decrypt(encryptedToken.RefreshToken)
	if err != nil {
		return nil, err
	}
	decryptedToken.RefreshToken = refreshToken

	idToken, err := tc.decrypt(encryptedToken.IDToken)
	if err != nil {
		return nil, err
	}
	decryptedToken.IDToken = idToken

	decryptedToken.Encrypted = false

	return &decryptedToken, nil
}

// encrypt encrypts a string using AES-GCM
func (tc *tokenCacheImpl) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(tc.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a string using AES-GCM
func (tc *tokenCacheImpl) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(tc.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// evictOldestToken removes the oldest token from the cache
func (tc *tokenCacheImpl) evictOldestToken() error {
	if len(tc.tokens) == 0 {
		return nil
	}

	// Find oldest token
	var oldestUserID string
	var oldestTime time.Time
	first := true

	for userID, token := range tc.tokens {
		if first || token.CachedAt.Before(oldestTime) {
			oldestUserID = userID
			oldestTime = token.CachedAt
			first = false
		}
	}

	// Remove oldest token
	delete(tc.tokens, oldestUserID)

	return nil
}

// updateStats updates cache statistics
func (tc *tokenCacheImpl) updateStats() {
	now := time.Now()
	tc.stats.TotalTokens = len(tc.tokens)
	tc.stats.ActiveTokens = 0
	tc.stats.ExpiredTokens = 0

	var totalAge time.Duration
	var oldestTime, newestTime time.Time
	first := true

	for _, token := range tc.tokens {
		if now.Before(token.ExpiresAt) {
			tc.stats.ActiveTokens++
		} else {
			tc.stats.ExpiredTokens++
		}

		age := now.Sub(token.CachedAt)
		totalAge += age

		if first || token.CachedAt.Before(oldestTime) {
			oldestTime = token.CachedAt
		}
		if first || token.CachedAt.After(newestTime) {
			newestTime = token.CachedAt
		}
		first = false
	}

	if tc.stats.TotalTokens > 0 {
		tc.stats.AverageAge = totalAge / time.Duration(tc.stats.TotalTokens)
		tc.stats.OldestToken = oldestTime
		tc.stats.NewestToken = newestTime
	}

	// Estimate cache size in bytes (rough approximation)
	tc.stats.CacheSize = int64(tc.stats.TotalTokens * 1024) // Assume ~1KB per token
}

// getOrCreateCacheEncryptionKey gets or creates an encryption key for cache
func getOrCreateCacheEncryptionKey() ([]byte, error) {
	// Try to get key from environment variable
	if keyStr := os.Getenv("TOKEN_CACHE_ENCRYPTION_KEY"); keyStr != "" {
		key, err := base64.StdEncoding.DecodeString(keyStr)
		if err == nil && len(key) == 32 {
			return key, nil
		}
	}

	// Generate a new 32-byte key for AES-256
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	// In production, you should store this key securely
	fmt.Printf("Generated new cache encryption key. Set TOKEN_CACHE_ENCRYPTION_KEY=%s\n", 
		base64.StdEncoding.EncodeToString(key))

	return key, nil
}

// loadFromDisk loads cache from disk
func (tc *tokenCacheImpl) loadFromDisk() error {
	// Implementation would load from tc.config.CacheFilePath
	// For now, this is a placeholder
	return nil
}

// saveToDisk saves cache to disk
func (tc *tokenCacheImpl) saveToDisk() error {
	// Implementation would save to tc.config.CacheFilePath
	// For now, this is a placeholder
	return nil
}

// startCleanupRoutine starts the background cleanup routine
func (tc *tokenCacheImpl) startCleanupRoutine() {
	tc.cleanupTicker = time.NewTicker(tc.config.CleanupInterval)
	go func() {
		for {
			select {
			case <-tc.cleanupTicker.C:
				tc.CleanupExpiredTokens()
			case <-tc.stopCleanup:
				tc.cleanupTicker.Stop()
				return
			}
		}
	}()
}

// Stop stops the token cache and cleanup routine
func (tc *tokenCacheImpl) Stop() {
	if tc.cleanupTicker != nil {
		tc.stopCleanup <- true
	}
}