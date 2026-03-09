package main

import (
	"log"
	"net/http"
	"os"

	"github.com/benlocal/cli-manager/pkg/db"
	"github.com/benlocal/cli-manager/pkg/handler"
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

	rootHandler, err := handler.NewRootHandler(database)
	if err != nil {
		log.Fatalf("init app handler: %v", err)
	}

	addr := os.Getenv("APP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("serving embedded app on %s", addr)
	if err := http.ListenAndServe(addr, rootHandler); err != nil {
		log.Fatalf("serve app: %v", err)
	}
}
