# Swiftseer (Unofficial futureEXPERT CLI)

A tiny Go CLI to run plain batch forecasts on the future platform from a CSV:
upload -> check‑in -> start forecast -> poll -> save results.

> Unofficial tool. Not endorsed by future/futureEXPERT.

## Why

I often need simple forecasts to kick off projects. This CLI lets me start
immediately from the terminal (or CI) without setting up a Python stack.

## Use futureEXPERT for more

For advanced workflows (covariates, analytics, pandas exports, plotting, rich
configs), use the [official Python
client](https://discovertomorrow.github.io/futureEXPERT/).

## Quick start
- Build: `make build` (Go 1.25+). Binary: `bin/swiftseer`
- Login: `swiftseer login --user <name> --group <group>`
- Forecast: `swiftseer forecast data.csv` (flags: `--horizon`, `--confidence`,
  `--output`)
- Resume: `swiftseer forecast --report-id <id>`

Use `--no-ui`/`--json-events` for CI.
