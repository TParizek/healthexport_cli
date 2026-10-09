package cmd

import (
	"fmt"
	"strings"

	"github.com/TParizek/healthexport_cli/internal/output"
	"github.com/TParizek/healthexport_cli/internal/service"
	"github.com/spf13/cobra"
)

func init() { rootCmd.AddCommand(newWorkoutsCommand()) }

func newWorkoutsCommand() *cobra.Command {
	parent := &cobra.Command{Use: "workouts", Short: "Read and export uploaded workout details", Long: "Read uploaded workout details and decrypt them locally. Stored measurements may be aggregated; this does not sync HealthKit or include summary-only workouts. Use 'he data --type 26' for legacy summaries."}
	var from, to, format string
	list := &cobra.Command{Use: "list", Short: "List workouts with uploaded details (no sample streams)", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		resolved := resolveOutputFormat(format)
		if err := workoutFormat(resolved); err != nil {
			return err
		}
		reader, err := service.NewWorkoutReader(service.Options{AccountKey: accountKey, APIURL: apiURL})
		if err != nil {
			return err
		}
		workouts, err := reader.ListAllWorkouts(cmd.Context(), from, to)
		if err != nil {
			return err
		}
		return output.FormatWorkoutList(cmd.OutOrStdout(), resolved, workouts)
	}}
	list.Flags().StringVarP(&from, "from", "f", "", "Inclusive start: YYYY-MM-DD (UTC midnight) or RFC3339 (at most millisecond precision)")
	list.Flags().StringVarP(&to, "to", "T", "", "Inclusive end: YYYY-MM-DD (UTC midnight) or RFC3339; range <=366 days; reversed dates are swapped")
	list.Flags().StringVar(&format, "format", "", "Output format: csv or json (defaults to config)")
	_ = list.MarkFlagRequired("from")
	_ = list.MarkFlagRequired("to")
	_ = list.RegisterFlagCompletionFunc("format", cobra.FixedCompletions([]cobra.Completion{"csv", "json"}, cobra.ShellCompDirectiveNoFileComp))
	list.Example = "  he workouts list -f 2026-10-01 -T 2026-10-10 --format json"
	parent.AddCommand(list)
	for _, kind := range []string{"export", "series", "splits", "events"} {
		parent.AddCommand(newWorkoutReadCommand(kind))
	}
	return parent
}
func workoutFormat(format string) error {
	if format != "csv" && format != "json" {
		return fmt.Errorf("%w: unsupported format %q (use csv or json)", service.ErrInvalidInput, format)
	}
	return nil
}
func newWorkoutReadCommand(kind string) *cobra.Command {
	var revision, section, unit, splitUnit, format string
	descriptions := map[string]string{"export": "Export the complete workout as JSON in canonical units (always JSON, independent of configured format)", "series": "Read all stored points for one numeric measurement", "splits": "Read recorded kilometer or mile splits; distances remain meters", "events": "Read recorded pauses, resumes, laps, segments and markers"}
	cmd := &cobra.Command{Use: kind + " <workout-id>", Short: descriptions[kind], Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("%w: supply exactly one workout ID from workouts list", service.ErrInvalidInput)
		}
		return nil
	}, RunE: func(cmd *cobra.Command, args []string) error {
		resolved := resolveOutputFormat(format)
		if kind != "export" {
			if err := workoutFormat(resolved); err != nil {
				return err
			}
		}
		if kind == "splits" && splitUnit != "km" && splitUnit != "mi" {
			return fmt.Errorf("%w: --split-unit must be km or mi", service.ErrInvalidInput)
		}
		reader, err := service.NewWorkoutReader(service.Options{AccountKey: accountKey, APIURL: apiURL})
		if err != nil {
			return err
		}
		if kind == "export" {
			data, err := reader.ExportWorkout(cmd.Context(), args[0], revision)
			if err != nil {
				return err
			}
			return output.FormatWorkoutExport(cmd.OutOrStdout(), data)
		}
		if kind == "events" {
			section = "events"
		}
		if kind == "splits" {
			section = "splitsKm"
			if splitUnit == "mi" {
				section = "splitsMi"
			}
		}
		if kind == "series" && !strings.Contains("|heartRate|distance|speed|elevation|power|cadence|", "|"+section+"|") {
			return fmt.Errorf("%w: unsupported numeric section", service.ErrInvalidInput)
		}
		data, err := reader.FetchWorkoutSection(cmd.Context(), service.WorkoutPageQuery{WorkoutID: args[0], Revision: revision, Section: section, Unit: unit})
		if err != nil {
			return err
		}
		if len(data.Items) == 0 {
			if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "No stored %s measurements (%s).\n", section, data.Availability); err != nil {
				return err
			}
		}
		return output.FormatWorkoutSection(cmd.OutOrStdout(), resolved, data)
	}}
	examples := map[string]string{
		"export": "  he workouts export <workout-id> --revision <revision> > workout.json",
		"series": "  he workouts series <workout-id> --revision <revision> --section heartRate --format csv > heart-rate.csv",
		"splits": "  he workouts splits <workout-id> --revision <revision> --split-unit km --format csv > splits.csv",
		"events": "  he workouts events <workout-id> --revision <revision> --format json",
	}
	cmd.Example = examples[kind]
	cmd.Flags().StringVar(&revision, "revision", "", "Revision returned by workouts list; refresh the list after a new upload")
	_ = cmd.MarkFlagRequired("revision")
	if kind != "export" {
		cmd.Flags().StringVar(&format, "format", "", "Output format: csv or json (defaults to config)")
		_ = cmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions([]cobra.Completion{"csv", "json"}, cobra.ShellCompDirectiveNoFileComp))
	}
	if kind == "series" {
		cmd.Flags().StringVar(&section, "section", "", "Measurement: heartRate, distance, speed, elevation, power or cadence")
		cmd.Flags().StringVar(&unit, "unit", "", "Output unit (default canonical): bpm, m, m/s, W, rpm; conversions km, mi, ft, km/h, mph where compatible")
		_ = cmd.MarkFlagRequired("section")
		_ = cmd.RegisterFlagCompletionFunc("section", cobra.FixedCompletions([]cobra.Completion{"heartRate", "distance", "speed", "elevation", "power", "cadence"}, cobra.ShellCompDirectiveNoFileComp))
		_ = cmd.RegisterFlagCompletionFunc("unit", cobra.FixedCompletions([]cobra.Completion{"bpm", "m", "m/s", "W", "rpm", "km", "mi", "ft", "km/h", "mph"}, cobra.ShellCompDirectiveNoFileComp))
	}
	if kind == "splits" {
		cmd.Flags().StringVar(&splitUnit, "split-unit", "km", "Select recorded splits: km or mi (distance values stay in meters)")
		_ = cmd.RegisterFlagCompletionFunc("split-unit", cobra.FixedCompletions([]cobra.Completion{"km", "mi"}, cobra.ShellCompDirectiveNoFileComp))
	}
	return cmd
}
