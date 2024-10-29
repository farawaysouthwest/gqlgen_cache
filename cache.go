package gqlgen_cache

import (
	"context"
	"errors"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"log/slog"
	"os"
	"reflect"
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
	Args       map[string]interface{}
	ObjectName string
	parent     interface{}
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

func (c *fieldCache) findIdField(obj interface{}) (string, error) {
	v := reflect.ValueOf(obj)

	// Ensure we have a pointer to a struct
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		if v.Kind() == reflect.String {
			return v.String(), nil
		}

		return "", errors.New("expected a struct or a string")
	}

	// Iterate over the fields of the struct
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		if field.Name == "ID" || field.Name == "Id" || field.Name == "id" {
			return v.Field(i).String(), nil
		}
	}

	return "", errors.New("id field not found")
}

func (c *fieldCache) generateKey(ctx context.Context, obj interface{}) (uint64, string) {

	queryContext := graphql.GetOperationContext(ctx)
	fieldContext := graphql.GetFieldContext(ctx)

	id, err := c.findIdField(obj)
	if err != nil {
		c.logger.Debug("id not found", "error", err)
		return 0, ""
	}

	// Create a struct to hold the relevant data
	data := keyData{
		FieldName:  fieldContext.Field.Name,
		Args:       queryContext.Variables,
		ObjectName: fieldContext.Object,
		parent:     id,
	}

	b := fmt.Sprint(data.ObjectName, ":", data.FieldName, ":", data.Args, ":", data.parent)

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
