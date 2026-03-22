package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type authTestVectors struct {
	UIDDerivation           []uidDerivationVector           `json:"uid_derivation"`
	DecryptionKeyExtraction []decryptionKeyExtractionVector `json:"decryption_key_extraction"`
}

type uidDerivationVector struct {
	AccountKey  string `json:"account_key"`
	ExpectedUID string `json:"expected_uid"`
}

type decryptionKeyExtractionVector struct {
	AccountKey  string `json:"account_key"`
	ExpectedKey string `json:"expected_key"`
}

func TestParseMatchesBackendUIDVectors(t *testing.T) {
	vectors := loadAuthVectors(t)

	for _, vector := range vectors.UIDDerivation {
		got, err := Parse(vector.AccountKey)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", vector.AccountKey, err)
		}

		if got.Raw != vector.AccountKey {
			t.Fatalf("Raw = %q, want %q", got.Raw, vector.AccountKey)
		}

		if got.UID != vector.ExpectedUID {
			t.Fatalf("UID = %q, want %q", got.UID, vector.ExpectedUID)
		}
	}
}

func TestParseMatchesBackendDecryptionKeyVectors(t *testing.T) {
	vectors := loadAuthVectors(t)

	for _, vector := range vectors.DecryptionKeyExtraction {
		got, err := Parse(vector.AccountKey)
		if err != nil {
			t.Fatalf("Parse(%q) error = %v", vector.AccountKey, err)
		}

		if got.DecryptionKey != vector.ExpectedKey {
			t.Fatalf("DecryptionKey = %q, want %q", got.DecryptionKey, vector.ExpectedKey)
		}
	}
}

func TestParseInvalidKeyTooShort(t *testing.T) {
	_, err := Parse("abc")
	if !errors.Is(err, ErrInvalidKeyFormat) {
		t.Fatalf("Parse() error = %v, want ErrInvalidKeyFormat", err)
	}
}

func TestParseInvalidKeyUppercase(t *testing.T) {
	_, err := Parse("ABCDEF.0123456789abcdef0123456789abcdef.gh01")
	if !errors.Is(err, ErrInvalidKeyFormat) {
		t.Fatalf("Parse() error = %v, want ErrInvalidKeyFormat", err)
	}
}

func TestParseInvalidKeyWrongDotPositions(t *testing.T) {
	_, err := Parse("abcdef0123456789abcdef0123456789abcdef.gh01")
	if !errors.Is(err, ErrInvalidKeyFormat) {
		t.Fatalf("Parse() error = %v, want ErrInvalidKeyFormat", err)
	}
}

func TestParseEmptyString(t *testing.T) {
	_, err := Parse("")
	if !errors.Is(err, ErrInvalidKeyFormat) {
		t.Fatalf("Parse() error = %v, want ErrInvalidKeyFormat", err)
	}
}

func TestParseTrimsWhitespace(t *testing.T) {
	got, err := Parse("\n abcdef.0123456789abcdef0123456789abcdef.gh01 \t")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got.Raw != "abcdef.0123456789abcdef0123456789abcdef.gh01" {
		t.Fatalf("Raw = %q, want trimmed key", got.Raw)
	}
}

func TestMaskedKeyFullFormat(t *testing.T) {
	ak := &AccountKey{Raw: "abcdef.0123456789abcdef0123456789abcdef.gh01"}

	if got, want := ak.MaskedKey(), "abcdef.******************************.gh01"; got != want {
		t.Fatalf("MaskedKey() = %q, want %q", got, want)
	}
}

func TestMaskedKeyRawFormat(t *testing.T) {
	ak := &AccountKey{Raw: "0123456789abcdef0123456789abcdef"}

	if got, want := ak.MaskedKey(), "0123************************cdef"; got != want {
		t.Fatalf("MaskedKey() = %q, want %q", got, want)
	}
}

func loadAuthVectors(t *testing.T) authTestVectors {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "test_vectors.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}

	var vectors authTestVectors
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	return vectors
}
