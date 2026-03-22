package api

import (
	"fmt"
	"strings"
)

type EncryptedRecord struct {
	Time   string `json:"time"`
	Nonce  string `json:"nonce"`
	Cipher string `json:"cipher"`
}

type EncryptedUnitGroup struct {
	Units   string            `json:"units"`
	Records []EncryptedRecord `json:"records"`
}

type EncryptedPackage struct {
	Type     int                  `json:"type"`
	TypeName string               `json:"type_name,omitempty"`
	Data     []EncryptedUnitGroup `json:"data"`
}

type DecryptedRecord struct {
	Time  string `json:"time"`
	Value string `json:"value"`
}

type DecryptedUnitGroup struct {
	Units   string            `json:"units"`
	Records []DecryptedRecord `json:"records"`
}

type DecryptedPackage struct {
	Type     int                  `json:"type"`
	TypeName string               `json:"type_name"`
	Data     []DecryptedUnitGroup `json:"data"`
}

type AggregatedRecord struct {
	Period string  `json:"period"`
	Value  float64 `json:"value"`
}

type AggregatedUnitGroup struct {
	Units   string             `json:"units"`
	Records []AggregatedRecord `json:"records"`
}

type AggregatedPackage struct {
	Type     int                   `json:"type"`
	TypeName string                `json:"type_name"`
	Data     []AggregatedUnitGroup `json:"data"`
}

type HealthType struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	SubCategory string `json:"subCategory"`
}

type HealthTypeSection struct {
	Name  string       `json:"name"`
	Types []HealthType `json:"types"`
}

type HealthTypesResponse struct {
	Aggregated []HealthTypeSection `json:"aggregated"`
	Record     []HealthTypeSection `json:"record"`
	Workout    []HealthTypeSection `json:"workout"`
}

type APIError struct {
	StatusCode int
	Body       string
	Endpoint   string
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}

	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("api request to %s failed with status %d", e.Endpoint, e.StatusCode)
	}

	return fmt.Sprintf("api request to %s failed with status %d: %s", e.Endpoint, e.StatusCode, body)
}

const defaultUserAgentVersion = "dev"

var userAgentVersion = defaultUserAgentVersion

func SetUserAgentVersion(version string) {
	if strings.TrimSpace(version) == "" {
		userAgentVersion = defaultUserAgentVersion
		return
	}

	userAgentVersion = version
}
