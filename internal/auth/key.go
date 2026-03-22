package auth

import (
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidKeyFormat = errors.New("invalid account key format")

	fullFormatPattern = regexp.MustCompile(`^[a-z0-9]{6}\.[a-z0-9]{32}\.[a-z0-9]{4}$`)
	rawFormatPattern  = regexp.MustCompile(`^[a-z0-9]{32}$`)
)

type AccountKey struct {
	Raw           string
	DecryptionKey string
	UID           string
}

func Parse(raw string) (*AccountKey, error) {
	trimmed := strings.TrimSpace(raw)
	if !fullFormatPattern.MatchString(trimmed) && !rawFormatPattern.MatchString(trimmed) {
		return nil, ErrInvalidKeyFormat
	}

	return &AccountKey{
		Raw:           trimmed,
		DecryptionKey: extractDecryptionKey(trimmed),
		UID:           deriveUID(trimmed),
	}, nil
}

func (ak *AccountKey) MaskedKey() string {
	if ak == nil || ak.Raw == "" {
		return ""
	}

	if strings.Contains(ak.Raw, ".") {
		return ak.Raw[:7] + strings.Repeat("*", 30) + ak.Raw[len(ak.Raw)-5:]
	}

	return ak.Raw[:4] + strings.Repeat("*", len(ak.Raw)-8) + ak.Raw[len(ak.Raw)-4:]
}

func deriveUID(raw string) string {
	hash := sha512.Sum512([]byte(raw))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func extractDecryptionKey(raw string) string {
	if strings.Contains(raw, ".") {
		return raw[7:39]
	}

	return raw
}
