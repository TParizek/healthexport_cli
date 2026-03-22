package crypto

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/api"
)

type decryptTestVectors struct {
	Decryption []decryptVector `json:"decryption"`
}

type decryptVector struct {
	Key               string `json:"key"`
	NonceB64          string `json:"nonce_b64"`
	CipherB64         string `json:"cipher_b64"`
	ExpectedPlaintext string `json:"expected_plaintext"`
}

func TestDecryptMatchesBackendVectors(t *testing.T) {
	vectors := loadDecryptVectors(t)

	for _, vector := range vectors.Decryption {
		got, err := Decrypt(vector.Key, vector.NonceB64, vector.CipherB64)
		if err != nil {
			t.Fatalf("Decrypt() error = %v", err)
		}

		if got != vector.ExpectedPlaintext {
			t.Fatalf("Decrypt() = %q, want %q", got, vector.ExpectedPlaintext)
		}
	}
}

func TestDecryptWrongKeyLength(t *testing.T) {
	_, err := Decrypt("short", "AQIDBAUGBwgJCgsM", "lmaYhg==")
	assertErrorContains(t, err, "key must be 32 bytes")
}

func TestDecryptInvalidBase64Nonce(t *testing.T) {
	_, err := Decrypt("0123456789abcdef0123456789abcdef", "%%%not-base64%%%", "lmaYhg==")
	assertErrorContains(t, err, "invalid nonce base64")
}

func TestDecryptInvalidBase64Cipher(t *testing.T) {
	_, err := Decrypt("0123456789abcdef0123456789abcdef", "AQIDBAUGBwgJCgsM", "%%%not-base64%%%")
	assertErrorContains(t, err, "invalid cipher base64")
}

func TestDecryptWrongNonceLength(t *testing.T) {
	_, err := Decrypt("0123456789abcdef0123456789abcdef", "AQIDBA==", "lmaYhg==")
	assertErrorContains(t, err, "nonce must be 12 bytes")
}

func TestDecryptEmptyCipherReturnsEmptyString(t *testing.T) {
	got, err := Decrypt("0123456789abcdef0123456789abcdef", "AQIDBAUGBwgJCgsM", "")
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if got != "" {
		t.Fatalf("Decrypt() = %q, want empty string", got)
	}
}

func TestDecryptRecordsTransformsPackages(t *testing.T) {
	packages := []api.EncryptedPackage{
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
	}

	got, err := DecryptRecords(packages, "0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("DecryptRecords() error = %v", err)
	}

	want := []api.DecryptedPackage{
		{
			Type: 9,
			Data: []api.DecryptedUnitGroup{
				{
					Units: "count",
					Records: []api.DecryptedRecord{
						{Time: "2024-01-14T12:00:00Z", Value: "75.5"},
						{Time: "2024-01-14T18:00:00Z", Value: "8432"},
					},
				},
			},
		},
	}

	if !decryptedPackagesEqual(got, want) {
		t.Fatalf("DecryptRecords() = %#v, want %#v", got, want)
	}
}

func TestDecryptRecordsIncludesRecordLocationOnFailure(t *testing.T) {
	packages := []api.EncryptedPackage{
		{
			Type: 9,
			Data: []api.EncryptedUnitGroup{
				{
					Units: "count",
					Records: []api.EncryptedRecord{
						{
							Time:   "2024-01-14T12:00:00Z",
							Nonce:  "bad",
							Cipher: "lmaYhg==",
						},
					},
				},
			},
		},
	}

	_, err := DecryptRecords(packages, "0123456789abcdef0123456789abcdef")
	assertErrorContains(t, err, "decrypt package 0 (type 9) group 0 record 0")
}

func loadDecryptVectors(t *testing.T) decryptTestVectors {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "test_vectors.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	var vectors decryptTestVectors
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return vectors
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %q, want substring %q", err.Error(), want)
	}
}

func decryptedPackagesEqual(got, want []api.DecryptedPackage) bool {
	return reflect.DeepEqual(got, want)
}

func TestDecryptRecordsReturnsWrappedDecryptError(t *testing.T) {
	packages := []api.EncryptedPackage{
		{
			Type: 9,
			Data: []api.EncryptedUnitGroup{
				{
					Units: "count",
					Records: []api.EncryptedRecord{
						{
							Time:   "2024-01-14T12:00:00Z",
							Nonce:  "AQIDBAUGBwgJCgsM",
							Cipher: "%%%bad%%%",
						},
					},
				},
			},
		},
	}

	_, err := DecryptRecords(packages, "0123456789abcdef0123456789abcdef")
	if err == nil {
		t.Fatal("DecryptRecords() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "invalid cipher base64") {
		t.Fatalf("DecryptRecords() error = %v, want wrapped decrypt error", err)
	}
}
