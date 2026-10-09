// Package workouttest supplies synthetic encrypted workout API fixtures for
// command, service and protocol integration tests. It contains no user data.
package workouttest

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/TParizek/healthexport_cli/internal/auth"
)

const Account = "0123456789abcdef0123456789abcdef"
const ID = "11111111-1111-4111-8111-111111111111"
const Revision = "22222222-2222-4222-8222-222222222222"
const Start = "2026-10-01T10:00:00Z"

// Seal uses a fixed nonce exclusively for reproducible synthetic tests.
func Seal(part string, data any) string {
	key, _ := auth.Parse(Account)
	raw, _ := json.Marshal(data)
	block, _ := aes.NewCipher([]byte(key.DecryptionKey))
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, 12)
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, raw, []byte("HEWorkoutDetail:1:"+key.UID+":"+ID+":"+Revision+":"+part)))
}
func Manifest() map[string]any {
	return map[string]any{"schemaVersion": 1, "workoutId": ID, "start": Start, "end": "2026-10-01T11:00:00Z", "sport": "running", "source": "Synthetic Watch", "activeDuration": 3500, "elapsedDuration": 3600, "totalDistance": 1800,
		"availability": map[string]string{"speed": "available", "events": "available", "splitsKm": "available", "splitsMi": "available", "heartRate": "unavailable"},
		"sections":     map[string]int{"speed": 9, "events": 1, "splitsKm": 1, "splitsMi": 1},
		"aggregation":  map[string]any{"speed": map[string]any{"intervalSeconds": 5, "method": "mean", "source": "healthkit"}}}
}

// Response returns sparse global page indices to exercise section-filtered paging.
func Response(r *http.Request) map[string]any {
	manifest := Manifest()
	if r.URL.Path == "/api/v2/workout-details" {
		return map[string]any{"workouts": []any{map[string]any{"workoutId": ID, "revision": Revision, "start": Start, "manifest": Seal("manifest", manifest)}}, "nextOffset": nil}
	}
	q := r.URL.Query()
	section := q.Get("section")
	offset, _ := strconv.Atoi(q.Get("offset"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	total := manifest["sections"].(map[string]int)[section]
	pages := []any{}
	for i := offset; i < min(total, offset+limit); i++ {
		index := i * 2
		var items []map[string]any
		switch section {
		case "speed":
			items = []map[string]any{{"t": i * 10, "end": i*10 + 5, "value": 0, "kind": "timeMean", "count": 1, "coverage": 5, "breakBefore": i == 8}}
		case "events":
			index = 21
			items = []map[string]any{{"kind": "pause", "start": 100, "end": 110}}
		default:
			index = 23
			if section == "splitsMi" {
				index = 25
			}
			distance := 1000.0
			if section == "splitsMi" {
				distance = 1609.344
			}
			items = []map[string]any{{"index": 1, "distance": distance, "start": 0, "end": 400, "elapsedDuration": 400, "estimated": false}}
		}
		pages = append(pages, map[string]any{"index": index, "section": section, "sealed": Seal(strconv.Itoa(index), map[string]any{"section": section, "items": items})})
	}
	var next any
	if offset+len(pages) < total {
		next = offset + len(pages)
	}
	return map[string]any{"workoutId": ID, "revision": Revision, "start": Start, "manifest": Seal("manifest", manifest), "pages": pages, "nextOffset": next}
}
func Handler(mutate func(*http.Request, map[string]any)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, _ := auth.Parse(Account)
		if r.URL.Query().Get("uid") != key.UID || r.Header.Get("Authorization") != "" {
			http.Error(w, "invalid identity", 400)
			return
		}
		if r.URL.Path != "/api/v2/workout-details" && r.URL.Path != "/api/v2/workout-details/"+ID {
			http.NotFound(w, r)
			return
		}
		if r.URL.Path == "/api/v2/workout-details/"+ID && r.URL.Query().Get("revision") != Revision {
			http.Error(w, "changed", 409)
			return
		}
		body := Response(r)
		if mutate != nil {
			mutate(r, body)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(body); err != nil {
			panic(fmt.Sprintf("fixture encode: %v", err))
		}
	}
}
