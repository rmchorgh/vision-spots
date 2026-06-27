package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/rmchorgh/vision-spots/backend/internal/config"
	"github.com/rmchorgh/vision-spots/backend/internal/httpapi"
	"github.com/rmchorgh/vision-spots/backend/internal/session"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-health" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "5055"
		}
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/healthz", port))
		if err != nil {
			fmt.Fprintf(os.Stderr, "health check failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "health check failed: status %d\n", resp.StatusCode)
			os.Exit(1)
		}
		os.Exit(0)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	store := session.NewStore()

	fmt.Printf("vision-spots backend starting on :%d...\n", cfg.Port)

	r := httpapi.NewRouter(cfg, store)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), r); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
