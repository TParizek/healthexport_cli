package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/config"
	"github.com/TParizek/healthexport_cli/internal/typemap"
)

func TestFetchHealthDataRejectsEmptyTypes(t *testing.T) {
	_, err := FetchHealthData(Options{}, FetchRequest{
		From:      "2026-03-20",
		To:        "2026-03-26",
		Types:     []string{},
		Aggregate: "day",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("FetchHealthData() error = %v, want ErrInvalidInput", err)
	}

	if got, want := err.Error(), "invalid input: the 'types' parameter must contain at least one health type"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestFetchHealthDataRejectsUnknownTypeNamesAndIDs(t *testing.T) {
	configPath := writeServiceConfig(t)
	client := newServiceAPIServer(t, serviceHealthTypesResponse(), nil, nil)
	defer client.Close()

	_, err := FetchHealthData(Options{ConfigPath: configPath, APIURL: client.URL}, FetchRequest{
		From:      "2026-03-20",
		To:        "2026-03-26",
		Types:     []string{"nonexistent_type", "999"},
		Aggregate: "day",
	})
	if !errors.Is(err, ErrInvalidInput) || !errors.Is(err, typemap.ErrUnknownType) {
		t.Fatalf("FetchHealthData() error = %v, want invalid unknown type error", err)
	}

	if got, want := err.Error(), `invalid input: unknown health type: nonexistent_type; unknown health type ID: 999`; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestFetchHealthDataRejectsUnknownTypeNameWithoutAggregate(t *testing.T) {
	configPath := writeServiceConfig(t)
	client := newServiceAPIServer(t, serviceHealthTypesResponse(), nil, nil)
	defer client.Close()

	_, err := FetchHealthData(Options{ConfigPath: configPath, APIURL: client.URL}, FetchRequest{
		From:  "2026-03-20",
		To:    "2026-03-26",
		Types: []string{"nonexistent_type"},
	})
	if !errors.Is(err, ErrInvalidInput) || !errors.Is(err, typemap.ErrUnknownType) {
		t.Fatalf("FetchHealthData() error = %v, want invalid unknown type error", err)
	}

	if got, want := err.Error(), `invalid input: unknown health type: nonexistent_type`; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestFetchHealthDataSwapsReversedDateRange(t *testing.T) {
	configPath := writeServiceConfig(t)
	var capturedQuery url.Values
	client := newServiceAPIServer(t, serviceHealthTypesResponse(), []api.EncryptedPackage{
		{
			Type: 9,
			Data: []api.EncryptedUnitGroup{
				{
					Units: "count",
					Records: []api.EncryptedRecord{
						{
							Time:   "2026-03-20T12:00:00Z",
							Nonce:  "AQIDBAUGBwgJCgsM",
							Cipher: "lmaYhg==",
						},
					},
				},
			},
		},
	}, &capturedQuery)
	defer client.Close()

	result, err := FetchHealthData(Options{ConfigPath: configPath, APIURL: client.URL}, FetchRequest{
		From:  "2026-03-26",
		To:    "2026-03-20",
		Types: []string{"step_count"},
	})
	if err != nil {
		t.Fatalf("FetchHealthData() error = %v", err)
	}

	if got, want := result.From, "2026-03-20T00:00:00Z"; got != want {
		t.Fatalf("From = %q, want %q", got, want)
	}

	if got, want := result.To, "2026-03-26T00:00:00Z"; got != want {
		t.Fatalf("To = %q, want %q", got, want)
	}

	if got, want := capturedQuery.Get("dateFrom"), "2026-03-20T00:00:00Z"; got != want {
		t.Fatalf("dateFrom = %q, want %q", got, want)
	}

	if got, want := capturedQuery.Get("dateTo"), "2026-03-26T00:00:00Z"; got != want {
		t.Fatalf("dateTo = %q, want %q", got, want)
	}
}

func TestFetchHealthDataAllowsPartialAggregationForMixedTypes(t *testing.T) {
	configPath := writeServiceConfig(t)
	client := newServiceAPIServer(t, serviceHealthTypesResponse(), []api.EncryptedPackage{
		{
			Type: 9,
			Data: []api.EncryptedUnitGroup{
				{
					Units: "count",
					Records: []api.EncryptedRecord{
						{
							Time:   "2026-03-20T12:00:00Z",
							Nonce:  "AQIDBAUGBwgJCgsM",
							Cipher: "lmaYhg==",
						},
						{
							Time:   "2026-03-20T18:00:00Z",
							Nonce:  "DAsKCQgHBgUEAwIB",
							Cipher: "wEQM8w==",
						},
					},
				},
			},
		},
		{
			Type: 0,
			Data: []api.EncryptedUnitGroup{
				{
					Units: "kg",
					Records: []api.EncryptedRecord{
						{
							Time:   "2026-03-20T08:00:00Z",
							Nonce:  "AQIDBAUGBwgJCgsM",
							Cipher: "lmaYhg==",
						},
					},
				},
			},
		},
	}, nil)
	defer client.Close()

	result, err := FetchHealthData(Options{ConfigPath: configPath, APIURL: client.URL}, FetchRequest{
		From:                    "2026-03-20",
		To:                      "2026-03-26",
		Types:                   []string{"step_count", "body_mass"},
		Aggregate:               "day",
		AllowPartialAggregation: true,
	})
	if err != nil {
		t.Fatalf("FetchHealthData() error = %v", err)
	}

	if got, want := result.Aggregate, "day"; got != want {
		t.Fatalf("Aggregate = %q, want %q", got, want)
	}

	if len(result.Aggregated) != 1 {
		t.Fatalf("len(Aggregated) = %d, want 1", len(result.Aggregated))
	}

	if got, want := result.Aggregated[0].Data[0].Records[0].Value, 8508.0; got != want {
		t.Fatalf("aggregated count = %v, want %v", got, want)
	}

	if len(result.Decrypted) != 1 {
		t.Fatalf("len(Decrypted) = %d, want 1", len(result.Decrypted))
	}

	if got, want := result.Decrypted[0].TypeName, "Body mass"; got != want {
		t.Fatalf("TypeName = %q, want %q", got, want)
	}

	if len(result.Warnings) == 0 {
		t.Fatal("Warnings = empty, want partial aggregation warning")
	}
}

func TestFetchHealthDataIgnoresAggregationForPureRecordTypes(t *testing.T) {
	configPath := writeServiceConfig(t)
	client := newServiceAPIServer(t, serviceHealthTypesResponse(), []api.EncryptedPackage{
		{
			Type: 0,
			Data: []api.EncryptedUnitGroup{
				{
					Units: "kg",
					Records: []api.EncryptedRecord{
						{
							Time:   "2026-03-20T08:00:00Z",
							Nonce:  "AQIDBAUGBwgJCgsM",
							Cipher: "lmaYhg==",
						},
					},
				},
			},
		},
	}, nil)
	defer client.Close()

	result, err := FetchHealthData(Options{ConfigPath: configPath, APIURL: client.URL}, FetchRequest{
		From:                    "2026-03-20",
		To:                      "2026-03-26",
		Types:                   []string{"body_mass"},
		Aggregate:               "day",
		AllowPartialAggregation: true,
	})
	if err != nil {
		t.Fatalf("FetchHealthData() error = %v", err)
	}

	if len(result.Aggregated) != 0 {
		t.Fatalf("len(Aggregated) = %d, want 0", len(result.Aggregated))
	}

	if len(result.Decrypted) != 1 {
		t.Fatalf("len(Decrypted) = %d, want 1", len(result.Decrypted))
	}

	if result.Aggregate != "" {
		t.Fatalf("Aggregate = %q, want empty when no aggregation was applied", result.Aggregate)
	}
}

func TestAggregatePackagesRoundsCountOnly(t *testing.T) {
	packages := []api.DecryptedPackage{
		{
			Type:     9,
			TypeName: "Step count",
			Data: []api.DecryptedUnitGroup{
				{
					Units: "count",
					Records: []api.DecryptedRecord{
						{Time: "2026-03-20T08:00:00Z", Value: "26413.999999999996"},
						{Time: "2026-03-20T09:00:00Z", Value: "0.000000000004"},
					},
				},
			},
		},
		{
			Type:     1,
			TypeName: "Active energy burned",
			Data: []api.DecryptedUnitGroup{
				{
					Units: "kcal",
					Records: []api.DecryptedRecord{
						{Time: "2026-03-20T08:00:00Z", Value: "10.25"},
						{Time: "2026-03-20T09:00:00Z", Value: "0.25"},
					},
				},
			},
		},
	}

	aggregated, err := aggregatePackages(packages, "day")
	if err != nil {
		t.Fatalf("aggregatePackages() error = %v", err)
	}

	if got, want := aggregated[0].Data[0].Records[0].Value, 26414.0; got != want {
		t.Fatalf("count aggregate = %v, want %v", got, want)
	}

	if got, want := aggregated[1].Data[0].Records[0].Value, 10.5; got != want {
		t.Fatalf("kcal aggregate = %v, want %v", got, want)
	}
}

func writeServiceConfig(t *testing.T) string {
	t.Helper()

	configPath := t.TempDir() + "/config.yaml"
	if err := (&config.Config{AccountKey: serviceTestAccountKey}).SaveToPath(configPath); err != nil {
		t.Fatalf("SaveToPath() error = %v", err)
	}

	return configPath
}

func serviceHealthTypesResponse() *api.HealthTypesResponse {
	return &api.HealthTypesResponse{
		Aggregated: []api.HealthTypeSection{
			{
				Name: "Activity",
				Types: []api.HealthType{
					{ID: 9, Name: "Step count", Category: "Cumulative", SubCategory: "Activity"},
				},
			},
		},
		Record: []api.HealthTypeSection{
			{
				Name: "Body",
				Types: []api.HealthType{
					{ID: 0, Name: "Body mass", Category: "Record", SubCategory: "Body"},
					{ID: 24, Name: "Time asleep", Category: "Record", SubCategory: "Sleep"},
				},
			},
		},
		Workout: []api.HealthTypeSection{
			{
				Name: "Workout",
				Types: []api.HealthType{
					{ID: 26, Name: "Workouts", Category: "Workout", SubCategory: "Workout"},
				},
			},
		},
	}
}

func newServiceAPIServer(t *testing.T, healthTypes *api.HealthTypesResponse, encrypted []api.EncryptedPackage, capturedQuery *url.Values) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthtypes":
			writeServiceJSONResponse(t, w, healthTypes)
		case "/healthdata/encrypted":
			if capturedQuery != nil {
				*capturedQuery = r.URL.Query()
			}
			writeServiceJSONResponse(t, w, encrypted)
		default:
			http.NotFound(w, r)
		}
	}))
}

func writeServiceJSONResponse(t *testing.T, w http.ResponseWriter, response any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
}
