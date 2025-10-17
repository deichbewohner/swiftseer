package main

import (
	"fmt"
	"os"

	"github.com/deichbewohner/swiftseer/cmd"
	"github.com/deichbewohner/swiftseer/internal/version"
)

func main() {
    if len(os.Args) < 2 {
        printUsage()
        os.Exit(1)
    }

    subcommand := os.Args[1]
    args := os.Args[2:]

    var err error
    switch subcommand {
	case "login":
		err = cmd.LoginCmd(args)
	case "logout":
		err = cmd.LogoutCmd(args)
	case "status":
		err = cmd.StatusCmd(args)
	case "forecast":
		err = cmd.ForecastCmd(args)
	case "version", "--version", "-v":
		fmt.Printf("swiftseer version %s\n", version.GetFullVersion())
		return
	case "help", "--help", "-h":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown subcommand '%s'\n\n", subcommand)
		printUsage()
		os.Exit(1)
	}

    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}

func printUsage() {
	fmt.Printf(`swiftseer %s - Unofficial CLI for futureEXPERT

Usage:
  swiftseer <command> [flags]

Commands:
  login       Authenticate and save credentials
  forecast    Generate forecast from CSV file
  status      Show configuration and token status
  logout      Clear saved credentials
  version     Show version information

Use "swiftseer <command> --help" for command-specific options.

Examples:
  swiftseer login --user Analyst1 --group group-expert
  swiftseer forecast data.csv
  swiftseer forecast --horizon 18 --confidence 0.8 data.csv
  swiftseer status

Config: ~/.config/swiftseer/config.yaml
`, version.GetVersion())
}
