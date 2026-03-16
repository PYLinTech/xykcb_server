package cache

import (
	"sync"
	"time"
)

type TokenCache struct {
	mu      sync.RWMutex
	entries map[string]*TokenEntry
	ttl     time.Duration
	maxSize int
}

type TokenEntry struct {
	token   string
	expires time.Time
	created time.Time
}

func NewTokenCache(ttl time.Duration, maxSize int) *TokenCache {
	return &TokenCache{
		entries: make(map[string]*TokenEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
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

	// 检查容量，满了就淘汰最旧的
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

	// 找出最旧的 entry
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

// GetAllEntries 用于调试，返回所有缓存条目
func (c *TokenCache) GetAllEntries() map[string]struct{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]struct{}, len(c.entries))
	for key := range c.entries {
		result[key] = struct{}{}
	}
	return result
}

// tokenCache 全局实例
var tokenCache = NewTokenCache(5*time.Minute, 10000)

// GetToken 获取 token，先从缓存获取，若不存在或过期则登录获取
func GetToken(providerKey, account, password string, loginFunc func(account, password string) (string, error)) (string, error) {
	key := providerKey + ":" + account

	// 先尝试从缓存获取
	if token, ok := tokenCache.Get(key); ok {
		return token, nil
	}

	// 缓存不存在，调用登录函数获取
	token, err := loginFunc(account, password)
	if err != nil {
		return "", err
	}

	// 存入缓存
	tokenCache.Set(key, token)

	return token, nil
}

// InvalidateToken 使指定 key 的 token 失效
func InvalidateToken(providerKey, account string) {
	key := providerKey + ":" + account
	tokenCache.Delete(key)
}
