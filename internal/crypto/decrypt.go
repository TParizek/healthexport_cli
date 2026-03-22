package crypto

import (
	"encoding/base64"
	"fmt"

	"github.com/TParizek/healthexport_cli/internal/api"
	"golang.org/x/crypto/chacha20"
)

func Decrypt(key string, nonceB64 string, cipherB64 string) (string, error) {
	keyBytes := []byte(key)
	if len(keyBytes) != chacha20.KeySize {
		return "", fmt.Errorf("key must be %d bytes, got %d", chacha20.KeySize, len(keyBytes))
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return "", fmt.Errorf("invalid nonce base64: %w", err)
	}

	if len(nonce) != chacha20.NonceSize {
		return "", fmt.Errorf("nonce must be %d bytes, got %d", chacha20.NonceSize, len(nonce))
	}

	ciphertext, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", fmt.Errorf("invalid cipher base64: %w", err)
	}

	cipher, err := chacha20.NewUnauthenticatedCipher(keyBytes, nonce)
	if err != nil {
		return "", fmt.Errorf("chacha20 init failed: %w", err)
	}

	plaintext := make([]byte, len(ciphertext))
	cipher.XORKeyStream(plaintext, ciphertext)

	return string(plaintext), nil
}

func DecryptRecords(packages []api.EncryptedPackage, key string) ([]api.DecryptedPackage, error) {
	decrypted := make([]api.DecryptedPackage, 0, len(packages))

	for pkgIndex, pkg := range packages {
		outPkg := api.DecryptedPackage{
			Type: pkg.Type,
			Data: make([]api.DecryptedUnitGroup, 0, len(pkg.Data)),
		}

		for groupIndex, group := range pkg.Data {
			outGroup := api.DecryptedUnitGroup{
				Units:   group.Units,
				Records: make([]api.DecryptedRecord, 0, len(group.Records)),
			}

			for recordIndex, record := range group.Records {
				value, err := Decrypt(key, record.Nonce, record.Cipher)
				if err != nil {
					return nil, fmt.Errorf(
						"decrypt package %d (type %d) group %d record %d: %w",
						pkgIndex,
						pkg.Type,
						groupIndex,
						recordIndex,
						err,
					)
				}

				outGroup.Records = append(outGroup.Records, api.DecryptedRecord{
					Time:  record.Time,
					Value: value,
				})
			}

			outPkg.Data = append(outPkg.Data, outGroup)
		}

		decrypted = append(decrypted, outPkg)
	}

	return decrypted, nil
}
