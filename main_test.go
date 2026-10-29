package main

import "testing"

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
