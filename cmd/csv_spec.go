package cmd

import (
	"fmt"

	"github.com/deichbewohner/swiftseer/internal/docs"
)

func CsvSpecCmd(args []string) error {
	if docs.CSVSpec != "" {
		fmt.Print(docs.CSVSpec)
		return nil
	}
	return fmt.Errorf("embedded CSV spec not found")
}
