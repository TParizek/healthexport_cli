package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

type CSVFormatter struct{}

func (f CSVFormatter) FormatData(w io.Writer, packages []DecryptedPackage) error {
	writer := csv.NewWriter(w)
	writer.UseCRLF = false

	if err := writer.Write([]string{"type", "type_name", "units", "time", "value"}); err != nil {
		return err
	}

	for _, pkg := range packages {
		for _, group := range pkg.Data {
			for _, record := range group.Records {
				if err := writer.Write([]string{
					strconv.Itoa(pkg.Type),
					pkg.TypeName,
					group.Units,
					record.Time,
					record.Value,
				}); err != nil {
					return err
				}
			}
		}
	}

	writer.Flush()
	return writer.Error()
}

func (f CSVFormatter) FormatAggregatedData(w io.Writer, packages []AggregatedPackage) error {
	writer := csv.NewWriter(w)
	writer.UseCRLF = false

	if err := writer.Write([]string{"type", "type_name", "units", "period", "value"}); err != nil {
		return err
	}

	for _, pkg := range packages {
		for _, group := range pkg.Data {
			for _, record := range group.Records {
				if err := writer.Write([]string{
					strconv.Itoa(pkg.Type),
					pkg.TypeName,
					group.Units,
					record.Period,
					strconv.FormatFloat(record.Value, 'f', -1, 64),
				}); err != nil {
					return err
				}
			}
		}
	}

	writer.Flush()
	return writer.Error()
}

func (f CSVFormatter) FormatRawData(w io.Writer, packages []EncryptedPackage) error {
	if _, err := fmt.Fprintln(stderr, "Note: --raw output is always JSON"); err != nil {
		return err
	}

	return JSONFormatter{}.FormatRawData(w, packages)
}

func (f CSVFormatter) FormatTypes(w io.Writer, types []HealthType) error {
	writer := csv.NewWriter(w)
	writer.UseCRLF = false

	if err := writer.Write([]string{"id", "name", "category", "subcategory"}); err != nil {
		return err
	}

	for _, healthType := range types {
		if err := writer.Write([]string{
			strconv.Itoa(healthType.ID),
			healthType.Name,
			healthType.Category,
			healthType.SubCategory,
		}); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}
