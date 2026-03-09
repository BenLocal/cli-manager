package main

import (
	"log"
	"net/http"
	"os"

	"github.com/benlocal/cli-manager/pkg/handler"
)

func main() {
	rootHandler, err := handler.NewRootHandler()
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
