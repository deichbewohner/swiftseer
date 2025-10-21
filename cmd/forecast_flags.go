package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type forecastOptions struct {
	Horizon    int
	Confidence float64
	Output     string
	Title      string
	Verbose    bool
	ReportID   int
	Clean      bool
	NoUI       bool
	JSONEvents bool
	CSVPath    string
	DateColumn string
	DateFormat string
	ValueCols  []string
	GroupCols  []string
}

func parseForecastFlags(args []string) (*forecastOptions, error) {
	fs := flag.NewFlagSet("forecast", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Generate forecast from CSV file

Usage:
  swiftseer forecast [flags] <csv-file>
  swiftseer forecast --report-id <id> [flags]

Flags:
  --horizon       Forecast horizon in periods (default: 12)
  --confidence    Confidence level 0.0-1.0 (default: 0.75)
  --output, -o    Output file path (default: forecast-results.json)
  --title         Forecast title (default: "CLI Forecast")
  --verbose, -v   Verbose logging
  --report-id     Resume polling existing forecast by report ID
  --clean         Delete upload and report after completion
  --no-ui         Force non-interactive mode (no TUI)
  --json-events   Emit workflow events as NDJSON on stdout

CSV help:
  swiftseer csv-spec        # CSV guide
  swiftseer inspect file.csv # Analyze detection
  swiftseer csv-template    # Print a minimal template

Examples:
  # Basic forecast
  swiftseer forecast data.csv

  # Custom settings
  swiftseer forecast --horizon 18 --confidence 0.8 --output results.json data.csv

  # Resume existing forecast
  swiftseer forecast --report-id 12345

  # Non-interactive mode (CI/CD)
  swiftseer forecast --no-ui --json-events data.csv
`)
	}
	var (
		horizon    int
		confidence float64
		output     string
		title      string
		verbose    bool
		reportID   int
		clean      bool
		noUI       bool
		jsonEvents bool
		dateColumn string
		dateFormat string
		valueCols  string
		groupCols  string
	)
	fs.IntVar(&horizon, "horizon", 12, "Forecast horizon in periods")
	fs.Float64Var(&confidence, "confidence", 0.75, "Confidence level (0.0-1.0)")
	fs.StringVar(&output, "output", "forecast-results.json", "Output file path")
	fs.StringVar(&output, "o", "forecast-results.json", "Output file path (short form)")
	fs.StringVar(&title, "title", "CLI Forecast", "Forecast title")
	fs.BoolVar(&verbose, "verbose", false, "Verbose logging")
	fs.BoolVar(&verbose, "v", false, "Verbose logging (short form)")
	fs.IntVar(&reportID, "report-id", 0, "Resume polling existing forecast by report ID")
	fs.BoolVar(&clean, "clean", false, "Delete upload and report after completion")
	fs.BoolVar(&noUI, "no-ui", false, "Force non-interactive mode (no TUI)")
	fs.BoolVar(&jsonEvents, "json-events", false, "Emit workflow events as NDJSON on stdout")
	fs.StringVar(&dateColumn, "date-column", "", "Override date column name")
	fs.StringVar(&dateFormat, "date-format", "", "Override date format (e.g. %Y-%m-%d)")
	fs.StringVar(&valueCols, "value-columns", "", "Comma-separated value column names")
	fs.StringVar(&groupCols, "group-columns", "", "Comma-separated group column names")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		return nil, fmt.Errorf("failed to parse flags: %w", err)
	}

	var csvPath string
	if reportID == 0 {
		if fs.NArg() != 1 {
			fs.Usage()
			return nil, fmt.Errorf("missing required argument: <csv-file>")
		}
		csvPath = fs.Arg(0)
		if horizon <= 0 {
			return nil, fmt.Errorf("horizon must be positive")
		}
		if confidence <= 0 || confidence >= 1 {
			return nil, fmt.Errorf("confidence must be between 0 and 1")
		}
	} else {
		if fs.NArg() > 0 {
			return nil, fmt.Errorf("CSV file should not be provided when using --report-id")
		}
	}

	// Parse lists
	var valueList, groupList []string
	if valueCols != "" {
		for _, s := range strings.Split(valueCols, ",") {
			v := strings.TrimSpace(s)
			if v != "" {
				valueList = append(valueList, v)
			}
		}
	}
	if groupCols != "" {
		for _, s := range strings.Split(groupCols, ",") {
			v := strings.TrimSpace(s)
			if v != "" {
				groupList = append(groupList, v)
			}
		}
	}

	return &forecastOptions{
		Horizon:    horizon,
		Confidence: confidence,
		Output:     output,
		Title:      title,
		Verbose:    verbose,
		ReportID:   reportID,
		Clean:      clean,
		NoUI:       noUI,
		JSONEvents: jsonEvents,
		CSVPath:    csvPath,
		DateColumn: dateColumn,
		DateFormat: dateFormat,
		ValueCols:  valueList,
		GroupCols:  groupList,
	}, nil
}
