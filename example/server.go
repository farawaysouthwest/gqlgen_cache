package main

import (
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/farawaysouthwest/gqlgen_cache"
	"github.com/farawaysouthwest/gqlgen_cache/example/graph"
	"log"
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

	cache := gqlgen_cache.NewFieldCache(100, time.Minute*1)

	c := graph.Config{Resolvers: &graph.Resolver{}}
	c.Directives.Cache = cache.Handle

	http.Handle("/query", handler.NewDefaultServer(graph.NewExecutableSchema(c)))
	http.Handle("/", playground.Handler("Todo", "/query"))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
