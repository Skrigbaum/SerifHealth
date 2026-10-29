package main

import (
	"strings"
	"testing"
)

func TestBuildProviderSearchQuery(t *testing.T) {
	query, args := buildProviderSearchQuery("tin_value", "11-3471596")
	if query == "" {
		t.Fatal("expected query to be generated")
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	if args[0] != "%11-3471596%" {
		t.Fatalf("expected wildcard search argument, got %v", args[0])
	}
}

func TestBuildBillingCodeQuery(t *testing.T) {
	query, args := buildBillingCodeQuery("10006")
	if query == "" {
		t.Fatal("expected query to be generated")
	}
	if len(args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(args))
	}
	if args[0] != "10006" {
		t.Fatalf("expected exact code, got %v", args[0])
	}
}

func TestBuildInitialNetworkItemsQuery(t *testing.T) {
	query := buildInitialNetworkItemsQuery()
	if !strings.Contains(query, "ORDER BY id ASC LIMIT 10") {
		t.Fatalf("expected the initial query to return the first 10 rows in ID order, got %q", query)
	}
}
