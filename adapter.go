package gqlgen_cache

import (
	"github.com/farawaysouthwest/gqlgen_cache/model"
)

type CacheAdapter interface {
	Get(key uint64) (model.CacheField, bool)
	Set(key uint64, value model.CacheField) bool
	Delete(key uint64) bool
}
