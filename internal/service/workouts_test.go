package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"github.com/TParizek/healthexport_cli/internal/crypto"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

const workoutID = "11111111-1111-4111-8111-111111111111"
const workoutRevision = "22222222-2222-4222-8222-222222222222"

func sealWorkout(t *testing.T, c *WorkoutReader, part string, data any) string {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := aes.NewCipher([]byte(c.key))
	a, _ := cipher.NewGCM(b)
	nonce := make([]byte, 12)
	return base64.StdEncoding.EncodeToString(a.Seal(nonce, nonce, raw, []byte("HEWorkoutDetail:1:"+c.uid+":"+workoutID+":"+workoutRevision+":"+part)))
}
func workoutManifest() WorkoutManifest {
	return WorkoutManifest{SchemaVersion: 1, WorkoutID: workoutID, Start: "2026-10-01T10:00:00Z", End: "2026-10-01T11:00:00Z", Sport: "running", Source: "Watch", ActiveDuration: 3500, ElapsedDuration: 3600, Availability: map[string]string{"speed": "available"}, Sections: map[string]int{"speed": 2, "events": 1, "splitsKm": 1}, Aggregation: map[string]WorkoutAggregation{"speed": {IntervalSeconds: 5, Method: "mean", Source: "derivedDistance"}}}
}
func TestWorkoutEndpointsAndScopedPages(t *testing.T) {
	var c *WorkoutReader
	var sections []string
	c = workoutTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("uid") != c.uid || strings.Contains(r.URL.String(), testAccount) || r.Header.Get("Authorization") != "" {
			t.Error("credential boundary")
		}
		manifest := sealWorkout(t, c, "manifest", workoutManifest())
		if r.URL.Path == "/api/v2/workout-details" {
			if r.URL.Query().Get("dateFrom") != "2026-10-01T00:00:00.000Z" {
				t.Error("date not canonicalized", r.URL.Query().Get("dateFrom"))
			}
			writeWorkoutResponse(t, w, map[string]any{"workouts": []any{map[string]any{"workoutId": workoutID, "revision": workoutRevision, "start": workoutManifest().Start, "manifest": manifest}}, "nextOffset": nil})
			return
		}
		if r.URL.Path != "/api/v2/workout-details/"+workoutID || r.URL.Query().Get("revision") != workoutRevision {
			t.Error("wrong page endpoint")
		}
		section := r.URL.Query().Get("section")
		sections = append(sections, section)
		index := 7
		items := []map[string]any{{"t": 0, "end": 5, "value": 2.5, "kind": "timeMean", "count": 3, "coverage": 5, "breakBefore": true}}
		var next any = 1
		if section == "events" {
			items = []map[string]any{{"kind": "pause", "start": 100, "end": 110}}
			next = nil
		} else if section == "splitsKm" {
			items = []map[string]any{{"index": 1, "distance": 1000, "start": 0, "end": 400, "elapsedDuration": 400, "estimated": true}}
			next = nil
		}
		sealed := sealWorkout(t, c, strconv.Itoa(index), map[string]any{"section": section, "items": items})
		writeWorkoutResponse(t, w, map[string]any{"workoutId": workoutID, "revision": workoutRevision, "start": workoutManifest().Start, "manifest": manifest, "pages": []any{map[string]any{"index": index, "section": section, "sealed": sealed}}, "nextOffset": next})
	})
	ctx := context.Background()
	list, err := c.ListWorkouts(ctx, WorkoutListQuery{Start: "2026-10-01T02:00:00+02:00", End: "2026-10-02T00:00:00Z"})
	if err != nil || len(list.Workouts) != 1 {
		t.Fatal(list, err)
	}
	series, err := c.GetWorkoutSeries(ctx, WorkoutPageQuery{WorkoutID: workoutID, Revision: workoutRevision, Section: "speed", Unit: "km/h"})
	if err != nil || series.Unit != "km/h" || series.Pages[0].Items[0]["value"] != float64(9) || series.NextOffset == nil || *series.NextOffset != 1 {
		t.Fatal(series, err)
	}
	if series.Aggregation == nil || series.Aggregation.Source != "derivedDistance" || series.Pages[0].Items[0]["count"] != float64(3) || series.Pages[0].Items[0]["coverage"] != float64(5) || series.Pages[0].Items[0]["breakBefore"] != true {
		t.Fatal("aggregation metadata lost", series)
	}
	detail, err := c.GetWorkoutDetail(ctx, WorkoutDetailQuery{WorkoutID: workoutID, Revision: workoutRevision})
	if err != nil || len(detail.Sections) != 2 {
		t.Fatal(detail, err)
	}
	if strings.Join(sections, ",") != "speed,events,splitsKm" {
		t.Fatal("implicit stream expansion", sections)
	}
}
func TestWorkoutAuthenticationAndBounds(t *testing.T) {
	c := workoutTestClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("invalid request reached network") })
	for _, q := range []WorkoutPageQuery{{WorkoutID: "../bad", Revision: workoutRevision, Section: "speed"}, {WorkoutID: workoutID, Revision: workoutRevision, Section: "events"}, {WorkoutID: workoutID, Revision: workoutRevision, Section: "speed", Limit: 9}, {WorkoutID: workoutID, Revision: workoutRevision, Section: "speed", Unit: "bpm"}} {
		if _, err := c.GetWorkoutSeries(context.Background(), q); err == nil {
			t.Fatal("accepted invalid query")
		}
	}
	var out map[string]any
	sealed := sealWorkout(t, c, "7", map[string]any{"value": strings.Repeat("x", 5000)})
	if err := c.openWorkout(sealed, workoutID, workoutRevision, "7", &out); err != nil {
		t.Fatal("detail must exceed legacy 4K cap", err)
	}
	for _, part := range []string{"8", "manifest"} {
		if err := c.openWorkout(sealed, workoutID, workoutRevision, part, &out); err != crypto.ErrWorkoutDecrypt {
			t.Fatal("AAD not authenticated", err)
		}
	}
	if err := c.openWorkout(sealed, workoutID, "33333333-3333-4333-8333-333333333333", "7", &out); err != crypto.ErrWorkoutDecrypt {
		t.Fatal("revision not authenticated")
	}
	if err := c.openWorkout(strings.Repeat("A", 50000), workoutID, workoutRevision, "7", &out); err != crypto.ErrWorkoutEnvelope {
		t.Fatal("missing byte cap")
	}
}

func TestWorkoutManifestEquivalentInstants(t *testing.T) {
	if !validManifest(workoutManifest(), workoutID, "2026-10-01T10:00:00.000Z") {
		t.Fatal("equivalent timestamp rejected")
	}
	if validManifest(workoutManifest(), workoutID, "2026-10-01T10:00:01.000Z") {
		t.Fatal("different timestamp accepted")
	}
}

// This fixture was emitted by the production Swift CryptoKit exporter, not Go's test sealer.
func TestSwiftWorkoutEnvelopeInteroperability(t *testing.T) {
	raw, err := os.ReadFile("testdata/workout-detail-ios.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		WorkoutID string `json:"workoutId"`
		Revision  string `json:"revision"`
		Start     string `json:"start"`
		Manifest  string `json:"manifest"`
		Pages     []struct {
			Section string `json:"section"`
			Sealed  string `json:"sealed"`
		} `json:"pages"`
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	c, err := NewWorkoutReader(Options{APIURL: "https://backend.example.test/api/v2", AccountKey: "0123456789abcdef0123456789abcdef"})
	if err != nil {
		t.Fatal(err)
	}
	var manifest WorkoutManifest
	if err = c.openWorkout(fixture.Manifest, fixture.WorkoutID, fixture.Revision, "manifest", &manifest); err != nil {
		t.Fatal("Swift manifest decryption", err)
	}
	if !validManifest(manifest, fixture.WorkoutID, fixture.Start) || manifest.Source != "Interop fixture" || manifest.ElapsedDuration != 60 || manifest.TotalDistance == nil || *manifest.TotalDistance != 200 || manifest.Sections["heartRate"] != 1 {
		t.Fatal("Swift manifest mismatch", manifest)
	}
	var page struct {
		Section string           `json:"section"`
		Items   []map[string]any `json:"items"`
	}
	if err = c.openWorkout(fixture.Pages[0].Sealed, fixture.WorkoutID, fixture.Revision, "0", &page); err != nil {
		t.Fatal("Swift page decryption", err)
	}
	if page.Section != "heartRate" || len(page.Items) != 1 || page.Items[0]["t"] != float64(0) || page.Items[0]["end"] != float64(10) || page.Items[0]["value"] != float64(120) {
		t.Fatal("Swift page mismatch", page)
	}
	for index, encrypted := range fixture.Pages {
		page.Items = nil
		if err := c.openWorkout(encrypted.Sealed, fixture.WorkoutID, fixture.Revision, strconv.Itoa(index), &page); err != nil {
			t.Fatal("Swift aggregate page", index, err)
		}
		if page.Section != encrypted.Section {
			t.Fatal("Swift section mismatch", index)
		}
		for _, item := range page.Items {
			if !validWorkoutItem(page.Section, item) {
				t.Fatal("Swift aggregate rejected", page.Section, item)
			}
		}
	}
	if err = c.openWorkout(fixture.Pages[0].Sealed, fixture.WorkoutID, fixture.Revision, "1", &page); err != crypto.ErrWorkoutDecrypt {
		t.Fatal("Swift page identity not authenticated", err)
	}
}

func TestWorkoutAggregationValidation(t *testing.T) {
	valid := []struct {
		section string
		item    map[string]any
	}{
		{"speed", map[string]any{"t": 0.0, "end": 40.0, "value": 2.0, "kind": "sample"}},
		{"elevation", map[string]any{"t": 0.0, "end": 5.0, "value": -10.0, "kind": "sampleMean", "count": 2.0}},
		{"heartRate", map[string]any{"t": 0.0, "end": 10.0, "value": 120.0, "kind": "timeMean", "count": 2.0, "coverage": 10.0, "min": 110.0, "max": 130.0, "breakBefore": true}},
		{"distance", map[string]any{"t": 5.0, "value": 12.0, "kind": "endpoint", "count": 2.0}},
	}
	for _, tc := range valid {
		if !validWorkoutItem(tc.section, tc.item) {
			t.Fatalf("valid %s rejected: %v", tc.section, tc.item)
		}
	}
	for _, tc := range []struct {
		name, section, field string
		value                any
	}{
		{"missing kind", "speed", "kind", nil}, {"invalid kind", "speed", "kind", "average"},
		{"negative value", "speed", "value", -1.0}, {"fractional count", "heartRate", "count", 1.5},
		{"zero count", "heartRate", "count", 0.0}, {"missing count", "heartRate", "count", nil},
		{"overcoverage", "heartRate", "coverage", 10.01}, {"missing coverage", "heartRate", "coverage", nil},
		{"negative coverage", "heartRate", "coverage", -1.0}, {"missing extreme", "heartRate", "min", nil},
		{"reversed extreme", "heartRate", "min", 121.0}, {"bad break", "heartRate", "breakBefore", "true"},
		{"endpoint only distance", "speed", "kind", "endpoint"}, {"sample cannot have extrema", "heartRate", "kind", "sample"},
		{"coordinate rejected", "heartRate", "latitude", 50.0}, {"distance cannot average", "distance", "kind", "sampleMean"},
		{"endpoint no span", "distance", "end", 6.0}, {"sampleMean no coverage", "elevation", "coverage", 5.0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var item map[string]any
			for _, candidate := range valid {
				if candidate.section == tc.section {
					item = make(map[string]any)
					for k, v := range candidate.item {
						item[k] = v
					}
					break
				}
			}
			if tc.value == nil {
				delete(item, tc.field)
			} else {
				item[tc.field] = tc.value
			}
			if validWorkoutItem(tc.section, item) {
				t.Fatal("invalid item accepted", item)
			}
		})
	}
	for _, mutate := range []func(*WorkoutManifest){
		func(m *WorkoutManifest) { m.Aggregation = nil },
		func(m *WorkoutManifest) { delete(m.Aggregation, "speed") },
		func(m *WorkoutManifest) { m.Aggregation["events"] = WorkoutAggregation{5, "mean", "healthkit"} },
		func(m *WorkoutManifest) { m.Aggregation["speed"] = WorkoutAggregation{30, "mean", "healthkit"} },
		func(m *WorkoutManifest) { m.Aggregation["speed"] = WorkoutAggregation{5, "last", "healthkit"} },
		func(m *WorkoutManifest) { m.Aggregation["speed"] = WorkoutAggregation{5, "mean", "other"} },
	} {
		m := workoutManifest()
		mutate(&m)
		if validManifest(m, workoutID, m.Start) {
			t.Fatal("invalid metadata accepted", m)
		}
	}
}

func TestWorkoutSeriesAuthenticatesManifest(t *testing.T) {
	var c *WorkoutReader
	c = workoutTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeWorkoutResponse(t, w, map[string]any{"workoutId": workoutID, "revision": workoutRevision, "start": workoutManifest().Start, "manifest": sealWorkout(t, c, "wrong-part", workoutManifest()), "pages": []any{}, "nextOffset": nil})
	})
	if _, err := c.GetWorkoutSeries(context.Background(), WorkoutPageQuery{WorkoutID: workoutID, Revision: workoutRevision, Section: "speed"}); err != crypto.ErrWorkoutDecrypt {
		t.Fatal("series accepted unauthenticated metadata", err)
	}
}

func TestWorkoutSeriesPreservesObservedExtrema(t *testing.T) {
	var c *WorkoutReader
	c = workoutTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		manifest := workoutManifest()
		manifest.Sections = map[string]int{"power": 1}
		manifest.Availability = map[string]string{"power": "available"}
		manifest.Aggregation = map[string]WorkoutAggregation{"power": {IntervalSeconds: 5, Method: "mean", Source: "healthkit"}}
		items := []map[string]any{{"t": 0, "end": 5, "value": 275, "kind": "timeMean", "count": 5, "coverage": 5, "min": 150, "max": 900}}
		writeWorkoutResponse(t, w, map[string]any{"workoutId": workoutID, "revision": workoutRevision, "start": manifest.Start, "manifest": sealWorkout(t, c, "manifest", manifest), "pages": []any{map[string]any{"index": 0, "section": "power", "sealed": sealWorkout(t, c, "0", map[string]any{"section": "power", "items": items})}}, "nextOffset": nil})
	})
	result, err := c.GetWorkoutSeries(context.Background(), WorkoutPageQuery{WorkoutID: workoutID, Revision: workoutRevision, Section: "power", Unit: "W"})
	if err != nil {
		t.Fatal(err)
	}
	point := result.Pages[0].Items[0]
	if point["min"] != float64(150) || point["max"] != float64(900) || point["value"] != float64(275) || point["count"] != float64(5) || result.Aggregation == nil || result.Aggregation.IntervalSeconds != 5 {
		t.Fatal("mean/extrema metadata changed", result)
	}
}

const testAccount = "0123456789abcdef0123456789abcdef"

func workoutTestClient(t *testing.T, handler http.HandlerFunc) *WorkoutReader {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	reader, err := NewWorkoutReader(Options{APIURL: server.URL + "/api/v2", AccountKey: testAccount})
	if err != nil {
		t.Fatal(err)
	}
	return reader
}

func writeWorkoutResponse(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Error(err)
	}
}
