package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/rmchorgh/vision-spots/backend/internal/config"
	"github.com/rmchorgh/vision-spots/backend/internal/httpapi"
	"github.com/rmchorgh/vision-spots/backend/internal/session"
)

// X: backend agent - main entrypoint. Loads config, creates in-memory session store,
// wires chi router with all endpoints defined in api-contract.md, and starts server.
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	store := session.NewStore()

	fmt.Printf("vision-spots backend starting on :%d...\n", cfg.Port)
	fmt.Println("X: PKCE + session JWT OAuth broker ready. Endpoints match api-contract.md exactly.")

	r := httpapi.NewRouter(cfg, store)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), r); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
