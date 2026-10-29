package main

import (
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
	mux.HandleFunc("/api/search/provider-price", func(w http.ResponseWriter, r *http.Request) {
		searchProviderPriceHandler(w, r, dbPath)
	})
	mux.HandleFunc("/api/search/billing-code-types", func(w http.ResponseWriter, r *http.Request) {
		listBillingCodeTypesHandler(w, r, dbPath)
	})

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", withCORS(mux)))
}
