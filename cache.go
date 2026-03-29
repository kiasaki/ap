package ap

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"
)

var ErrCacheNotFound = errors.New("Cache key not found")

type Cache struct {
	mu           sync.RWMutex
	itemValues   map[string][]byte
	itemExpiries map[string]time.Time
	refreshing   map[string]bool
}

func NewCache() *Cache {
	cache := &Cache{}
	cache.itemValues = map[string][]byte{}
	cache.itemExpiries = map[string]time.Time{}
	cache.refreshing = map[string]bool{}
	return cache
}

func (c *Cache) get(value interface{}, key string, allowExpired bool) (bool, error) {
	c.mu.RLock()
	expiry, hasExpiry := c.itemExpiries[key]
	payload, ok := c.itemValues[key]
	c.mu.RUnlock()
	if !ok {
		return false, ErrCacheNotFound
	}
	expired := hasExpiry && !expiry.IsZero() && time.Now().After(expiry)
	if expired && !allowExpired {
		c.mu.Lock()
		delete(c.itemValues, key)
		delete(c.itemExpiries, key)
		c.mu.Unlock()
		return true, ErrCacheNotFound
	}
	return expired, json.Unmarshal(payload, value)
}

func (c *Cache) Get(value interface{}, key string) error {
	_, err := c.get(value, key, false)
	return err
}

func (c *Cache) GetStale(value interface{}, key string) (bool, error) {
	return c.get(value, key, true)
}

func (c *Cache) Set(key string, value interface{}, expiresIn time.Duration) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.itemValues[key] = payload
	if expiresIn > 0 {
		c.itemExpiries[key] = time.Now().Add(expiresIn)
	} else {
		c.itemExpiries[key] = time.Time{}
	}
	return nil
}

func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.itemValues, key)
	delete(c.itemExpiries, key)
	delete(c.refreshing, key)
}

func (c *Cache) Ensure(value interface{}, key string, expiry time.Duration, getter func() (interface{}, error)) error {
	err := c.Get(value, key)
	if err == ErrCacheNotFound {
		v, err := getter()
		if err != nil {
			return err
		}
		c.Set(key, v, expiry)
		return c.Get(value, key)
	}
	return err
}

func (c *Cache) EnsureStale(value interface{}, key string, expiry time.Duration, getter func() (interface{}, error)) error {
	expired, err := c.GetStale(value, key)
	if err == ErrCacheNotFound {
		v, err := getter()
		if err != nil {
			return err
		}
		if err := c.Set(key, v, expiry); err != nil {
			return err
		}
		return c.Get(value, key)
	}
	if err != nil {
		return err
	}
	if expired && c.startRefresh(key) {
		go func() {
			defer c.finishRefresh(key)
			v, err := getter()
			if err != nil {
				log.Printf("cache refresh failed for %s: %v", key, err)
				return
			}
			if err := c.Set(key, v, expiry); err != nil {
				log.Printf("cache set failed for %s: %v", key, err)
			}
		}()
	}
	return nil
}

func (c *Cache) startRefresh(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.refreshing[key] {
		return false
	}
	c.refreshing[key] = true
	return true
}

func (c *Cache) finishRefresh(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.refreshing, key)
}
