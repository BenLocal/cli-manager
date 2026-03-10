package main

import (
	"log"
	"os"

	"github.com/benlocal/cli-manager/pkg/db"
	"github.com/benlocal/cli-manager/pkg/http"

	hertzServer "github.com/cloudwego/hertz/pkg/app/server"

	_ "github.com/benlocal/cli-manager/pkg/handler"
)

func main() {
	database, err := db.Open(os.Getenv("SQLITE_DSN"))
	if err != nil {
		log.Fatalf("open sqlite: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("close sqlite: %v", err)
		}
	}()

	addr := os.Getenv("APP_ADDR")
	if addr == "" {
		addr = ":8082"
	}

	log.Printf("serving embedded app on %s", addr)
	server := hertzServer.Default(
		hertzServer.WithHostPorts(addr),
		hertzServer.WithMaxRequestBodySize(1*1024*1024*1024), // 1GB
	)

	registry := http.DefaultRegistry
	registryContext := http.NewRegistryContext(database)
	for _, binding := range registry.Bindings() {
		binding(registryContext, server.Engine)
	}

	server.Spin()
}
