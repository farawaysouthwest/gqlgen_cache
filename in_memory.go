package gqlgen_cache

import (
	"github.com/farawaysouthwest/gqlgen_cache/model"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"time"
)

type inMemoryCache struct {
	store *expirable.LRU[uint64, model.CacheField]
}

func NewInMemoryCache(cap int, ttl time.Duration) CacheAdapter {
	store := expirable.NewLRU[uint64, model.CacheField](cap, nil, ttl)
	return &inMemoryCache{store: store}
}

func (c *inMemoryCache) Get(key uint64) (model.CacheField, bool) {
	value, ok := c.store.Get(key)
	return value, ok
}

func (c *inMemoryCache) Set(key uint64, value model.CacheField) bool {
	return c.store.Add(key, value)
}

func (c *inMemoryCache) Delete(key uint64) bool {
	return c.store.Remove(key)
}
