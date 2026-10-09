package crypto_test

import (
	"encoding/base64"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/auth"
	"github.com/TParizek/healthexport_cli/internal/crypto"
	"github.com/TParizek/healthexport_cli/internal/workouttest"
)

func TestWorkoutEnvelopeAuthenticatesEveryIdentity(t *testing.T) {
	key, err := auth.Parse(workouttest.Account)
	if err != nil {
		t.Fatal(err)
	}
	sealed := workouttest.Seal("manifest", workouttest.Manifest())
	var out map[string]any
	if err := crypto.OpenWorkout(sealed, key.DecryptionKey, key.UID, workouttest.ID, workouttest.Revision, "manifest", &out); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"key", "uid", "id", "revision", "part", "ciphertext"} {
		t.Run(field, func(t *testing.T) {
			k, u, id, r, p, s := key.DecryptionKey, key.UID, workouttest.ID, workouttest.Revision, "manifest", sealed
			switch field {
			case "key":
				k = "fedcba9876543210fedcba9876543210"
			case "uid":
				u = "different-uid"
			case "id":
				id = workouttest.Revision
			case "revision":
				r = workouttest.ID
			case "part":
				p = "0"
			case "ciphertext":
				raw, e := base64.StdEncoding.DecodeString(s)
				if e != nil {
					t.Fatal(e)
				}
				raw[len(raw)-1] ^= 1
				s = base64.StdEncoding.EncodeToString(raw)
			}
			if err := crypto.OpenWorkout(s, k, u, id, r, p, &out); err != crypto.ErrWorkoutDecrypt {
				t.Fatal("unauthenticated envelope", err)
			}
		})
	}
}
