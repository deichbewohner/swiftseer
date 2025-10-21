# CSV Input Specification

This CLI accepts a single CSV file as input. The tool auto‑detects structure, but a minimal shape is required.

## Required
- Header row with column names
- One date column with values in one of:
  - `YYYY-MM-DD`
  - `YYYY/MM/DD`
  - `DD.MM.YYYY`
  - `MM/DD/YYYY`
  - `YYYY-MM-DD HH:MM:SS`
- At least one numeric value column (floats use `.` as decimal)
- At least one data row after the header

## Optional
- Any number of grouping columns (categorical dimensions)
- Delimiter: auto‑detected among comma `,`, semicolon `;`, or tab `\t`
- Encoding: `utf-8`

## Detection Rules (summary)
- Date column:
  - Prefer columns named like `date`, `time`, `timestamp`, `datetime`, `datum`
  - Otherwise the first column whose first sample parses as a date
- Value columns:
  - Any non‑date column where every non‑empty sampled cell parses as a float
  - Empty cells allowed; non‑numeric text disqualifies a column
- Group columns:
  - Remaining non‑date, non‑value columns

## Minimal Example
```csv
date,units
2020-11-01,77383
2020-12-01,75628
2021-01-01,72231
```

## Grouped Example
```csv
date,product,customer,units
2020-11-01,Liquid Soap,REWE Group,77383
2020-12-01,Liquid Soap,REWE Group,75628
2021-01-01,Liquid Soap,REWE Group,72231
```

## Tips
- Keep date values consistent in one format.
- Use `.` for decimals; thousands separators are not supported in numeric columns.
- If detection struggles, run `swiftseer inspect <file>` to see what the tool detects.
