package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/rmchorgh/vision-spots/backend/internal/config"
	"github.com/rmchorgh/vision-spots/backend/internal/httpapi"
)

// X: backend agent - initial main.go skeleton with config and router setup
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("vision-spots backend starting on :%d...\n", cfg.Port)

	// TODO: wire up full router with all endpoints from api-contract.md
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), nil); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
