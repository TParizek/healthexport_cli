package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEncryptedPackageFixtureUnmarshal(t *testing.T) {
	data := readFixture(t, "encrypted_response.json")

	var packages []EncryptedPackage
	if err := json.Unmarshal(data, &packages); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(packages) != 1 {
		t.Fatalf("len(packages) = %d, want 1", len(packages))
	}

	if got, want := packages[0].Type, 9; got != want {
		t.Fatalf("packages[0].Type = %d, want %d", got, want)
	}

	if got, want := len(packages[0].Data[0].Records), 2; got != want {
		t.Fatalf("len(packages[0].Data[0].Records) = %d, want %d", got, want)
	}
}

func TestHealthTypesResponseFixtureUnmarshal(t *testing.T) {
	data := readFixture(t, "healthtypes_response.json")

	var response HealthTypesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if got, want := response.Aggregated[0].Types[0].Name, "Distance cycling"; got != want {
		t.Fatalf("Aggregated[0].Types[0].Name = %q, want %q", got, want)
	}

	if got, want := response.Record[0].Types[0].Category, "Record"; got != want {
		t.Fatalf("Record[0].Types[0].Category = %q, want %q", got, want)
	}

	if got, want := response.Workout[0].Types[0].ID, 26; got != want {
		t.Fatalf("Workout[0].Types[0].ID = %d, want %d", got, want)
	}
}

func TestAPIErrorError(t *testing.T) {
	err := &APIError{
		StatusCode: 429,
		Body:       "Rate limit exceeded\n",
		Endpoint:   "/healthdata/encrypted",
	}

	if got, want := err.Error(), "api request to /healthdata/encrypted failed with status 429: Rate limit exceeded"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	return data
}
