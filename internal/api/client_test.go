package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	client := NewClient("https://example.com/api/v2/")

	if got, want := client.BaseURL, "https://example.com/api/v2"; got != want {
		t.Fatalf("BaseURL = %q, want %q", got, want)
	}

	if client.HTTPClient == nil {
		t.Fatal("HTTPClient is nil")
	}

	if got, want := client.HTTPClient.Timeout.String(), "30s"; got != want {
		t.Fatalf("HTTPClient.Timeout = %q, want %q", got, want)
	}
}

func TestFetchEncryptedDataParsesValidResponse(t *testing.T) {
	restore := snapshotUserAgentVersion()
	t.Cleanup(restore)
	SetUserAgentVersion("1.2.3")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodGet; got != want {
			t.Fatalf("method = %q, want %q", got, want)
		}

		if got, want := r.URL.Path, "/api/v2/healthdata/encrypted"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}

		if got, want := r.Header.Get("Accept"), "application/json"; got != want {
			t.Fatalf("Accept header = %q, want %q", got, want)
		}

		if got, want := r.Header.Get("User-Agent"), "healthexport-cli/1.2.3"; got != want {
			t.Fatalf("User-Agent header = %q, want %q", got, want)
		}

		if got, want := r.URL.Query().Get("uid"), "user-123"; got != want {
			t.Fatalf("uid = %q, want %q", got, want)
		}

		if got, want := r.URL.Query()["type[]"], []string{"9"}; !equalStrings(got, want) {
			t.Fatalf("type[] = %v, want %v", got, want)
		}

		if got, want := r.URL.Query().Get("dateFrom"), "2024-01-14T00:00:00Z"; got != want {
			t.Fatalf("dateFrom = %q, want %q", got, want)
		}

		if got, want := r.URL.Query().Get("dateTo"), "2024-01-15T00:00:00Z"; got != want {
			t.Fatalf("dateTo = %q, want %q", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"type":9,"data":[{"units":"count","records":[{"time":"2024-01-14T12:00:00Z","nonce":"TAJDRM2t8DhP1nDO","cipher":"VBX7VLWQ"}]}]}]`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL + "/api/v2")
	packages, err := client.FetchEncryptedData("user-123", []int{9}, "2024-01-14T00:00:00Z", "2024-01-15T00:00:00Z")
	if err != nil {
		t.Fatalf("FetchEncryptedData() error = %v", err)
	}

	if len(packages) != 1 {
		t.Fatalf("len(packages) = %d, want 1", len(packages))
	}

	if got, want := packages[0].Type, 9; got != want {
		t.Fatalf("packages[0].Type = %d, want %d", got, want)
	}

	if got, want := packages[0].Data[0].Records[0].Cipher, "VBX7VLWQ"; got != want {
		t.Fatalf("packages[0].Data[0].Records[0].Cipher = %q, want %q", got, want)
	}
}

func TestFetchEncryptedDataIncludesMultipleTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query()["type[]"], []string{"9", "10", "11"}; !equalStrings(got, want) {
			t.Fatalf("type[] = %v, want %v", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL)
	if _, err := client.FetchEncryptedData("user-123", []int{9, 10, 11}, "2024-01-14T00:00:00Z", "2024-01-15T00:00:00Z"); err != nil {
		t.Fatalf("FetchEncryptedData() error = %v", err)
	}
}

func TestFetchEncryptedDataServerErrors(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "bad request", statusCode: http.StatusBadRequest, body: "Missing uid parametr"},
		{name: "rate limited", statusCode: http.StatusTooManyRequests, body: "Rate limit exceeded"},
		{name: "internal error", statusCode: http.StatusInternalServerError, body: "Internal server error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, tc.body, tc.statusCode)
			}))
			t.Cleanup(server.Close)

			client := NewClient(server.URL)
			_, err := client.FetchEncryptedData("user-123", []int{9}, "2024-01-14T00:00:00Z", "2024-01-15T00:00:00Z")
			if err == nil {
				t.Fatal("FetchEncryptedData() error = nil, want APIError")
			}

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %T, want *APIError", err)
			}

			if got, want := apiErr.StatusCode, tc.statusCode; got != want {
				t.Fatalf("StatusCode = %d, want %d", got, want)
			}

			if !strings.Contains(apiErr.Body, tc.body) {
				t.Fatalf("Body = %q, want substring %q", apiErr.Body, tc.body)
			}
		})
	}
}

func TestFetchEncryptedDataMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"not":"an array"`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL)
	_, err := client.FetchEncryptedData("user-123", []int{9}, "2024-01-14T00:00:00Z", "2024-01-15T00:00:00Z")
	if err == nil {
		t.Fatal("FetchEncryptedData() error = nil, want decode error")
	}

	if !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("error = %q, want decode response error", err)
	}
}

func TestFetchEncryptedDataNetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	baseURL := server.URL
	server.Close()

	client := NewClient(baseURL)
	_, err := client.FetchEncryptedData("user-123", []int{9}, "2024-01-14T00:00:00Z", "2024-01-15T00:00:00Z")
	if err == nil {
		t.Fatal("FetchEncryptedData() error = nil, want network error")
	}

	if !strings.Contains(err.Error(), "send request") {
		t.Fatalf("error = %q, want send request context", err)
	}
}

func TestFetchEncryptedDataEncodesQueryParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawQuery := r.URL.RawQuery
		if !strings.Contains(rawQuery, "uid=user%2B123%2F%3D%3F%26value") {
			t.Fatalf("RawQuery = %q, want encoded uid", rawQuery)
		}

		if !strings.Contains(rawQuery, "dateFrom=2024-01-14T12%3A00%3A00%2B02%3A00") {
			t.Fatalf("RawQuery = %q, want encoded dateFrom", rawQuery)
		}

		if !strings.Contains(rawQuery, "type%5B%5D=9") || !strings.Contains(rawQuery, "type%5B%5D=10") {
			t.Fatalf("RawQuery = %q, want encoded type[] parameters", rawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL)
	if _, err := client.FetchEncryptedData("user+123/=?&value", []int{9, 10}, "2024-01-14T12:00:00+02:00", "2024-01-15T12:00:00+02:00"); err != nil {
		t.Fatalf("FetchEncryptedData() error = %v", err)
	}
}

func TestFetchHealthTypesParsesValidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/api/v2/healthtypes"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"aggregated":[{"name":"Activity","types":[{"id":9,"name":"Step count","category":"Cumulative","subCategory":"Activity"}]}],"record":[{"name":"Body","types":[{"id":0,"name":"Body mass","category":"Record","subCategory":"Body"}]}],"workout":[{"name":"Workout","types":[{"id":26,"name":"Workouts","category":"Workout","subCategory":"Workout"}]}]}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL + "/api/v2/")
	response, err := client.FetchHealthTypes()
	if err != nil {
		t.Fatalf("FetchHealthTypes() error = %v", err)
	}

	if got, want := len(response.Aggregated), 1; got != want {
		t.Fatalf("len(Aggregated) = %d, want %d", got, want)
	}

	if got, want := response.Aggregated[0].Types[0].Name, "Step count"; got != want {
		t.Fatalf("Aggregated[0].Types[0].Name = %q, want %q", got, want)
	}

	if got, want := response.Record[0].Types[0].SubCategory, "Body"; got != want {
		t.Fatalf("Record[0].Types[0].SubCategory = %q, want %q", got, want)
	}
}

func TestFetchHealthTypesServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "backend unavailable", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)

	client := NewClient(server.URL)
	_, err := client.FetchHealthTypes()
	if err == nil {
		t.Fatal("FetchHealthTypes() error = nil, want APIError")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T, want *APIError", err)
	}

	if got, want := apiErr.StatusCode, http.StatusBadGateway; got != want {
		t.Fatalf("StatusCode = %d, want %d", got, want)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}

func snapshotUserAgentVersion() func() {
	prev := userAgentVersion
	return func() {
		userAgentVersion = prev
	}
}
