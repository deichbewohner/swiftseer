package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/deichbewohner/swiftseer/internal/client"
)

type inspectReport struct {
	Delimiter  string   `json:"delimiter"`
	Decimal    string   `json:"decimal"`
	Encoding   string   `json:"encoding"`
	DateColumn string   `json:"date_column"`
	DateFormat string   `json:"date_format"`
	ValueCols  []string `json:"value_columns"`
	GroupCols  []string `json:"group_columns"`
}

func InspectCmd(args []string) error {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "Emit report as JSON")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Analyze a CSV and report detected columns

Usage:
  swiftseer inspect [--json] <csv-file>
`)
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("missing required argument: <csv-file>")
	}
	path := fs.Arg(0)

	analysis, err := client.AnalyzeCSV(path)
	if err != nil {
		return err
	}

	r := inspectReport{
		Delimiter:  analysis.Delimiter,
		Decimal:    ".",
		Encoding:   "utf-8",
		DateColumn: analysis.DateColumn,
		DateFormat: analysis.DateFormat,
		ValueCols:  analysis.ValueCols,
		GroupCols:  analysis.GroupCols,
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(r)
	}

	fmt.Printf("Delimiter: %q\n", r.Delimiter)
	fmt.Printf("Encoding: %s\n", r.Encoding)
	fmt.Printf("Decimal: %s\n", r.Decimal)
	fmt.Printf("Date column: %s\n", r.DateColumn)
	fmt.Printf("Date format: %s\n", r.DateFormat)
	fmt.Printf("Value columns: %v\n", r.ValueCols)
	fmt.Printf("Group columns: %v\n", r.GroupCols)
	return nil
}
