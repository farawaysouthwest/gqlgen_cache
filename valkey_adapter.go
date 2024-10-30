package gqlgen_cache

import (
	"context"
	"github.com/farawaysouthwest/gqlgen_cache/model"
	"github.com/valkey-io/valkey-go"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type valKeyCache struct {
	store valkey.Client
	ttl   time.Duration
}

func NewValKeyCache(ttl time.Duration) CacheAdapter {

	valkeyHost := os.Getenv("VALKEY_HOST")
	if valkeyHost == "" {
		valkeyHost = "127.0.0.1:6379"
	}

	client, err := valkey.NewClient(valkey.ClientOption{InitAddress: []string{valkeyHost}})
	if err != nil {
		slog.Error("error creating valkey client", "error", err)
		panic(err)
	}

	return &valKeyCache{
		store: client,
		ttl:   ttl,
	}
}

func (c *valKeyCache) stringKey(key uint64) string {
	return strconv.FormatUint(key, 10)
}

func (c *valKeyCache) parseError(err error) error {
	if !valkey.IsValkeyNil(err) && err != nil {
		return err
	}
	return nil
}

func (c *valKeyCache) Get(key uint64) (model.CacheField, bool) {
	ctx := context.Background()

	stringKey := c.stringKey(key)

	b := c.store.B().Get().Key(stringKey).Build()

	r := c.store.Do(ctx, b)
	if c.parseError(r.Error()) != nil {
		slog.Error("error getting from valkey", "key", stringKey, "error", r.Error())
		return model.CacheField{}, false
	}

	resBytes, err := r.AsBytes()
	if c.parseError(err) != nil {
		slog.Error("error converting result to bytes", "key", stringKey, "error", err)
		return model.CacheField{}, false
	}

	if resBytes == nil {
		return model.CacheField{}, false
	}

	var destination model.CacheField
	err = destination.UnmarshalJSON(resBytes)
	if err != nil {
		slog.Error("error unmarshalling result", "key", stringKey, "error", err)
		return model.CacheField{}, false
	}

	slog.Debug("data retrieved from valkey", "key", stringKey, "value", destination)
	return destination, true
}

func (c *valKeyCache) Set(key uint64, value model.CacheField) bool {

	ctx := context.Background()

	stringKey := c.stringKey(key)

	valueBytes, err := value.MarshalJSON()
	if err != nil {
		slog.Error("error marshalling value", "key", stringKey, "error", err)
		return false
	}

	b := c.store.B().Set().Key(stringKey).Value(valkey.BinaryString(valueBytes)).ExSeconds(int64(c.ttl.Seconds())).Build()

	r := c.store.Do(ctx, b)
	if r.Error() != nil {
		slog.Error("error setting value in valkey", "key", stringKey, "error", r.Error())
		return false
	}

	return true
}

func (c *valKeyCache) Delete(key uint64) bool {
	ctx := context.Background()

	stringKey := c.stringKey(key)

	b := c.store.B().Del().Key(stringKey).Build()

	r := c.store.Do(ctx, b)
	if r.Error() != nil {
		slog.Error("error deleting value from valkey", "key", stringKey, "error", r.Error())
		return false
	}

	return true
}
