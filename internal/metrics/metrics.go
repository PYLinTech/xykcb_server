package metrics

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	mu               sync.RWMutex
	accessCount      int64
	userCounts       map[string]int64
	tokenCacheHits   int64
	tokenCacheMisses int64
}

var globalMetrics = &Metrics{
	userCounts: make(map[string]int64),
}

func RecordAccess(school string) {
	atomic.AddInt64(&globalMetrics.accessCount, 1)

	globalMetrics.mu.Lock()
	globalMetrics.userCounts[school]++
	globalMetrics.mu.Unlock()
}

func RecordTokenCacheHit() {
	atomic.AddInt64(&globalMetrics.tokenCacheHits, 1)
}

func RecordTokenCacheMiss() {
	atomic.AddInt64(&globalMetrics.tokenCacheMisses, 1)
}

func GetMetrics() (accessCount int64, userCount int, hitRate float64) {
	accessCount = atomic.LoadInt64(&globalMetrics.accessCount)

	globalMetrics.mu.RLock()
	userCount = len(globalMetrics.userCounts)
	globalMetrics.mu.RUnlock()

	hits := atomic.LoadInt64(&globalMetrics.tokenCacheHits)
	misses := atomic.LoadInt64(&globalMetrics.tokenCacheMisses)
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return
}

func Reset() {
	atomic.StoreInt64(&globalMetrics.accessCount, 0)
	atomic.StoreInt64(&globalMetrics.tokenCacheHits, 0)
	atomic.StoreInt64(&globalMetrics.tokenCacheMisses, 0)
	globalMetrics.mu.Lock()
	for k := range globalMetrics.userCounts {
		delete(globalMetrics.userCounts, k)
	}
	globalMetrics.mu.Unlock()
}

func StartReporter(interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			access, users, _ := GetMetrics()

			hits := atomic.LoadInt64(&globalMetrics.tokenCacheHits)
			misses := atomic.LoadInt64(&globalMetrics.tokenCacheMisses)
			total := hits + misses
			hitRate := 0.0
			if total > 0 {
				hitRate = float64(hits) / float64(total) * 100
			}

			fmt.Printf("[指标] 访问量: %d | 用户数: %d | Token缓存命中率: %.1f%%\n", access, users, hitRate)

			Reset()
		case <-stopCh:
			return
		}
	}
}
