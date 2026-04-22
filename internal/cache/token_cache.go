package cache

import (
	"sync"
	"time"
	"xykcb_server/internal/metrics"
)

type TokenCache struct {
	mu          sync.RWMutex
	entries     map[string]*TokenEntry
	ttl         time.Duration
	maxSize     int
	cleanupDone chan struct{}
}

type TokenEntry struct {
	token   string
	expires time.Time
	created time.Time
}

func NewTokenCache(ttl time.Duration, maxSize int) *TokenCache {
	return &TokenCache{
		entries:     make(map[string]*TokenEntry),
		ttl:         ttl,
		maxSize:     maxSize,
		cleanupDone: make(chan struct{}),
	}
}

func (c *TokenCache) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.cleanup()
			case <-c.cleanupDone:
				return
			}
		}
	}()
}

func (c *TokenCache) StopCleanup() {
	close(c.cleanupDone)
}

func (c *TokenCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return "", false
	}

	if time.Now().After(entry.expires) {
		return "", false
	}

	return entry.token, true
}

func (c *TokenCache) Set(key, token string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	now := time.Now()
	c.entries[key] = &TokenEntry{
		token:   token,
		expires: now.Add(c.ttl),
		created: now,
	}
}

func (c *TokenCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
}

func (c *TokenCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expires) {
			delete(c.entries, key)
		}
	}
}

func (c *TokenCache) evictOldest() {
	if len(c.entries) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range c.entries {
		if first || entry.created.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.created
			first = false
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

func (c *TokenCache) GetAllEntries() map[string]struct{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]struct{}, len(c.entries))
	for key := range c.entries {
		result[key] = struct{}{}
	}
	return result
}

var tokenCache = NewTokenCache(5*time.Minute, 10000)

func GetTokenCache() *TokenCache {
	return tokenCache
}

func GetToken(providerKey, account, password string, loginFunc func(account, password string) (string, error)) (string, error) {
	key := providerKey + ":" + account

	if token, ok := tokenCache.Get(key); ok {
		metrics.RecordTokenCacheHit()
		return token, nil
	}

	metrics.RecordTokenCacheMiss()
	token, err := loginFunc(account, password)
	if err != nil {
		return "", err
	}

	tokenCache.Set(key, token)

	return token, nil
}

func InvalidateToken(providerKey, account string) {
	key := providerKey + ":" + account
	tokenCache.Delete(key)
}

func StartTokenCacheCleanup(interval time.Duration) {
	tokenCache.StartCleanup(interval)
}

func StopTokenCacheCleanup() {
	tokenCache.StopCleanup()
}
