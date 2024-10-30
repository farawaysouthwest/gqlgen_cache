package gqlgen_cache

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/99designs/gqlgen/graphql"
	"github.com/farawaysouthwest/gqlgen_cache/model"
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
	store  CacheAdapter
	logger *slog.Logger
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

func NewFieldCache(cap int, ttl time.Duration, logLevel slog.Level, adapter CacheAdapter) FieldCache {

	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	return &fieldCache{
		cap:    cap,
		store:  adapter,
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

		// Unmarshal the cached data into the expected type
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

	var id string
	var err error

	queryContext := graphql.GetOperationContext(ctx)
	fieldContext := graphql.GetFieldContext(ctx)

	if obj != nil {
		id, err = c.findIdField(obj)
		if err != nil {
			c.logger.Debug("id not found", "error", err)

			// If we can't find an ID, we will create a hash based on the object
			b, err := json.Marshal(obj)
			if err != nil {
				c.logger.Debug("failed to marshal object", "error", err)
				return 0, ""
			}

			id = base64.StdEncoding.EncodeToString(b)
		}

	} else {
		id = base64.StdEncoding.EncodeToString([]byte(queryContext.RawQuery))
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
	//
	//if v.Created+int64(v.MaxAge) < time.Now().Unix() {
	//	c.logger.Debug("cache expired", "key", k)
	//	c.release(k)
	//	return nil, false
	//}

	return v.Data, ok
}

func (c *fieldCache) set(k uint64, maxAge int, v interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store.Set(k, model.CacheField{
		Id:      k,
		MaxAge:  maxAge,
		Created: time.Now().Unix(),
		Data:    v,
	})
}

func (c *fieldCache) release(k uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store.Delete(k)
}
