package output

import (
	"bytes"
	"testing"
)

func TestJSONFormatterFormatData(t *testing.T) {
	packages := []DecryptedPackage{
		{
			Type:     0,
			TypeName: "Body Mass",
			Data: []DecryptedUnitGroup{
				{
					Units: "kg",
					Records: []DecryptedRecord{
						{Time: "2024-01-14T16:41:33Z", Value: "75.5"},
					},
				},
			},
		},
	}

	var out bytes.Buffer
	if err := (JSONFormatter{}).FormatData(&out, packages); err != nil {
		t.Fatalf("FormatData() error = %v", err)
	}

	want := "" +
		"[\n" +
		"  {\n" +
		"    \"type\": 0,\n" +
		"    \"type_name\": \"Body Mass\",\n" +
		"    \"data\": [\n" +
		"      {\n" +
		"        \"units\": \"kg\",\n" +
		"        \"records\": [\n" +
		"          {\n" +
		"            \"time\": \"2024-01-14T16:41:33Z\",\n" +
		"            \"value\": \"75.5\"\n" +
		"          }\n" +
		"        ]\n" +
		"      }\n" +
		"    ]\n" +
		"  }\n" +
		"]\n"

	if got := out.String(); got != want {
		t.Fatalf("FormatData() output = %q, want %q", got, want)
	}
}

func TestJSONFormatterFormatDataEmpty(t *testing.T) {
	var out bytes.Buffer
	if err := (JSONFormatter{}).FormatData(&out, nil); err != nil {
		t.Fatalf("FormatData() error = %v", err)
	}

	if got, want := out.String(), "[]\n"; got != want {
		t.Fatalf("FormatData() output = %q, want %q", got, want)
	}
}

func TestJSONFormatterFormatAggregatedData(t *testing.T) {
	packages := []AggregatedPackage{
		{
			Type:     9,
			TypeName: "Step count",
			Data: []AggregatedUnitGroup{
				{
					Units: "count",
					Records: []AggregatedRecord{
						{Period: "2024-01-01", Value: 8432},
					},
				},
			},
		},
	}

	var out bytes.Buffer
	if err := (JSONFormatter{}).FormatAggregatedData(&out, packages); err != nil {
		t.Fatalf("FormatAggregatedData() error = %v", err)
	}

	want := "" +
		"[\n" +
		"  {\n" +
		"    \"type\": 9,\n" +
		"    \"type_name\": \"Step count\",\n" +
		"    \"data\": [\n" +
		"      {\n" +
		"        \"units\": \"count\",\n" +
		"        \"records\": [\n" +
		"          {\n" +
		"            \"period\": \"2024-01-01\",\n" +
		"            \"value\": 8432\n" +
		"          }\n" +
		"        ]\n" +
		"      }\n" +
		"    ]\n" +
		"  }\n" +
		"]\n"

	if got := out.String(); got != want {
		t.Fatalf("FormatAggregatedData() output = %q, want %q", got, want)
	}
}

func TestJSONFormatterFormatRawData(t *testing.T) {
	packages := []EncryptedPackage{
		{
			Type:     0,
			TypeName: "Body Mass",
			Data: []EncryptedUnitGroup{
				{
					Units: "kg",
					Records: []EncryptedRecord{
						{
							Time:   "2024-01-14T16:41:33Z",
							Nonce:  "TAJDRM2t8DhP1nDO",
							Cipher: "VBX7VLWQ3UDMS1aH1WHRD8",
						},
					},
				},
			},
		},
	}

	var out bytes.Buffer
	if err := (JSONFormatter{}).FormatRawData(&out, packages); err != nil {
		t.Fatalf("FormatRawData() error = %v", err)
	}

	want := "" +
		"[\n" +
		"  {\n" +
		"    \"type\": 0,\n" +
		"    \"type_name\": \"Body Mass\",\n" +
		"    \"data\": [\n" +
		"      {\n" +
		"        \"units\": \"kg\",\n" +
		"        \"records\": [\n" +
		"          {\n" +
		"            \"time\": \"2024-01-14T16:41:33Z\",\n" +
		"            \"nonce\": \"TAJDRM2t8DhP1nDO\",\n" +
		"            \"cipher\": \"VBX7VLWQ3UDMS1aH1WHRD8\"\n" +
		"          }\n" +
		"        ]\n" +
		"      }\n" +
		"    ]\n" +
		"  }\n" +
		"]\n"

	if got := out.String(); got != want {
		t.Fatalf("FormatRawData() output = %q, want %q", got, want)
	}
}

func TestJSONFormatterFormatTypes(t *testing.T) {
	types := []HealthType{
		{ID: 0, Name: "Body mass", Category: "record", SubCategory: "Body"},
	}

	var out bytes.Buffer
	if err := (JSONFormatter{}).FormatTypes(&out, types); err != nil {
		t.Fatalf("FormatTypes() error = %v", err)
	}

	want := "" +
		"[\n" +
		"  {\n" +
		"    \"id\": 0,\n" +
		"    \"name\": \"Body mass\",\n" +
		"    \"category\": \"record\",\n" +
		"    \"subcategory\": \"Body\"\n" +
		"  }\n" +
		"]\n"

	if got := out.String(); got != want {
		t.Fatalf("FormatTypes() output = %q, want %q", got, want)
	}
}
