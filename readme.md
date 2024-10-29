# Field-Level Cache Middleware

This project provides a field-level cache middleware for a GraphQL server using Go. The middleware caches the results of GraphQL field resolvers to improve performance by avoiding redundant computations.

## Features

- **Field-Level Caching**: Cache results of individual GraphQL field resolvers.
- **Configurable Cache Capacity**: Set the maximum number of cache entries.
- **Expirable Entries**: Cache entries can have a time-to-live (TTL) to automatically expire old entries.

## Installation

To use this middleware, you need to have Go installed. You can install the required dependencies using `go mod`:

```sh
go mod tidy
```

## Usage

### Define the Cache Structure

The cache is implemented using a thread-safe map with a mutex for synchronization.

### Implement the Cache Middleware

The middleware intercepts field resolver calls, checks the cache, and either returns the cached result or calls the resolver and stores the result in the cache.

### Integrate the Cache into Your GraphQL Server

You need to integrate the cache middleware into your GraphQL server configuration.

## Example

Here is an example of how to set up and use the field-level cache middleware:

```go
package main

import (
	"context"
	"github.com/99designs/gqlgen/graphql"
	"github.com/farawaysouthwest/gqlgen_cache"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	cache := gqlgen_cache.NewFieldCache(100, 10*time.Minute)
	c := gqlgen_cache.Config{Resolvers: &gqlgen_cache.Resolver{}}
	c.Directives.Cache = cache.Handle

	http.Handle("/query", handler.NewDefaultServer(gqlgen_cache.NewExecutableSchema(c)))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

## API

### `NewFieldCache(cap int, ttl time.Duration) FieldCache`

Creates a new field cache with the specified capacity and TTL.

### `Handle(ctx context.Context, obj interface{}, next graphql.Resolver, maxAge *int) (res interface{}, err error)`

Middleware function to handle caching logic.

### `GenerateKey(obj interface{}) uint64`

Generates a unique key for the given object.

### `Get(k uint64) (interface{}, bool)`

Retrieves a cached value by key.

### `Set(k uint64, maxAge int, v interface{})`

Stores a value in the cache with the specified key and max age.

### `Release(k uint64)`

Removes a value from the cache by key.

## License

This project is licensed under the MIT License.