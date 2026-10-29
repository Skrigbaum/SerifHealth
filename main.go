package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"
)

const defaultDBPath = "serifhealth.db"

func main() {
	dbPath := defaultDBPath

	if _, err := os.Stat(dbPath); err != nil {
		log.Fatalf("database not found at %s: %v", dbPath, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/search/billing-code", func(w http.ResponseWriter, r *http.Request) {
		searchBillingCodeHandler(w, r, dbPath)
	})
	mux.HandleFunc("/api/search/provider", func(w http.ResponseWriter, r *http.Request) {
		searchProviderHandler(w, r, dbPath)
	})
	mux.HandleFunc("/api/search/billing-code-types", func(w http.ResponseWriter, r *http.Request) {
		listBillingCodeTypesHandler(w, r, dbPath)
	})

	fmt.Println("Listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}


