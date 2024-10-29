package gqlgen_cache

import (
	"context"
	"encoding/json"
	"github.com/99designs/gqlgen/graphql"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"time"

	"hash/fnv"
	"sync"
)

type fieldCache struct {
	mu    sync.RWMutex
	cap   int
	store *expirable.LRU[uint64, cacheField]
}

type cacheField struct {
	id      uint64
	maxAge  int
	created int64
	data    interface{}
}

type FieldCache interface {
	Handle(ctx context.Context, obj interface{}, next graphql.Resolver, maxAge *int) (res interface{}, err error)
	Get(uint64) (interface{}, bool)
	Set(uint64, int, interface{})
	GenerateKey(obj interface{}) uint64
	Release(uint64)
}

func NewFieldCache(cap int, ttl time.Duration) FieldCache {

	s := expirable.NewLRU[uint64, cacheField](cap, nil, ttl)

	return &fieldCache{
		cap:   cap,
		store: s,
	}
}

func (c *fieldCache) Handle(ctx context.Context, obj interface{}, next graphql.Resolver, maxAge *int) (res interface{}, err error) {
	if maxAge == nil || obj == nil {
		return next(ctx)
	}

	key := c.GenerateKey(obj)
	if v, ok := c.Get(key); ok {
		return v, nil
	}

	res, err = next(ctx)
	if err != nil {
		return nil, err
	}

	c.Set(key, *maxAge, res)
	return res, nil
}

func (c *fieldCache) GenerateKey(obj interface{}) uint64 {

	if obj == nil {
		return 0
	}

	b, err := json.Marshal(obj)
	if err != nil {
		return 0
	}

	hash := fnv.New64a()

	if _, err := hash.Write(b); err != nil {
		return 0
	}
	return hash.Sum64()
}

func (c *fieldCache) Get(k uint64) (interface{}, bool) {

	c.mu.RLock()
	v, ok := c.store.Get(k)
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if v.created+int64(v.maxAge) < time.Now().Unix() {
		c.Release(k)
		return nil, false
	}

	return v.data, ok
}

func (c *fieldCache) Set(k uint64, maxAge int, v interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store.Add(k, cacheField{
		id:      k,
		maxAge:  maxAge,
		created: time.Now().Unix(),
		data:    v,
	})
}

func (c *fieldCache) Release(k uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store.Remove(k)
}
