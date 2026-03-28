package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/aggregator"
	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/typemap"
	"github.com/spf13/cobra"
)

func TestParseDateDateOnly(t *testing.T) {
	got, err := parseDate("2024-01-15")
	if err != nil {
		t.Fatalf("parseDate() error = %v", err)
	}

	if got != "2024-01-15T00:00:00Z" {
		t.Fatalf("parseDate() = %q, want %q", got, "2024-01-15T00:00:00Z")
	}
}

func TestParseDateRFC3339(t *testing.T) {
	got, err := parseDate("2024-01-15T10:30:00Z")
	if err != nil {
		t.Fatalf("parseDate() error = %v", err)
	}

	if got != "2024-01-15T10:30:00Z" {
		t.Fatalf("parseDate() = %q, want %q", got, "2024-01-15T10:30:00Z")
	}
}

func TestParseDateInvalid(t *testing.T) {
	if _, err := parseDate("not-a-date"); err == nil {
		t.Fatal("parseDate() error = nil, want error")
	}
}

func TestParseDateEmpty(t *testing.T) {
	if _, err := parseDate(""); err == nil {
		t.Fatal("parseDate() error = nil, want error")
	}
}

func TestResolveTypesNumericWithoutAggregateSkipsTypeLookup(t *testing.T) {
	gotIDs, gotNames, err := resolveTypes(nil, []string{"9", "9", "52"}, nil)
	if err != nil {
		t.Fatalf("resolveTypes() error = %v", err)
	}

	if got, want := intsToString(gotIDs), "9,52"; got != want {
		t.Fatalf("type IDs = %q, want %q", got, want)
	}

	if gotNames != nil {
		t.Fatalf("typeNames = %#v, want nil", gotNames)
	}
}

func TestResolveTypesAggregateFiltersAmbiguousNameToAggregatableMatch(t *testing.T) {
	client, closeServer := newHealthTypesClient(t, testResolveTypesResponse())
	t.Cleanup(closeServer)

	period := aggregator.PeriodDay
	gotIDs, gotNames, err := resolveTypes(client, []string{"heart_rate"}, &period)
	if err != nil {
		t.Fatalf("resolveTypes() error = %v", err)
	}

	if got, want := intsToString(gotIDs), "6"; got != want {
		t.Fatalf("type IDs = %q, want %q", got, want)
	}

	if got, want := gotNames[6], "Heart rate"; got != want {
		t.Fatalf("typeNames[6] = %q, want %q", got, want)
	}
}

func TestResolveTypesAggregateRejectsNonAggregatableNumericType(t *testing.T) {
	client, closeServer := newHealthTypesClient(t, testResolveTypesResponse())
	t.Cleanup(closeServer)

	period := aggregator.PeriodDay
	_, _, err := resolveTypes(client, []string{"0"}, &period)
	if !errors.Is(err, aggregator.ErrNotAggregatable) {
		t.Fatalf("resolveTypes() error = %v, want ErrNotAggregatable", err)
	}
}

func TestRunDataRawNumericSkipsTypeLookup(t *testing.T) {
	restore := snapshotDataState()
	t.Cleanup(restore)
	setCmdConfigHome(t)

	healthTypesCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthdata/encrypted":
			writeJSONResponse(t, w, []api.EncryptedPackage{
				{
					Type: 9,
					Data: []api.EncryptedUnitGroup{
						{
							Units: "count",
							Records: []api.EncryptedRecord{
								{
									Time:   "2024-01-14T12:00:00Z",
									Nonce:  "AQIDBAUGBwgJCgsM",
									Cipher: "lmaYhg==",
								},
							},
						},
					},
				},
			})
		case "/healthtypes":
			healthTypesCalled = true
			http.Error(w, "unexpected type lookup", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	accountKey = "0123456789abcdef0123456789abcdef"
	apiURL = server.URL
	dataTypes = []string{"9"}
	dataFrom = "2024-01-14"
	dataTo = "2024-01-15"
	dataFormat = ""
	dataRaw = true
	dataAggregate = ""

	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	if err := runData(cmd, nil); err != nil {
		t.Fatalf("runData() error = %v", err)
	}

	if healthTypesCalled {
		t.Fatal("health type lookup was called, want skipped for raw numeric input")
	}

	output := stdout.String()
	if strings.Contains(output, "\"type_name\"") {
		t.Fatalf("raw output = %q, want no type_name for numeric raw lookup", output)
	}

	if !strings.Contains(output, "\"cipher\": \"lmaYhg==\"") {
		t.Fatalf("raw output = %q, want encrypted payload", output)
	}
}

func TestRunDataDecryptsAndFormatsJSONWithFetchedTypeNames(t *testing.T) {
	restore := snapshotDataState()
	t.Cleanup(restore)
	setCmdConfigHome(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthtypes":
			writeJSONResponse(t, w, testResolveTypesResponse())
		case "/healthdata/encrypted":
			writeJSONResponse(t, w, []api.EncryptedPackage{
				{
					Type: 9,
					Data: []api.EncryptedUnitGroup{
						{
							Units: "count",
							Records: []api.EncryptedRecord{
								{
									Time:   "2024-01-14T18:00:00Z",
									Nonce:  "DAsKCQgHBgUEAwIB",
									Cipher: "wEQM8w==",
								},
							},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	accountKey = "0123456789abcdef0123456789abcdef"
	apiURL = server.URL
	dataTypes = []string{"9"}
	dataFrom = "2024-01-14"
	dataTo = "2024-01-15"
	dataFormat = "json"
	dataRaw = false
	dataAggregate = ""

	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	if err := runData(cmd, nil); err != nil {
		t.Fatalf("runData() error = %v", err)
	}

	output := stdout.String()
	assertContains(t, output, "\"type_name\": \"Step count\"")
	assertContains(t, output, "\"value\": \"8432\"")
}

func TestRunDataAggregatesResolvedNamedType(t *testing.T) {
	restore := snapshotDataState()
	t.Cleanup(restore)
	setCmdConfigHome(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthtypes":
			writeJSONResponse(t, w, testResolveTypesResponse())
		case "/healthdata/encrypted":
			writeJSONResponse(t, w, []api.EncryptedPackage{
				{
					Type: 9,
					Data: []api.EncryptedUnitGroup{
						{
							Units: "count",
							Records: []api.EncryptedRecord{
								{
									Time:   "2024-01-14T12:00:00Z",
									Nonce:  "AQIDBAUGBwgJCgsM",
									Cipher: "lmaYhg==",
								},
								{
									Time:   "2024-01-14T18:00:00Z",
									Nonce:  "DAsKCQgHBgUEAwIB",
									Cipher: "wEQM8w==",
								},
							},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	accountKey = "0123456789abcdef0123456789abcdef"
	apiURL = server.URL
	dataTypes = []string{"step_count"}
	dataFrom = "2024-01-14"
	dataTo = "2024-01-15"
	dataFormat = "json"
	dataRaw = false
	dataAggregate = "day"

	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	if err := runData(cmd, nil); err != nil {
		t.Fatalf("runData() error = %v", err)
	}

	output := stdout.String()
	assertContains(t, output, "\"period\": \"2024-01-14\"")
	assertContains(t, output, "\"value\": 8508")
	assertContains(t, output, "\"type_name\": \"Step count\"")
}

func TestRunDataReturnsExitCode4ForUnknownType(t *testing.T) {
	restore := snapshotDataState()
	t.Cleanup(restore)
	setCmdConfigHome(t)

	client, closeServer := newHealthTypesClient(t, testResolveTypesResponse())
	t.Cleanup(closeServer)

	accountKey = "0123456789abcdef0123456789abcdef"
	apiURL = client.BaseURL
	dataTypes = []string{"unknown_type"}
	dataFrom = "2024-01-14"
	dataTo = "2024-01-15"
	dataFormat = "json"
	dataRaw = false
	dataAggregate = ""

	err := runData(&cobra.Command{}, nil)
	if !errors.Is(err, typemap.ErrUnknownType) {
		t.Fatalf("runData() error = %v, want unknown type error", err)
	}

	if got := exitCodeForError(err); got != 4 {
		t.Fatalf("exitCodeForError() = %d, want 4", got)
	}
}

func newHealthTypesClient(t *testing.T, response *api.HealthTypesResponse) (*api.Client, func()) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthtypes":
			writeJSONResponse(t, w, response)
		default:
			http.NotFound(w, r)
		}
	}))

	return api.NewClient(server.URL), server.Close
}

func testResolveTypesResponse() *api.HealthTypesResponse {
	return &api.HealthTypesResponse{
		Aggregated: []api.HealthTypeSection{
			{
				Name: "Activity",
				Types: []api.HealthType{
					{ID: 6, Name: "Heart rate", Category: "Cumulative", SubCategory: "Heart rate"},
					{ID: 9, Name: "Step count", Category: "Cumulative", SubCategory: "Activity"},
				},
			},
		},
		Record: []api.HealthTypeSection{
			{
				Name: "Body",
				Types: []api.HealthType{
					{ID: 0, Name: "Body mass", Category: "Record", SubCategory: "Body"},
				},
			},
			{
				Name: "Heart rate",
				Types: []api.HealthType{
					{ID: 52, Name: "Heart rate", Category: "Record", SubCategory: "Heart rate"},
				},
			},
		},
	}
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, response any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Fatalf("json.NewEncoder().Encode() error = %v", err)
	}
}

func intsToString(values []int) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}

	return strings.Join(parts, ",")
}

func snapshotDataState() func() {
	prevTypes := append([]string(nil), dataTypes...)
	prevFrom := dataFrom
	prevTo := dataTo
	prevFormat := dataFormat
	prevRaw := dataRaw
	prevAggregate := dataAggregate
	restoreRoot := snapshotVersionState()

	return func() {
		dataTypes = prevTypes
		dataFrom = prevFrom
		dataTo = prevTo
		dataFormat = prevFormat
		dataRaw = prevRaw
		dataAggregate = prevAggregate
		restoreRoot()
	}
}
