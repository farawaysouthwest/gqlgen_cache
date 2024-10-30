package model

import (
	"github.com/99designs/gqlgen/graphql"
	"github.com/vmihailenco/msgpack/v5"
)

type CacheField struct {
	Id      uint64
	MaxAge  int
	Created int64
	Data    interface{}
	Type    string
}

func (cf *CacheField) MarshalJSON() ([]byte, error) {
	return msgpack.Marshal(cf.Data)
}

func (cf *CacheField) UnmarshalJSON(data []byte) error {

	err := msgpack.Unmarshal(data, &cf.Data)
	if err != nil {
		return err
	}

	v, err := graphql.UnmarshalAny(data)
	if err != nil {
		return err
	}

	cf.Data = v

	return nil
}
