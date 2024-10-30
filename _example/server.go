package main

import (
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/farawaysouthwest/gqlgen_cache"
	"github.com/farawaysouthwest/gqlgen_cache/_example/graph"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
)

const defaultPort = "8080"

type resolvers struct {
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	var adapter gqlgen_cache.CacheAdapter
	if os.Getenv("USE_VALKEY") == "true" {
		adapter = gqlgen_cache.NewValKeyCache(time.Second * 5)
	} else {
		adapter = gqlgen_cache.NewInMemoryCache(100, time.Second*5)
	}

	cache := gqlgen_cache.NewFieldCache(100, time.Second*5, slog.LevelDebug, adapter)

	c := graph.Config{Resolvers: &graph.Resolver{}}
	c.Directives.CacheControl = cache.Handle

	http.Handle("/query", handler.NewDefaultServer(graph.NewExecutableSchema(c)))
	http.Handle("/", playground.Handler("Todo", "/query"))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
