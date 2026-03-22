package output

import (
	"bytes"
	"testing"
)

func TestCSVFormatterFormatDataSingleRecord(t *testing.T) {
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
	if err := (CSVFormatter{}).FormatData(&out, packages); err != nil {
		t.Fatalf("FormatData() error = %v", err)
	}

	want := "" +
		"type,type_name,units,time,value\n" +
		"0,Body Mass,kg,2024-01-14T16:41:33Z,75.5\n"

	if got := out.String(); got != want {
		t.Fatalf("FormatData() output = %q, want %q", got, want)
	}
}

func TestCSVFormatterFormatDataMultipleTypesAndUnits(t *testing.T) {
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
				{
					Units: "lb",
					Records: []DecryptedRecord{
						{Time: "2024-01-14T17:00:00Z", Value: "166.4"},
					},
				},
			},
		},
		{
			Type:     9,
			TypeName: "Step count",
			Data: []DecryptedUnitGroup{
				{
					Units: "count",
					Records: []DecryptedRecord{
						{Time: "2024-01-14T12:00:00Z", Value: "3210"},
						{Time: "2024-01-14T18:00:00Z", Value: "5222"},
					},
				},
			},
		},
	}

	var out bytes.Buffer
	if err := (CSVFormatter{}).FormatData(&out, packages); err != nil {
		t.Fatalf("FormatData() error = %v", err)
	}

	want := "" +
		"type,type_name,units,time,value\n" +
		"0,Body Mass,kg,2024-01-14T16:41:33Z,75.5\n" +
		"0,Body Mass,lb,2024-01-14T17:00:00Z,166.4\n" +
		"9,Step count,count,2024-01-14T12:00:00Z,3210\n" +
		"9,Step count,count,2024-01-14T18:00:00Z,5222\n"

	if got := out.String(); got != want {
		t.Fatalf("FormatData() output = %q, want %q", got, want)
	}
}

func TestCSVFormatterFormatDataEmpty(t *testing.T) {
	var out bytes.Buffer
	if err := (CSVFormatter{}).FormatData(&out, nil); err != nil {
		t.Fatalf("FormatData() error = %v", err)
	}

	want := "type,type_name,units,time,value\n"
	if got := out.String(); got != want {
		t.Fatalf("FormatData() output = %q, want %q", got, want)
	}
}

func TestCSVFormatterFormatDataEscapesCommas(t *testing.T) {
	packages := []DecryptedPackage{
		{
			Type:     0,
			TypeName: "Body Mass",
			Data: []DecryptedUnitGroup{
				{
					Units: "kg",
					Records: []DecryptedRecord{
						{Time: "2024-01-14T16:41:33Z", Value: "75,5"},
					},
				},
			},
		},
	}

	var out bytes.Buffer
	if err := (CSVFormatter{}).FormatData(&out, packages); err != nil {
		t.Fatalf("FormatData() error = %v", err)
	}

	want := "" +
		"type,type_name,units,time,value\n" +
		"0,Body Mass,kg,2024-01-14T16:41:33Z,\"75,5\"\n"

	if got := out.String(); got != want {
		t.Fatalf("FormatData() output = %q, want %q", got, want)
	}
}

func TestCSVFormatterFormatAggregatedData(t *testing.T) {
	packages := []AggregatedPackage{
		{
			Type:     9,
			TypeName: "Step count",
			Data: []AggregatedUnitGroup{
				{
					Units: "count",
					Records: []AggregatedRecord{
						{Period: "2024-01-01", Value: 8432},
						{Period: "2024-01-02", Value: 10211},
					},
				},
			},
		},
	}

	var out bytes.Buffer
	if err := (CSVFormatter{}).FormatAggregatedData(&out, packages); err != nil {
		t.Fatalf("FormatAggregatedData() error = %v", err)
	}

	want := "" +
		"type,type_name,units,period,value\n" +
		"9,Step count,count,2024-01-01,8432\n" +
		"9,Step count,count,2024-01-02,10211\n"

	if got := out.String(); got != want {
		t.Fatalf("FormatAggregatedData() output = %q, want %q", got, want)
	}
}

func TestCSVFormatterFormatRawDataUsesJSONAndWritesNote(t *testing.T) {
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

	prevStderr := stderr
	var errOut bytes.Buffer
	stderr = &errOut
	t.Cleanup(func() {
		stderr = prevStderr
	})

	var out bytes.Buffer
	if err := (CSVFormatter{}).FormatRawData(&out, packages); err != nil {
		t.Fatalf("FormatRawData() error = %v", err)
	}

	wantJSON := "" +
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

	if got := out.String(); got != wantJSON {
		t.Fatalf("FormatRawData() output = %q, want %q", got, wantJSON)
	}

	if got, want := errOut.String(), "Note: --raw output is always JSON\n"; got != want {
		t.Fatalf("stderr note = %q, want %q", got, want)
	}
}

func TestCSVFormatterFormatTypes(t *testing.T) {
	types := []HealthType{
		{ID: 0, Name: "Body mass", Category: "record", SubCategory: "Body"},
		{ID: 1, Name: "Body fat percentage", Category: "record", SubCategory: "Body"},
	}

	var out bytes.Buffer
	if err := (CSVFormatter{}).FormatTypes(&out, types); err != nil {
		t.Fatalf("FormatTypes() error = %v", err)
	}

	want := "" +
		"id,name,category,subcategory\n" +
		"0,Body mass,record,Body\n" +
		"1,Body fat percentage,record,Body\n"

	if got := out.String(); got != want {
		t.Fatalf("FormatTypes() output = %q, want %q", got, want)
	}
}

func TestCSVFormatterFormatTypesEmpty(t *testing.T) {
	var out bytes.Buffer
	if err := (CSVFormatter{}).FormatTypes(&out, nil); err != nil {
		t.Fatalf("FormatTypes() error = %v", err)
	}

	want := "id,name,category,subcategory\n"
	if got := out.String(); got != want {
		t.Fatalf("FormatTypes() output = %q, want %q", got, want)
	}
}
