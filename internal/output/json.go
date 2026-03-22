package output

import (
	"encoding/json"
	"io"
)

type JSONFormatter struct{}

type jsonHealthType struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	SubCategory string `json:"subcategory"`
}

func (f JSONFormatter) FormatData(w io.Writer, packages []DecryptedPackage) error {
	return writeJSON(w, nonNilSlice(packages))
}

func (f JSONFormatter) FormatAggregatedData(w io.Writer, packages []AggregatedPackage) error {
	return writeJSON(w, nonNilSlice(packages))
}

func (f JSONFormatter) FormatRawData(w io.Writer, packages []EncryptedPackage) error {
	return writeJSON(w, nonNilSlice(packages))
}

func (f JSONFormatter) FormatTypes(w io.Writer, types []HealthType) error {
	formatted := make([]jsonHealthType, 0, len(types))
	for _, healthType := range types {
		formatted = append(formatted, jsonHealthType{
			ID:          healthType.ID,
			Name:        healthType.Name,
			Category:    healthType.Category,
			SubCategory: healthType.SubCategory,
		})
	}

	return writeJSON(w, nonNilSlice(formatted))
}

func writeJSON(w io.Writer, data any) error {
	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if _, err := w.Write(body); err != nil {
		return err
	}

	_, err = w.Write([]byte("\n"))
	return err
}

func nonNilSlice[T any](values []T) []T {
	if values == nil {
		return []T{}
	}

	return values
}
