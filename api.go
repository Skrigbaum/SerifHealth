package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

const providerPricePageSize = 50

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "SerifHealth API",
		"endpoints": []string{
			"/health",
			"/api/search/billing-code?billing_code=10006",
			"/api/search/billing-code-types",
			"/api/search/provider?npi=1396747861",
			"/api/search/provider?tin_value=11-3471596",
			"/api/search/provider-price?billing_code=10006",
			"/api/search/provider?tin_business_name=Pathologists",
		},
	})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func searchBillingCodeHandler(w http.ResponseWriter, r *http.Request, dbPath string) {
	code := strings.TrimSpace(r.URL.Query().Get("billing_code"))
	billingCodeType := strings.TrimSpace(r.URL.Query().Get("billing_code_type"))

	var items []NetworkItem
	var err error
	if code == "" {
		items, err = queryNetworkItems(dbPath, buildInitialNetworkItemsQuery())
	} else {
		query, args := buildBillingCodeQuery(code, billingCodeType)
		items, err = queryNetworkItems(dbPath, query, args...)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"billing_code":      code,
		"billing_code_type": billingCodeType,
		"count":             len(items),
		"items":             items,
	})
}

func buildInitialNetworkItemsQuery() string {
	return "SELECT id, reporting_entity_id, negotiation_arrangement, name, billing_code_type, billing_code_type_version, billing_code, description FROM network_item ORDER BY id ASC LIMIT 10"
}

func listBillingCodeTypesHandler(w http.ResponseWriter, r *http.Request, dbPath string) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT DISTINCT billing_code_type FROM network_item WHERE billing_code_type IS NOT NULL AND billing_code_type != '' ORDER BY billing_code_type ASC")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	values := make([]string, 0)
	for rows.Next() {
		var codeType string
		if err := rows.Scan(&codeType); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		values = append(values, codeType)
	}
	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"count": len(values), "types": values})
}

func searchProviderHandler(w http.ResponseWriter, r *http.Request, dbPath string) {
	field, value := detectProviderField(r)
	if field == "" || value == "" {
		writeError(w, http.StatusBadRequest, "provide one of: npi, tin_value, tin_business_name, or tin_buisness_name")
		return
	}

	query, args := buildProviderSearchQuery(field, value)
	items, err := queryProviderGroups(dbPath, query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"field": field,
		"value": value,
		"count": len(items),
		"items": items,
	})
}

func searchProviderPriceHandler(w http.ResponseWriter, r *http.Request, dbPath string) {
	values := r.URL.Query()
	billingCode := strings.TrimSpace(values.Get("billing_code"))
	billingCodeType := strings.TrimSpace(values.Get("billing_code_type"))
	planID := strings.TrimSpace(values.Get("plan_id"))
	npi := strings.TrimSpace(values.Get("npi"))
	tinValue := strings.TrimSpace(values.Get("tin_value"))
	providerIdentifier := strings.TrimSpace(values.Get("provider_identifier"))

	if billingCode == "" {
		writeError(w, http.StatusBadRequest, "billing_code is required")
		return
	}
	if providerIdentifier != "" {
		isNPI := len(providerIdentifier) == 10
		for _, digit := range providerIdentifier {
			if digit < '0' || digit > '9' {
				isNPI = false
				break
			}
		}
		if isNPI {
			npi, tinValue = providerIdentifier, ""
		} else {
			npi, tinValue = "", providerIdentifier
		}
	}
	page, err := strconv.Atoi(values.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	sortOrder := strings.ToUpper(values.Get("sort_order"))
	if sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	items, total, err := queryProviderPrices(dbPath, billingCode, billingCodeType, planID, npi, tinValue, page, sortOrder)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"search_type": "provider_price",
		"count":       len(items),
		"total":       total,
		"page":        page,
		"page_size":   providerPricePageSize,
		"sort_order":  sortOrder,
		"items":       items,
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

func queryProviderPrices(dbPath, billingCode, billingCodeType, planID, npi, tinValue string, page int, sortOrder string) ([]ProviderPrice, int, error) {
	from := `
		FROM negotiated_price np
		JOIN network_item ni ON ni.id = np.network_item_id
		CROSS JOIN json_each(np.provider_references) AS rate_provider
		JOIN provider_group pg ON pg.provider_reference_id = CAST(rate_provider.value AS INTEGER)
		WHERE ni.billing_code = ?`
	args := []any{billingCode}

	if billingCodeType != "" {
		from += " AND ni.billing_code_type = ?"
		args = append(args, billingCodeType)
	}
	if planID != "" {
		from += ` AND EXISTS (
			SELECT 1
			FROM reporting_plan_in_network_file rpif
			JOIN reporting_plan rp ON rp.id = rpif.reporting_plan_id
			WHERE rpif.in_network_file_id = ni.in_network_file_id AND rp.plan_id = ?
		)`
		args = append(args, planID)
	}
	if npi != "" {
		from += " AND EXISTS (SELECT 1 FROM json_each(pg.npi) AS provider_npi WHERE CAST(provider_npi.value AS TEXT) = ?)"
		args = append(args, npi)
	}
	if tinValue != "" {
		from += " AND REPLACE(pg.tin_value, '-', '') = ?"
		args = append(args, strings.ReplaceAll(tinValue, "-", ""))
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, 0, err
	}
	defer db.Close()

	var total int
	if err := db.QueryRow("SELECT COUNT(*) "+from, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT np.id, ni.billing_code, COALESCE(ni.billing_code_type, ''),
		       COALESCE((SELECT group_concat(DISTINCT rp.plan_id)
		                 FROM reporting_plan_in_network_file rpif
		                 JOIN reporting_plan rp ON rp.id = rpif.reporting_plan_id
		                 WHERE rpif.in_network_file_id = ni.in_network_file_id
		                   AND (? = '' OR rp.plan_id = ?)), ''),
		       COALESCE(ni.name, ''),
		       COALESCE((SELECT group_concat(value, ', ') FROM json_each(pg.npi)), ''),
		       COALESCE(pg.tin_value, ''), COALESCE(pg.tin_business_name, ''),
		       COALESCE(np.negotiated_type, ''), np.negotiated_rate,
		       COALESCE(np.billing_class, ''), COALESCE(np.setting, ''), COALESCE(np.expiration_date, '') ` + from +
		" ORDER BY np.negotiated_rate IS NULL, np.negotiated_rate " + sortOrder + ", ni.id, pg.tin_business_name, np.id, CAST(rate_provider.value AS INTEGER), pg.id LIMIT ? OFFSET ?"
	queryArgs := append([]any{planID, planID}, args...)
	pageArgs := append(queryArgs, providerPricePageSize, (page-1)*providerPricePageSize)
	rows, err := db.Query(query, pageArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]ProviderPrice, 0)
	for rows.Next() {
		var item ProviderPrice
		var rate sql.NullFloat64
		if err := rows.Scan(
			&item.ID,
			&item.BillingCode,
			&item.BillingCodeType,
			&item.PlanIDs,
			&item.Procedure,
			&item.NPI,
			&item.TINValue,
			&item.TINBusinessName,
			&item.NegotiatedType,
			&rate,
			&item.BillingClass,
			&item.Setting,
			&item.ExpirationDate,
		); err != nil {
			return nil, 0, err
		}
		if rate.Valid {
			item.NegotiatedRate = &rate.Float64
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
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
