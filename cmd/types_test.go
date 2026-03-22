package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TParizek/healthexport_cli/internal/api"
	"github.com/TParizek/healthexport_cli/internal/config"
	"github.com/spf13/cobra"
)

func TestTypesHelp(t *testing.T) {
	restore := snapshotTypesState()
	t.Cleanup(restore)
	setCmdConfigHome(t)

	rootCmd.SetArgs([]string{"types", "--help"})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := stdout.String() + stderr.String()
	assertContains(t, output, "List the health data types exposed by the API.")
	assertContains(t, output, "Authentication is not required.")
	assertContains(t, output, "he types --category aggregated")
}

func TestRunTypesUsesConfigDefaultsFiltersCategoryAndSortsByID(t *testing.T) {
	restore := snapshotTypesState()
	t.Cleanup(restore)
	setCmdConfigHome(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthtypes":
			writeJSONResponse(t, w, &api.HealthTypesResponse{
				Aggregated: []api.HealthTypeSection{
					{
						Name: "Activity",
						Types: []api.HealthType{
							{ID: 9, Name: "Step count", Category: "Cumulative", SubCategory: "Activity"},
						},
					},
				},
				Record: []api.HealthTypeSection{
					{
						Name: "Body",
						Types: []api.HealthType{
							{ID: 24, Name: "Time in bed", Category: "Record", SubCategory: "Sleep"},
							{ID: 0, Name: "Body mass", Category: "Record", SubCategory: "Body"},
						},
					},
					{
						Name: "Heart rate",
						Types: []api.HealthType{
							{ID: 52, Name: "Heart rate", Category: "Record", SubCategory: "Heart rate"},
						},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	cfg := &config.Config{
		Format: "json",
		APIURL: server.URL,
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	typesFormat = ""
	typesCategory = "record"

	var stdout bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)

	if err := runTypes(cmd, nil); err != nil {
		t.Fatalf("runTypes() error = %v", err)
	}

	want := "" +
		"[\n" +
		"  {\n" +
		"    \"id\": 0,\n" +
		"    \"name\": \"Body mass\",\n" +
		"    \"category\": \"record\",\n" +
		"    \"subcategory\": \"Body\"\n" +
		"  },\n" +
		"  {\n" +
		"    \"id\": 24,\n" +
		"    \"name\": \"Time in bed\",\n" +
		"    \"category\": \"record\",\n" +
		"    \"subcategory\": \"Sleep\"\n" +
		"  },\n" +
		"  {\n" +
		"    \"id\": 52,\n" +
		"    \"name\": \"Heart rate\",\n" +
		"    \"category\": \"record\",\n" +
		"    \"subcategory\": \"Heart rate\"\n" +
		"  }\n" +
		"]\n"

	if got := stdout.String(); got != want {
		t.Fatalf("runTypes() output = %q, want %q", got, want)
	}
}

func TestRunTypesRejectsInvalidCategory(t *testing.T) {
	restore := snapshotTypesState()
	t.Cleanup(restore)

	typesFormat = ""
	typesCategory = "invalid"

	err := runTypes(&cobra.Command{}, nil)
	if err == nil {
		t.Fatal("runTypes() error = nil, want invalid category error")
	}

	if !shouldPrintError(err) {
		t.Fatal("shouldPrintError() = false, want true")
	}

	if got := exitCodeForError(err); got != 4 {
		t.Fatalf("exitCodeForError() = %d, want 4", got)
	}

	assertContains(t, err.Error(), "aggregated, record, workout")
}

func snapshotTypesState() func() {
	prevFormat := typesFormat
	prevCategory := typesCategory
	restoreRoot := snapshotVersionState()

	return func() {
		typesFormat = prevFormat
		typesCategory = prevCategory
		restoreRoot()
	}
}
