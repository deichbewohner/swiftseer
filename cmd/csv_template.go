package cmd

import (
	"flag"
	"fmt"
)

func CsvTemplateCmd(args []string) error {
	fs := flag.NewFlagSet("csv-template", flag.ContinueOnError)
	withGroups := fs.Bool("with-groups", false, "Include example grouping columns")
	fs.Usage = func() {
		fmt.Printf(`Print a minimal CSV template to stdout

Usage:
  swiftseer csv-template [--with-groups]
`)
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *withGroups {
		fmt.Print("date,product,customer,units\n")
		fmt.Print("2020-11-01,Liquid Soap,REWE Group,77383\n")
		fmt.Print("2020-12-01,Liquid Soap,REWE Group,75628\n")
		fmt.Print("2021-01-01,Liquid Soap,REWE Group,72231\n")
		return nil
	}
	fmt.Print("date,units\n")
	fmt.Print("2020-11-01,77383\n")
	fmt.Print("2020-12-01,75628\n")
	fmt.Print("2021-01-01,72231\n")
	return nil
}
