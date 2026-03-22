package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TParizek/healthexport_cli/internal/api"
)

type (
	EncryptedRecord     = api.EncryptedRecord
	EncryptedUnitGroup  = api.EncryptedUnitGroup
	EncryptedPackage    = api.EncryptedPackage
	DecryptedRecord     = api.DecryptedRecord
	DecryptedUnitGroup  = api.DecryptedUnitGroup
	DecryptedPackage    = api.DecryptedPackage
	AggregatedRecord    = api.AggregatedRecord
	AggregatedUnitGroup = api.AggregatedUnitGroup
	AggregatedPackage   = api.AggregatedPackage
	HealthType          = api.HealthType
)

type Formatter interface {
	FormatData(w io.Writer, packages []DecryptedPackage) error
	FormatAggregatedData(w io.Writer, packages []AggregatedPackage) error
	FormatRawData(w io.Writer, packages []EncryptedPackage) error
	FormatTypes(w io.Writer, types []HealthType) error
}

var stderr io.Writer = os.Stderr

func NewFormatter(format string) (Formatter, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "csv":
		return CSVFormatter{}, nil
	case "json":
		return JSONFormatter{}, nil
	default:
		return nil, fmt.Errorf("unsupported output format %q", format)
	}
}
