package main

type NetworkItem struct {
	ID                     int    `json:"id"`
	ReportingEntityID      int    `json:"reporting_entity_id"`
	NegotiationArrangement string `json:"negotiation_arrangement"`
	Name                   string `json:"name"`
	BillingCodeType        string `json:"billing_code_type"`
	BillingCodeTypeVersion string `json:"billing_code_type_version"`
	BillingCode            string `json:"billing_code"`
	Description            string `json:"description"`
}

type ProviderGroup struct {
	ID                  int    `json:"id"`
	ProviderReferenceID int    `json:"provider_reference_id"`
	NPI                 string `json:"npi"`
	TINType             string `json:"tin_type"`
	TINValue            string `json:"tin_value"`
	TINBusinessName     string `json:"tin_business_name"`
}

type ProviderPrice struct {
	ID              int      `json:"id"`
	BillingCode     string   `json:"billing_code"`
	BillingCodeType string   `json:"billing_code_type"`
	PlanIDs         string   `json:"plan_ids"`
	Procedure       string   `json:"procedure"`
	NPI             string   `json:"npi"`
	TINValue        string   `json:"tin_value"`
	TINBusinessName string   `json:"tin_business_name"`
	NegotiatedType  string   `json:"negotiated_type"`
	NegotiatedRate  *float64 `json:"negotiated_rate"`
	BillingClass    string   `json:"billing_class"`
	Setting         string   `json:"setting"`
	ExpirationDate  string   `json:"expiration_date"`
}
