package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"service": "SerifHealth API",
		"endpoints": []string{
			"/health",
			"/api/search/billing-code?billing_code=10006",
			"/api/search/billing-code-types",
			"/api/search/provider?npi=1396747861",
			"/api/search/provider?tin_value=11-3471596",
			"/api/search/provider?tin_business_name=Pathologists",
		},
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func searchBillingCodeHandler(w http.ResponseWriter, r *http.Request, dbPath string) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	code := strings.TrimSpace(r.URL.Query().Get("billing_code"))
	billingCodeType := strings.TrimSpace(r.URL.Query().Get("billing_code_type"))
	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "billing_code query parameter is required"})
		return
	}

	query, args := buildBillingCodeQuery(code, billingCodeType)
	items, err := queryNetworkItems(dbPath, query, args...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"billing_code":      code,
		"billing_code_type": billingCodeType,
		"count":            len(items),
		"items":            items,
	})
}

func listBillingCodeTypesHandler(w http.ResponseWriter, r *http.Request, dbPath string) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT DISTINCT billing_code_type FROM network_item WHERE billing_code_type IS NOT NULL AND billing_code_type != '' ORDER BY billing_code_type ASC")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	values := make([]string, 0)
	for rows.Next() {
		var codeType string
		if err := rows.Scan(&codeType); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		values = append(values, codeType)
	}
	json.NewEncoder(w).Encode(map[string]any{"count": len(values), "types": values})
}

func searchProviderHandler(w http.ResponseWriter, r *http.Request, dbPath string) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	field, value := detectProviderField(r)
	if field == "" || value == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "provide one of: npi, tin_value, tin_business_name, or tin_buisness_name"})
		return
	}

	query, args := buildProviderSearchQuery(field, value)
	items, err := queryProviderGroups(dbPath, query, args...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"field": field,
		"value": value,
		"count": len(items),
		"items": items,
	})
}

func detectProviderField(r *http.Request) (string, string) {
	vals := r.URL.Query()
	for _, field := range []string{"npi", "tin_value", "tin_business_name", "tin_buisness_name"} {
		if v := strings.TrimSpace(vals.Get(field)); v != "" {
			return field, v
		}
	}
	if field := strings.TrimSpace(vals.Get("field")); field != "" {
		if v := strings.TrimSpace(vals.Get("value")); v != "" {
			return field, v
		}
	}
	return "", ""
}

func buildBillingCodeQuery(code string, billingCodeType ...string) (string, []any) {
	query := "SELECT id, reporting_entity_id, negotiation_arrangement, name, billing_code_type, billing_code_type_version, billing_code, description FROM network_item WHERE billing_code = ?"
	args := []any{code}
	if len(billingCodeType) > 0 && strings.TrimSpace(billingCodeType[0]) != "" {
		query += " AND billing_code_type = ?"
		args = append(args, billingCodeType[0])
	}
	query += " ORDER BY name ASC"
	return query, args
}

func buildProviderSearchQuery(field, value string) (string, []any) {
	field = normalizeProviderField(field)
	search := "%" + strings.TrimSpace(value) + "%"

	switch field {
	case "npi":
		return "SELECT id, provider_reference_id, npi, tin_type, tin_value, tin_business_name FROM provider_group WHERE LOWER(CAST(npi AS TEXT)) LIKE LOWER(?) ORDER BY tin_business_name ASC", []any{search}
	case "tin_value":
		return "SELECT id, provider_reference_id, npi, tin_type, tin_value, tin_business_name FROM provider_group WHERE LOWER(tin_value) LIKE LOWER(?) ORDER BY tin_business_name ASC", []any{search}
	case "tin_business_name":
		return "SELECT id, provider_reference_id, npi, tin_type, tin_value, tin_business_name FROM provider_group WHERE LOWER(tin_business_name) LIKE LOWER(?) ORDER BY tin_business_name ASC", []any{search}
	default:
		return "SELECT id, provider_reference_id, npi, tin_type, tin_value, tin_business_name FROM provider_group WHERE LOWER(CAST(npi AS TEXT)) LIKE LOWER(?) OR LOWER(tin_value) LIKE LOWER(?) OR LOWER(tin_business_name) LIKE LOWER(?) ORDER BY tin_business_name ASC", []any{search, search, search}
	}
}

func normalizeProviderField(field string) string {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "tin_buisness_name":
		return "tin_business_name"
	case "tinbusinessname":
		return "tin_business_name"
	case "tin_value", "tinvalue":
		return "tin_value"
	case "npi":
		return "npi"
	default:
		return "tin_business_name"
	}
}

func queryNetworkItems(dbPath, query string, args ...any) ([]NetworkItem, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]NetworkItem, 0)
	for rows.Next() {
		var item NetworkItem
		var reportingEntityID sql.NullInt64
		var negotiationArrangement sql.NullString
		var name sql.NullString
		var billingCodeType sql.NullString
		var billingCodeTypeVersion sql.NullString
		var billingCode sql.NullString
		var description sql.NullString

		if err := rows.Scan(&item.ID, &reportingEntityID, &negotiationArrangement, &name, &billingCodeType, &billingCodeTypeVersion, &billingCode, &description); err != nil {
			return nil, err
		}

		item.ReportingEntityID = nullInt64ToInt(reportingEntityID)
		item.NegotiationArrangement = nullString(negotiationArrangement)
		item.Name = nullString(name)
		item.BillingCodeType = nullString(billingCodeType)
		item.BillingCodeTypeVersion = nullString(billingCodeTypeVersion)
		item.BillingCode = nullString(billingCode)
		item.Description = nullString(description)
		items = append(items, item)
	}
	return items, rows.Err()
}

func queryProviderGroups(dbPath, query string, args ...any) ([]ProviderGroup, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ProviderGroup, 0)
	for rows.Next() {
		var item ProviderGroup
		var providerReferenceID sql.NullInt64
		var npi sql.NullString
		var tinType sql.NullString
		var tinValue sql.NullString
		var tinBusinessName sql.NullString

		if err := rows.Scan(&item.ID, &providerReferenceID, &npi, &tinType, &tinValue, &tinBusinessName); err != nil {
			return nil, err
		}

		item.ProviderReferenceID = nullInt64ToInt(providerReferenceID)
		item.NPI = nullString(npi)
		item.TINType = nullString(tinType)
		item.TINValue = nullString(tinValue)
		item.TINBusinessName = nullString(tinBusinessName)
		items = append(items, item)
	}
	return items, rows.Err()
}

func nullString(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

func nullInt64ToInt(v sql.NullInt64) int {
	if v.Valid {
		return int(v.Int64)
	}
	return 0
}
