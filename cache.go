package gqlgen_cache

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"log/slog"
	"os"
	"time"

	"hash/fnv"
	"sync"
)

type fieldCache struct {
	mu     sync.RWMutex
	cap    int
	store  *expirable.LRU[uint64, cacheField]
	logger *slog.Logger
}

type cacheField struct {
	id      uint64
	maxAge  int
	created int64
	data    interface{}
}

type keyData struct {
	FieldName  string
	ObjectName string
	opHash     string
}

type FieldCache interface {
	Handle(ctx context.Context, obj interface{}, next graphql.Resolver, maxAge *int) (res interface{}, err error)
}

func NewFieldCache(cap int, ttl time.Duration, logLevel slog.Level) FieldCache {

	s := expirable.NewLRU[uint64, cacheField](cap, nil, ttl)

	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	return &fieldCache{
		cap:    cap,
		store:  s,
		logger: l,
	}
}

func (c *fieldCache) Handle(ctx context.Context, obj interface{}, next graphql.Resolver, maxAge *int) (res interface{}, err error) {

	key, stringKey := c.generateKey(ctx, obj)

	// If key is 0, skip caching
	if key == 0 {
		return next(ctx)
	}

	if v, ok := c.get(key); ok {
		c.logger.Debug("cache hit", "key", stringKey, "hash", key)
		return v, nil
	}

	c.logger.Debug("cache miss", "key", stringKey, "hash", key)
	res, err = next(ctx)
	if err != nil {
		return nil, err
	}

	c.set(key, *maxAge, res)
	c.logger.Debug("cache set", "key", stringKey, "hash", key)
	return res, nil
}

func (c *fieldCache) generateKey(ctx context.Context, obj interface{}) (uint64, string) {

	var id string

	queryContext := graphql.GetOperationContext(ctx)
	fieldContext := graphql.GetFieldContext(ctx)

	bv, err := json.Marshal(queryContext.Variables)
	if err != nil {
		c.logger.Debug("failed to marshal variables", "error", err)
		return 0, ""
	}

	if obj != nil {
		b, err := json.Marshal(obj)
		if err != nil {
			c.logger.Debug("failed to marshal object", "error", err)
			return 0, ""
		}
		b = append(b, bv...)

		id = base64.StdEncoding.EncodeToString(b)
	} else {

		b := append([]byte(queryContext.RawQuery), bv...)

		id = base64.StdEncoding.EncodeToString(b)
	}

	// Create a struct to hold the relevant data
	data := keyData{
		ObjectName: fieldContext.Object,
		FieldName:  fieldContext.Field.Name,
		opHash:     id,
	}

	b := fmt.Sprint(data.ObjectName, ":", data.FieldName, ":", data.opHash)

	hash := fnv.New64a()

	if _, err := hash.Write([]byte(b)); err != nil {
		return 0, ""
	}
	return hash.Sum64(), b
}

func (c *fieldCache) get(k uint64) (interface{}, bool) {

	c.mu.RLock()
	v, ok := c.store.Get(k)
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if v.created+int64(v.maxAge) < time.Now().Unix() {
		c.logger.Debug("cache expired", "key", k)
		c.release(k)
		return nil, false
	}

	return v.data, ok
}

func (c *fieldCache) set(k uint64, maxAge int, v interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store.Add(k, cacheField{
		id:      k,
		maxAge:  maxAge,
		created: time.Now().Unix(),
		data:    v,
	})
}

func (c *fieldCache) release(k uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store.Remove(k)
}
