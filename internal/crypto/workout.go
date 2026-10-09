package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
)

var ErrWorkoutDecrypt = errors.New("workout decryption failed; verify the account key")
var ErrWorkoutEnvelope = errors.New("invalid workout envelope")

// OpenWorkout authenticates the UID, workout, revision and manifest/page identity.
// As with legacy records, key bytes are UTF-8, not hex-decoded.
func OpenWorkout(sealed, key, uid, id, revision, part string, out any) error {
	if len(sealed) > base64.StdEncoding.EncodedLen(32768+28) {
		return ErrWorkoutEnvelope
	}
	raw, err := base64.StdEncoding.Strict().DecodeString(sealed)
	if err != nil || len(raw) < 28 {
		return ErrWorkoutEnvelope
	}
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return ErrWorkoutDecrypt
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return ErrWorkoutDecrypt
	}
	plain, err := aead.Open(nil, raw[:12], raw[12:], []byte("HEWorkoutDetail:1:"+uid+":"+id+":"+revision+":"+part))
	if err != nil {
		return ErrWorkoutDecrypt
	}
	if len(plain) > 32768 || json.Unmarshal(plain, out) != nil {
		return ErrWorkoutEnvelope
	}
	return nil
}
