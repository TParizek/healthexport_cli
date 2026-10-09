package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/auth"
	"github.com/TParizek/healthexport_cli/internal/crypto"
	"math"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"time"
)

var ErrWorkoutQuery = fmt.Errorf("%w: use workout ID/revision from list, bounded pagination and a supported section/unit", ErrInvalidInput)
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var streamUnits = map[string]string{"heartRate": "bpm", "distance": "m", "speed": "m/s", "elevation": "m", "power": "W", "cadence": "rpm"}

type WorkoutListQuery struct {
	Start  string `json:"start" jsonschema:"Inclusive RFC3339 start; range at most 366 days"`
	End    string `json:"end"`
	Offset int    `json:"offset,omitempty"`
	Limit  int    `json:"limit,omitempty" jsonschema:"1 to 10; default 10"`
}
type WorkoutAggregation struct {
	IntervalSeconds int    `json:"intervalSeconds"`
	Method          string `json:"method"`
	Source          string `json:"source"`
}
type WorkoutManifest struct {
	SchemaVersion   int                           `json:"schemaVersion"`
	WorkoutID       string                        `json:"workoutId"`
	Start           string                        `json:"start"`
	End             string                        `json:"end"`
	Sport           string                        `json:"sport"`
	Source          string                        `json:"source"`
	ActiveDuration  float64                       `json:"activeDuration"`
	ElapsedDuration float64                       `json:"elapsedDuration"`
	TotalDistance   *float64                      `json:"totalDistance,omitempty"`
	Availability    map[string]string             `json:"availability"`
	Sections        map[string]int                `json:"sections"`
	Aggregation     map[string]WorkoutAggregation `json:"aggregation"`
}
type WorkoutSummary struct {
	WorkoutID string          `json:"workoutId"`
	Revision  string          `json:"revision"`
	Start     string          `json:"start"`
	Manifest  WorkoutManifest `json:"manifest"`
}
type WorkoutListResult struct {
	Workouts   []WorkoutSummary `json:"workouts"`
	NextOffset *int             `json:"nextOffset"`
}
type WorkoutPageQuery struct {
	WorkoutID string `json:"workoutId"`
	Revision  string `json:"revision"`
	Section   string `json:"section" jsonschema:"heartRate distance speed elevation power or cadence"`
	Offset    int    `json:"offset,omitempty"`
	Limit     int    `json:"limit,omitempty" jsonschema:"1 to 8 source pages; default 1"`
	Unit      string `json:"unit,omitempty" jsonschema:"Optional output unit: canonical unit or km mi ft km/h mph"`
}
type WorkoutPage struct {
	Index   int              `json:"index"`
	Section string           `json:"section"`
	Items   []map[string]any `json:"items"`
}
type WorkoutSeriesResult struct {
	Aggregation *WorkoutAggregation `json:"aggregation,omitempty"`
	WorkoutID   string              `json:"workoutId"`
	Revision    string              `json:"revision"`
	Section     string              `json:"section"`
	Unit        string              `json:"unit"`
	Pages       []WorkoutPage       `json:"pages"`
	NextOffset  *int                `json:"nextOffset"`
}
type WorkoutDetailQuery struct {
	WorkoutID string `json:"workoutId"`
	Revision  string `json:"revision"`
	SplitUnit string `json:"splitUnit,omitempty" jsonschema:"km or mi; default km"`
	Offset    int    `json:"offset,omitempty" jsonschema:"Page offset applied separately to events and selected splits"`
	Limit     int    `json:"limit,omitempty" jsonschema:"1 to 4 pages per section; default 1"`
}
type WorkoutDetailResult struct {
	Workout  WorkoutSummary        `json:"workout"`
	Sections []WorkoutSeriesResult `json:"sections"`
}

func validPage(q WorkoutPageQuery) bool {
	return uuidPattern.MatchString(q.WorkoutID) && uuidPattern.MatchString(q.Revision) && q.Offset >= 0 && q.Offset <= 512 && q.Limit >= 1 && q.Limit <= 8
}

// WorkoutReader shares authenticated, bounded reads between CLI and local MCP.
type WorkoutReader struct {
	client   *api.Client
	key, uid string
}

func NewWorkoutReader(opts Options) (*WorkoutReader, error) {
	key, _, err := auth.ResolveWithConfigPath(opts.AccountKey, opts.ConfigPath)
	if err != nil {
		return nil, err
	}
	return &WorkoutReader{client: opts.APIClient(), key: key.DecryptionKey, uid: key.UID}, nil
}

var ErrWorkoutData = fmt.Errorf("invalid or incomplete workout data")

func (c *WorkoutReader) get(ctx context.Context, path string, values url.Values, out any) error {
	return c.client.FetchWorkoutJSON(ctx, path, values, out)
}
func (c *WorkoutReader) openWorkout(sealed, id, rev, part string, out any) error {
	return crypto.OpenWorkout(sealed, c.key, c.uid, id, rev, part, out)
}
func workoutOutputBound(value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return ErrWorkoutData
	}
	if len(raw) > 512<<10 {
		return fmt.Errorf("workout response exceeds limit; request fewer pages")
	}
	return nil
}
func (c *WorkoutReader) ListWorkouts(ctx context.Context, q WorkoutListQuery) (WorkoutListResult, error) {
	var result WorkoutListResult
	if q.Limit == 0 {
		q.Limit = 10
	}
	a, e1 := time.Parse(time.RFC3339Nano, q.Start)
	b, e2 := time.Parse(time.RFC3339Nano, q.End)
	if e1 != nil || e2 != nil || b.Before(a) || b.Sub(a) > 366*24*time.Hour || a.Nanosecond()%int(time.Millisecond) != 0 || b.Nanosecond()%int(time.Millisecond) != 0 || q.Offset < 0 || q.Offset > 10000 || q.Limit < 1 || q.Limit > 10 {
		return result, ErrWorkoutQuery
	}
	var wire struct {
		Workouts []struct {
			WorkoutID string `json:"workoutId"`
			Revision  string `json:"revision"`
			Start     string `json:"start"`
			Manifest  string `json:"manifest"`
		}
		NextOffset *int `json:"nextOffset"`
	}
	v := url.Values{"uid": {c.uid}, "dateFrom": {a.UTC().Format("2006-01-02T15:04:05.000Z")}, "dateTo": {b.UTC().Format("2006-01-02T15:04:05.000Z")}, "offset": {strconv.Itoa(q.Offset)}, "limit": {strconv.Itoa(q.Limit)}}
	if err := c.get(ctx, "/workout-details", v, &wire); err != nil {
		return result, err
	}
	if wire.Workouts == nil || len(wire.Workouts) > q.Limit || !validNext(wire.NextOffset, q.Offset, len(wire.Workouts)) {
		return result, ErrWorkoutData
	}
	result.Workouts = make([]WorkoutSummary, 0, len(wire.Workouts))
	result.NextOffset = wire.NextOffset
	for _, w := range wire.Workouts {
		if !uuidPattern.MatchString(w.WorkoutID) || !uuidPattern.MatchString(w.Revision) {
			return WorkoutListResult{}, ErrWorkoutData
		}
		var m WorkoutManifest
		if err := c.openWorkout(w.Manifest, w.WorkoutID, w.Revision, "manifest", &m); err != nil {
			return WorkoutListResult{}, err
		}
		at, err := time.Parse(time.RFC3339Nano, w.Start)
		if err != nil || at.Before(a) || at.After(b) || !validManifest(m, w.WorkoutID, w.Start) {
			return WorkoutListResult{}, ErrWorkoutData
		}
		total := 0
		for section, count := range m.Sections {
			if !validSection(section) || count < 0 {
				return WorkoutListResult{}, ErrWorkoutData
			}
			total += count
		}
		if total > 512 {
			return WorkoutListResult{}, ErrWorkoutData
		}
		result.Workouts = append(result.Workouts, WorkoutSummary{w.WorkoutID, w.Revision, w.Start, m})
	}
	return result, workoutOutputBound(result)
}
func validNext(next *int, offset, count int) bool {
	return next == nil || (count > 0 && *next == offset+count)
}
func validSection(s string) bool {
	return streamUnits[s] != "" || s == "events" || s == "splitsKm" || s == "splitsMi"
}
func unitFactor(section, unit string) (string, float64, error) {
	canonical := streamUnits[section]
	if canonical == "" {
		return "", 0, ErrWorkoutQuery
	}
	if unit == "" || unit == canonical {
		return canonical, 1, nil
	}
	if canonical == "m" {
		switch unit {
		case "km":
			return unit, .001, nil
		case "mi":
			return unit, 1 / 1609.344, nil
		case "ft":
			return unit, 1 / .3048, nil
		}
	}
	if canonical == "m/s" {
		switch unit {
		case "km/h":
			return unit, 3.6, nil
		case "mph":
			return unit, 3600 / 1609.344, nil
		}
	}
	return "", 0, ErrWorkoutQuery
}
func (c *WorkoutReader) GetWorkoutSeries(ctx context.Context, q WorkoutPageQuery) (WorkoutSeriesResult, error) {
	unit, factor, err := unitFactor(q.Section, q.Unit)
	if err != nil {
		return WorkoutSeriesResult{}, err
	}
	out, err := c.workoutPages(ctx, q, nil)
	if err != nil {
		return out, err
	}
	out.Unit = unit
	for _, page := range out.Pages {
		for _, item := range page.Items {
			v, ok := item["value"].(float64)
			if !ok {
				return WorkoutSeriesResult{}, ErrWorkoutData
			}
			item["value"] = v * factor
			for _, key := range []string{"min", "max"} {
				if extreme, ok := item[key].(float64); ok {
					item[key] = extreme * factor
				}
			}
		}
	}
	return out, workoutOutputBound(out)
}
func (c *WorkoutReader) workoutPages(ctx context.Context, q WorkoutPageQuery, summary *WorkoutSummary) (WorkoutSeriesResult, error) {
	var out WorkoutSeriesResult
	if q.Limit == 0 {
		q.Limit = 1
	}
	if !validPage(q) || !validSection(q.Section) {
		return out, ErrWorkoutQuery
	}
	var wire struct {
		WorkoutID string `json:"workoutId"`
		Revision  string `json:"revision"`
		Start     string `json:"start"`
		Manifest  string `json:"manifest"`
		Pages     []struct {
			Index   int    `json:"index"`
			Section string `json:"section"`
			Sealed  string `json:"sealed"`
		}
		NextOffset *int `json:"nextOffset"`
	}
	v := url.Values{"uid": {c.uid}, "revision": {q.Revision}, "offset": {strconv.Itoa(q.Offset)}, "limit": {strconv.Itoa(q.Limit)}, "section": {q.Section}}
	if err := c.get(ctx, "/workout-details/"+q.WorkoutID, v, &wire); err != nil {
		return out, err
	}
	if wire.WorkoutID != q.WorkoutID || wire.Revision != q.Revision || wire.Pages == nil || len(wire.Pages) > q.Limit || !validNext(wire.NextOffset, q.Offset, len(wire.Pages)) {
		return out, ErrWorkoutData
	}
	var m WorkoutManifest
	if err := c.openWorkout(wire.Manifest, q.WorkoutID, q.Revision, "manifest", &m); err != nil {
		return out, err
	}
	if !validManifest(m, q.WorkoutID, wire.Start) {
		return out, ErrWorkoutData
	}
	if summary != nil {
		*summary = WorkoutSummary{q.WorkoutID, q.Revision, wire.Start, m}
	}
	if aggregation, ok := m.Aggregation[q.Section]; ok {
		out.Aggregation = &aggregation
	}
	total := m.Sections[q.Section]
	remaining := max(0, total-q.Offset)
	expected := min(q.Limit, remaining)
	if len(wire.Pages) != expected || (wire.NextOffset != nil) != (remaining > expected) {
		return out, ErrWorkoutData
	}

	out = WorkoutSeriesResult{Aggregation: out.Aggregation, WorkoutID: q.WorkoutID, Revision: q.Revision, Section: q.Section, Pages: make([]WorkoutPage, 0, len(wire.Pages)), NextOffset: wire.NextOffset}
	last := -1
	for _, p := range wire.Pages {
		if p.Index < 0 || p.Index >= 512 || p.Index <= last || p.Section != q.Section {
			return WorkoutSeriesResult{}, ErrWorkoutData
		}
		last = p.Index
		var data struct {
			Section string           `json:"section"`
			Items   []map[string]any `json:"items"`
		}
		if err := c.openWorkout(p.Sealed, q.WorkoutID, q.Revision, strconv.Itoa(p.Index), &data); err != nil {
			return WorkoutSeriesResult{}, err
		}
		if data.Section != q.Section || len(data.Items) == 0 || len(data.Items) > 256 {
			return WorkoutSeriesResult{}, ErrWorkoutData
		}
		for _, item := range data.Items {
			if !validWorkoutItem(q.Section, item) {
				return WorkoutSeriesResult{}, ErrWorkoutData
			}
			for _, key := range []string{"t", "end", "start"} {
				if offset, ok := item[key].(float64); ok && offset > m.ElapsedDuration+1 {
					return WorkoutSeriesResult{}, ErrWorkoutData
				}
			}
		}
		out.Pages = append(out.Pages, WorkoutPage{p.Index, p.Section, data.Items})
	}
	if q.Section == "events" {
		out.Unit = "seconds from workout start"
	} else if q.Section == "splitsKm" || q.Section == "splitsMi" {
		out.Unit = "distance m; times s; averageHeartRate bpm"
	}
	return out, workoutOutputBound(out)
}
func (c *WorkoutReader) GetWorkoutDetail(ctx context.Context, q WorkoutDetailQuery) (WorkoutDetailResult, error) {
	var out WorkoutDetailResult
	if q.Limit == 0 {
		q.Limit = 1
	}
	if q.SplitUnit == "" {
		q.SplitUnit = "km"
	}
	if (q.SplitUnit != "km" && q.SplitUnit != "mi") || q.Limit > 4 || !validPage(WorkoutPageQuery{WorkoutID: q.WorkoutID, Revision: q.Revision, Offset: q.Offset, Limit: q.Limit}) {
		return out, ErrWorkoutQuery
	}
	events, err := c.workoutPages(ctx, WorkoutPageQuery{WorkoutID: q.WorkoutID, Revision: q.Revision, Section: "events", Offset: q.Offset, Limit: q.Limit}, &out.Workout)
	if err != nil {
		return out, err
	}
	out.Sections = []WorkoutSeriesResult{events}
	split := "splitsKm"
	if q.SplitUnit == "mi" {
		split = "splitsMi"
	}
	if out.Workout.Manifest.Sections[split] > 0 {
		p, err := c.workoutPages(ctx, WorkoutPageQuery{WorkoutID: q.WorkoutID, Revision: q.Revision, Section: split, Offset: q.Offset, Limit: q.Limit}, nil)
		if err != nil {
			return WorkoutDetailResult{}, err
		}
		out.Sections = append(out.Sections, p)
	}
	return out, workoutOutputBound(out)
}

func validWorkoutItem(section string, item map[string]any) bool {
	var allowed map[string]bool
	if streamUnits[section] != "" {
		allowed = map[string]bool{"t": true, "end": true, "value": true, "kind": true, "min": true, "max": true, "count": true, "coverage": true, "breakBefore": true}
		if _, ok := item["t"].(float64); !ok {
			return false
		}
		if !validStreamItem(section, item) {
			return false
		}
	} else if section == "events" {
		allowed = map[string]bool{"kind": true, "start": true, "end": true}
		kind, _ := item["kind"].(string)
		if kind != "pause" && kind != "resume" && kind != "lap" && kind != "segment" && kind != "marker" {
			return false
		}
	} else {
		allowed = map[string]bool{"index": true, "distance": true, "start": true, "end": true, "elapsedDuration": true, "activeDuration": true, "averageHeartRate": true, "estimated": true}
		for _, field := range []string{"index", "distance", "elapsedDuration"} {
			n, ok := item[field].(float64)
			if !ok || n < 0 || (field == "index" && (n < 1 || n != math.Trunc(n))) {
				return false
			}
		}
		if _, ok := item["estimated"].(bool); !ok {
			return false
		}
	}
	if section == "events" || section == "splitsKm" || section == "splitsMi" {
		start, ok := item["start"].(float64)
		end, ok2 := item["end"].(float64)
		if !ok || !ok2 || start < 0 || end < start {
			return false
		}
	}
	if streamUnits[section] != "" {
		at := item["t"].(float64)
		if at < 0 {
			return false
		}
		if end, ok := item["end"].(float64); ok && end < at {
			return false
		}
	}
	for key, value := range item {
		if !allowed[key] {
			return false
		}
		if key != "kind" && key != "estimated" && key != "breakBefore" {
			if number, ok := value.(float64); !ok || math.IsNaN(number) || math.IsInf(number, 0) {
				return false
			}
		}
	}
	return true
}

func validManifest(m WorkoutManifest, id, start string) bool {
	a, e1 := time.Parse(time.RFC3339Nano, m.Start)
	indexStart, e3 := time.Parse(time.RFC3339Nano, start)
	b, e2 := time.Parse(time.RFC3339Nano, m.End)
	if m.SchemaVersion != 1 || m.WorkoutID != id || !a.Equal(indexStart) || e1 != nil || e2 != nil || e3 != nil || b.Before(a) || m.ActiveDuration < 0 || m.ElapsedDuration < 0 || m.Sections == nil || m.Availability == nil || m.Aggregation == nil {
		return false
	}
	switch m.Sport {
	case "running", "walking", "hiking", "cycling", "swimming", "other":
	default:
		return false
	}
	total := 0
	for section, count := range m.Sections {
		if !validSection(section) || count < 0 || count > 512 {
			return false
		}
		total += count
	}
	if total > 512 {
		return false
	}
	for section, count := range m.Sections {
		if streamUnits[section] != "" && count > 0 {
			if _, ok := m.Aggregation[section]; !ok {
				return false
			}
		}
	}
	for section, aggregation := range m.Aggregation {
		interval := 5
		if section == "heartRate" || section == "elevation" {
			interval = 10
		}
		method := "mean"
		if section == "distance" {
			method = "last"
		}
		if streamUnits[section] == "" || m.Sections[section] <= 0 || aggregation.IntervalSeconds != interval || aggregation.Method != method || (aggregation.Source != "healthkit" && (section != "speed" || aggregation.Source != "derivedDistance")) {
			return false
		}
	}
	for section, status := range m.Availability {
		if !validSection(section) || (status != "available" && status != "unavailable") {
			return false
		}
	}
	return true
}

func validStreamItem(section string, item map[string]any) bool {
	value, ok := item["value"].(float64)
	if !ok || (section != "elevation" && value < 0) {
		return false
	}
	at, ok := item["t"].(float64)
	if !ok {
		return false
	}
	end, hasEnd := item["end"].(float64)
	kind, _ := item["kind"].(string)
	count, hasCount := item["count"].(float64)
	if hasCount && (count < 1 || count != math.Trunc(count)) {
		return false
	}
	if b, exists := item["breakBefore"]; exists {
		if _, ok := b.(bool); !ok {
			return false
		}
	}
	coverage, hasCoverage := item["coverage"].(float64)
	switch kind {
	case "sample":
		if hasCount && count != 1 || hasCoverage {
			return false
		}
	case "endpoint":
		if section != "distance" || !hasCount || hasEnd && end != at || hasCoverage {
			return false
		}
	case "timeMean":
		if section == "distance" || !hasCount || !hasEnd || end <= at || !hasCoverage || coverage <= 0 || coverage > end-at+0.002 {
			return false
		}
	case "sampleMean":
		if section == "distance" || !hasCount || !hasEnd || end < at || hasCoverage {
			return false
		}
	default:
		return false
	}
	min, hasMin := item["min"].(float64)
	max, hasMax := item["max"].(float64)
	needsExtrema := (section == "heartRate" || section == "power") && (kind == "sampleMean" || kind == "timeMean")
	if needsExtrema {
		return hasMin && hasMax && min >= 0 && min <= value && value <= max
	}
	return !hasMin && !hasMax
}

// ListAllWorkouts follows index pagination. Offset pagination is not a snapshot:
// concurrent uploads can change the list; repeated identities are rejected.
func (c *WorkoutReader) ListAllWorkouts(ctx context.Context, from, to string) ([]WorkoutSummary, error) {
	a, _, err := parseDateValue(from)
	if err != nil {
		return nil, err
	}
	b, _, err := parseDateValue(to)
	if err != nil {
		return nil, err
	}
	if a.After(b) {
		a, b = b, a
	}
	q := WorkoutListQuery{Start: a.Format(time.RFC3339Nano), End: b.Format(time.RFC3339Nano), Limit: 10}
	all := []WorkoutSummary{}
	seen := map[string]bool{}
	for {
		result, err := c.ListWorkouts(ctx, q)
		if err != nil {
			return nil, err
		}
		for _, w := range result.Workouts {
			if seen[w.WorkoutID] {
				return nil, fmt.Errorf("%w: workout listing changed; retry", ErrWorkoutData)
			}
			seen[w.WorkoutID] = true
			all = append(all, w)
		}
		if result.NextOffset == nil {
			return all, nil
		}
		if *result.NextOffset > 10000 {
			return nil, fmt.Errorf("workout listing exceeds limit; select a shorter date range")
		}
		q.Offset = *result.NextOffset
	}
}

// WorkoutSection is a complete section, with availability separate from its rows.
type WorkoutSection struct {
	WorkoutID    string              `json:"workoutId"`
	Revision     string              `json:"revision"`
	Section      string              `json:"section"`
	Unit         string              `json:"unit"`
	Availability string              `json:"availability"`
	Aggregation  *WorkoutAggregation `json:"aggregation,omitempty"`
	Items        []map[string]any    `json:"items"`
}

func (c *WorkoutReader) FetchWorkoutSection(ctx context.Context, q WorkoutPageQuery) (WorkoutSection, error) {
	out, _, err := c.completeSection(ctx, q)
	return out, err
}
func (c *WorkoutReader) completeSection(ctx context.Context, q WorkoutPageQuery) (WorkoutSection, WorkoutSummary, error) {
	out := WorkoutSection{WorkoutID: q.WorkoutID, Revision: q.Revision, Section: q.Section, Items: []map[string]any{}}
	var first WorkoutSummary
	unit, factor := "", 1.0
	if streamUnits[q.Section] != "" {
		var err error
		unit, factor, err = unitFactor(q.Section, q.Unit)
		if err != nil {
			return out, first, err
		}
	} else if q.Unit != "" {
		return out, first, ErrWorkoutQuery
	}
	q.Offset, q.Limit = 0, 8
	last := -1
	for {
		var summary WorkoutSummary
		page, err := c.workoutPages(ctx, q, &summary)
		if err != nil {
			return WorkoutSection{}, WorkoutSummary{}, err
		}
		if q.Offset == 0 {
			first = summary
		} else if !reflect.DeepEqual(first, summary) {
			return WorkoutSection{}, WorkoutSummary{}, ErrWorkoutData
		}
		out.Unit, out.Aggregation = page.Unit, page.Aggregation
		if unit != "" {
			out.Unit = unit
		}
		out.Availability = first.Manifest.Availability[q.Section]
		if out.Availability == "" {
			out.Availability = "unavailable"
		}
		for _, p := range page.Pages {
			if p.Index <= last {
				return WorkoutSection{}, WorkoutSummary{}, ErrWorkoutData
			}
			last = p.Index
			for _, item := range p.Items {
				if factor != 1 {
					for _, field := range []string{"value", "min", "max"} {
						if v, ok := item[field].(float64); ok {
							item[field] = v * factor
						}
					}
				}
				out.Items = append(out.Items, item)
			}
		}
		if page.NextOffset == nil {
			return out, first, nil
		}
		q.Offset = *page.NextOffset
	}
}

var WorkoutSections = []string{"heartRate", "distance", "speed", "elevation", "power", "cadence", "events", "splitsKm", "splitsMi"}

// WorkoutExport is the same versioned, canonical-unit document as the web export.
type WorkoutExport struct {
	ExportVersion int                         `json:"exportVersion"`
	Description   string                      `json:"description"`
	Units         map[string]string           `json:"units"`
	Revision      string                      `json:"revision"`
	Manifest      WorkoutManifest             `json:"manifest"`
	Sections      map[string][]map[string]any `json:"sections"`
}

func (c *WorkoutReader) ExportWorkout(ctx context.Context, id, revision string) (WorkoutExport, error) {
	out := WorkoutExport{ExportVersion: 1, Description: "Stored workout detail measurements, including aggregation metadata. Not an export of all original HealthKit samples.", Revision: revision,
		Units: map[string]string{"timestamps": "ISO 8601", "offsets": "seconds from workout start", "durations": "seconds", "coverage": "seconds", "heartRate": "beats/minute", "distance": "meters", "speed": "meters/second", "elevation": "meters", "power": "watts", "cadence": "revolutions/minute", "splitDistance": "meters (both splitsKm and splitsMi)", "splitAverageHeartRate": "beats/minute", "aggregationInterval": "seconds"}, Sections: map[string][]map[string]any{}}
	// Events retrieves the manifest without downloading unrelated numeric streams.
	events, first, err := c.completeSection(ctx, WorkoutPageQuery{WorkoutID: id, Revision: revision, Section: "events"})
	if err != nil {
		return WorkoutExport{}, err
	}
	out.Manifest = first.Manifest
	out.Sections["events"] = events.Items
	for _, section := range WorkoutSections {
		if section == "events" {
			continue
		}
		out.Sections[section] = []map[string]any{}
		if first.Manifest.Sections[section] == 0 {
			continue
		}
		data, summary, err := c.completeSection(ctx, WorkoutPageQuery{WorkoutID: id, Revision: revision, Section: section})
		if err != nil {
			return WorkoutExport{}, err
		}
		if !reflect.DeepEqual(first, summary) {
			return WorkoutExport{}, ErrWorkoutData
		}
		out.Sections[section] = data.Items
	}
	return out, nil
}
