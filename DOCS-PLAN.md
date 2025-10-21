# CSV Documentation & UX Plan

## 1) Problem Statement
- Users need clear guidance on how to format CSV files accepted by the CLI, but we do not want this content in `README.md`.
- Current behavior is implicit in code (auto-detect delimiter/date/value/grouping) and surfaced only via errors.
- Discoverability is limited; there is no dedicated CLI entry to explain or validate a CSV before forecasting.

## 2) Goals
- Provide a canonical, versioned CSV specification outside `README.md`.
- Make the spec discoverable from the CLI (no browser or README needed).
- Let users self-validate their CSVs and see exactly what the tool detects.
- Offer a minimal, valid CSV template to get started quickly.
- Support advanced users with overrides and CI-friendly JSON outputs.

Non‑Goals
- Rework detection heuristics beyond simple refactors.
- Change external API behavior with the backend.

## 3) Current Behavior (from code)
Source: `internal/client/helpers.go`
- Header required; reads header + up to 10 sample rows.
- Delimiter auto-detected among `,`, `;`, `\t`.
- Date column detection:
  - Prefer columns whose name contains: `date`, `time`, `timestamp`, `datetime`, `datum` and whose first sample parses.
  - Fallback: first column whose first sample parses.
  - Supported formats: `YYYY-MM-DD`, `YYYY/MM/DD`, `DD.MM.YYYY`, `MM/DD/YYYY`, `YYYY-MM-DD HH:MM:SS`.
- Value columns: any non-date column where every non-empty sampled cell parses as float (decimal point `.`). Empty values permitted.
- Group columns: any remaining non-date, non-value columns.
- Errors if no data rows, no date column, or no numeric value columns.

Implication for users
- Required: a header row, at least one parsable date column, at least one numeric value column, at least one data row.
- Optional: grouping columns.

## 4) Best Practices (Research Summary)
- Keep README lean; place domain docs under `docs/` and reference from CLI help.
- Provide in-tool discoverability via dedicated subcommands (avoids context switching).
- Offer “doctor/inspect” command to explain detection and common errors before running the main workflow.
- Ship minimal working templates and realistic examples to accelerate adoption.
- Support overrides for edge cases and CI-friendly JSON outputs for automation.
- Keep documentation versioned with code; tests validate examples to prevent drift.

## 5) Deliverables
1. Documentation
   - `docs/csv.md`: canonical CSV spec with examples, detection rules, edge cases, and troubleshooting.
   - `examples/` folder with at least:
     - `examples/sales.csv` (date, product, customer, units)
     - `examples/minimal.csv` (minimal valid example)
2. CLI Discoverability
   - `swiftseer csv-spec` subcommand: prints or shows the CSV guide.
   - `swiftseer csv-template` subcommand: prints a minimal CSV template to stdout.
3. CSV Inspector
   - `swiftseer inspect <csv-file>`: reports delimiter, detected date column + format, value columns, group columns; emits actionable messages.
   - `--json` flag for machine-readable output (for CI).
4. Forecast Overrides (advanced)
   - Flags on `forecast`: `--date-column`, `--date-format`, `--value-columns`, `--group-columns`.
   - Respect overrides in check-in request builder.
5. Help Text Pointers
   - Update `forecast --help` usage to point to `swiftseer csv-spec` and `swiftseer inspect`.

## 6) Implementation Plan

Phase 1: Docs and Examples
- Add `docs/csv.md` describing:
  - Required vs optional columns
  - Detection logic and supported date formats
  - Delimiters, encoding (`utf-8`), decimals (`.`)
  - Example CSVs and common pitfalls
- Add `examples/sales.csv` and `examples/minimal.csv`.

Phase 2: Discoverability Subcommands
- Add `cmd/csv_spec.go`:
  - Prints `docs/csv.md` to stdout (or prints a short pointer if file missing).
  - Accepts `--pager` (optional later) to open in `$PAGER`.
- Add `cmd/csv_template.go`:
  - Writes a minimal valid template to stdout; supports `--with-groups`.

Phase 3: Inspector
- Add `cmd/inspect.go`:
  - Reuse detection functions from `internal/client/helpers.go`.
  - Print a human-readable report; on `--json` output structured JSON.
  - Exit non-zero on validation errors; zero otherwise.
- Consider small refactor to expose a pure function, e.g. `AnalyzeCSV(path) (Report, error)` in a new internal package or in `internal/client` if appropriate.

Phase 4: Forecast Overrides
- Extend `cmd/forecast_flags.go` to parse new flags:
  - `--date-column string`
  - `--date-format string`
  - `--value-columns string` (comma-separated)
  - `--group-columns string` (comma-separated)
- Extend check-in builder to accept optional overrides:
  - Add a new builder type that takes optional params and bypasses auto-detection when provided.

Phase 5: Tests and CI
- Unit tests for new flag parsing and inspector output (golden tests for JSON).
- Example CSVs validated in tests to avoid drift.

## 7) Detailed File Map (proposed)
- `docs/csv.md` — canonical CSV spec.
- `examples/sales.csv` — realistic example.
- `examples/minimal.csv` — minimal valid example.
- `cmd/csv_spec.go` — implements `swiftseer csv-spec`.
- `cmd/csv_template.go` — implements `swiftseer csv-template`.
- `cmd/inspect.go` — implements `swiftseer inspect`.
- `internal/csv/inspect.go` (optional) — `AnalyzeCSV` helper to share logic with CLI.
- `cmd/forecast_flags.go` — add override flags.
- `internal/workflow/checkin_builder.go` + new builder variant — accept overrides.

## 8) UX Notes
- Help strings: keep concise, add pointers: “CSV guide: run `swiftseer csv-spec`”.
- Inspector error messages should suggest either fixing the CSV or using override flags.
- `csv-template` should default to the most common, robust format: comma delimiter, `YYYY-MM-DD` dates, `.` decimals, monthly cadence.

## 9) Rollout Plan
- Ship docs + inspector + spec/template in a minor release.
- In release notes, emphasize new commands and how to validate CSVs.
- Keep README unchanged; add a single pointer in help output only.

## 10) Definition of Done
- `docs/csv.md` exists and is discoverable via `swiftseer csv-spec`.
- `swiftseer inspect` validates and reports detection; supports `--json`.
- `swiftseer csv-template` prints a minimal valid CSV.
- `forecast --help` references the new commands.
- Tests cover inspector JSON and flag parsing. Examples validated in CI.

## 11) Open Questions
- Should `csv-spec` open in a pager by default or just print?
- Should `inspect` optionally check full file (not just first 10 rows) for numeric columns? Trade-off: performance vs accuracy.
- Do we want to support locale-specific decimals (`,`) in future? Would require a `--decimal` override or smarter detection.

