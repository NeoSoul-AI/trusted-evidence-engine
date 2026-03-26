package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/evoevo/trusted-evidence-engine/internal/engine"
	"github.com/evoevo/trusted-evidence-engine/internal/httpapi"
	"github.com/evoevo/trusted-evidence-engine/internal/providers"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	handler := httpapi.NewHandler(engine.New(providers.DefaultProviders()))
	log.Printf("trusted-evidence-engine listening on %s", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatal(err)
	}
}
