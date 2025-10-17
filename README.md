<div align="center">

<picture>
  <source media="(prefers-color-scheme: dark)" srcset=".github/assets/logo.png">
  <source media="(prefers-color-scheme: light)" srcset=".github/assets/logo.png">
  <img alt="swiftseer" src=".github/assets/logo.png" width="400">
</picture>

**run plain batch forecasts on future platform (unofficial)**

[![CI](https://github.com/deichbewohner/swiftseer/actions/workflows/ci.yml/badge.svg?branch=release)](https://github.com/deichbewohner/swiftseer/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/deichbewohner/swiftseer)](https://goreportcard.com/report/github.com/deichbewohner/swiftseer)
[![Go Version](https://img.shields.io/github/go-mod/go-version/deichbewohner/swiftseer)](https://github.com/deichbewohner/swiftseer/blob/release/go.mod)

</div>

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
