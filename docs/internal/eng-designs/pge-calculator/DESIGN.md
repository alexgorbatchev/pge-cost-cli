# Engineering Design: PG&E Cost Calculator

| Metadata | Value |
|---|---|
| Created On | 2026-06-03 |
| Last Modified | 2026-06-03 |
| Status | Current |

## 1. Objective and Non-Goals
This project implements a Go CLI utility (`pge-cost`) to calculate the cost of running an electrical device 24/7 on various Pacific Gas & Electric (PG&E) residential rate plans.

### Objectives
- Calculate daily, monthly, and annual costs for a device running 24/7 at a specified wattage.
- Support standard residential rate plans: `E-1` (Tiered), `E-TOU-C` (Time-of-Use Everyday 4-9 PM), `E-TOU-D` (Time-of-Use Weekdays 5-8 PM), `EV2` (Electric Vehicle), and `E-ELEC` (Electric Home).
- Support `EV-B` (Electric Vehicle Metered Option B) for completeness.
- Read rates from a structured, easily updatable `rates.json` file.
- Account for variable day rates, seasons (Summer vs. Winter), weekdays vs. weekends, and legal holidays.
- Provide a subcommand to automatically update rates by parsing PG&E's official published rates sheet.

### Non-Goals
- Supporting commercial, industrial, or agricultural rate plans beyond basic general service.
- Tracking real-time fluctuating spot prices (PG&E rates are fixed and approved by the CPUC periodically).
- Supporting multi-tiered net energy metering (solar feedback calculations) beyond direct consumption.

## 2. Current Codebase Baseline
The repository is currently empty. We are starting with a completely clean workspace.

## 3. Non-Negotiable Constraints
- **Red/Green testing**: All code must have corresponding tests and follow a strict test-first or test-concurrent implementation.
- **Accurate calculations**: TOU periods, weekday vs. weekend, and holiday splits must match PG&E's actual 2026 calendar-based weights exactly.
- **Modularity**: The parsing/calculation engine must be separate from the CLI interface so it can be run as a script or embedded.
- **Ease of update**: Rates must not be hardcoded in Go source files; they must reside in a separate JSON file (`rates.json`) that can be edited by hand or updated via a command.

## 4. Exact Architecture Choice
The application will be a single Go module with two main components:
1. **Core Domain Library**: Implements the rate structure parser, the calendar weight calculator, and the cost calculator.
2. **CLI Entrypoint**: Implements the command-line flags and subcommands using Go's standard library `flag` package (no external CLI framework like cobra unless necessary, in keeping with Go standard-library-first conventions).

For updating the rates, we will provide a built-in subcommand `pge-cost fetch` which downloads the current `res-inclu-tou-current.xlsx` from PG&E's website, parses it, and updates the local `rates.json` file.
Wait, parsing `.xlsx` in Go requires an external dependency, `github.com/xuri/excelize/v2`, which is the industry standard for Go Excel parsing. This is a justified external dependency.

### Alternatives Rejected
- **Scraping HTML directly**: Rejected because PG&E's website has severe Akamai bot protection on consumer pages, making automated HTML scraping fragile and prone to blacklisting. Downloading their raw tariff spreadsheet from their static assets server is much more reliable and contains the absolute canonical figures.
- **Hardcoded constants**: Rejected because PG&E rates change multiple times per year.

## 5. Data Model / Schema
We will store the rates in `rates.json`. The JSON schema is defined as:

```json
{
  "last_updated": "2026-06-03",
  "plans": {
    "E-1": {
      "name": "Residential Services (Tiered)",
      "tier_1": 0.32561,
      "tier_2": 0.40702
    },
    "E-TOU-C": {
      "name": "Residential Time-Of-Use (4-9 PM Everyday)",
      "summer": { "peak": 0.52240, "off_peak": 0.39940 },
      "winter": { "peak": 0.39757, "off_peak": 0.36757 }
    },
    "E-TOU-D": {
      "name": "Residential Time-Of-Use (5-8 PM Weekdays)",
      "summer": { "peak": 0.47708, "off_peak": 0.34212 },
      "winter": { "peak": 0.38747, "off_peak": 0.34886 }
    },
    "EV2": {
      "name": "Residential EV / Battery Storage (Everyday)",
      "summer": { "peak": 0.53809, "partial_peak": 0.42760, "off_peak": 0.22558 },
      "winter": { "peak": 0.41099, "partial_peak": 0.39428, "off_peak": 0.22558 }
    },
    "E-ELEC": {
      "name": "Residential Electric Home (Everyday)",
      "summer": { "peak": 0.55214, "partial_peak": 0.39026, "off_peak": 0.33358 },
      "winter": { "peak": 0.32063, "partial_peak": 0.29854, "off_peak": 0.28468 }
    },
    "EV-B": {
      "name": "Residential EV Option B (Weekdays/Weekends)",
      "summer": { "peak": 0.62131, "partial_peak": 0.37720, "off_peak": 0.26465 },
      "winter": { "peak": 0.43878, "partial_peak": 0.30677, "off_peak": 0.23504 }
    }
  }
}
```

## 6. Types and Contracts
The internal calculation data structures in Go:

```go
package domain

type SeasonRates struct {
	Peak        float64 `json:"peak"`
	PartialPeak float64 `json:"partial_peak,omitempty"`
	OffPeak     float64 `json:"off_peak"`
}

type RatePlan struct {
	Name   string      `json:"name"`
	Tier1  float64     `json:"tier_1,omitempty"`
	Tier2  float64     `json:"tier_2,omitempty"`
	Summer SeasonRates `json:"summer,omitempty"`
	Winter SeasonRates `json:"winter,omitempty"`
}

type RatesDatabase struct {
	LastUpdated string              `json:"last_updated"`
	Plans       map[string]RatePlan `json:"plans"`
}
```

## 7. Exact File Plan

### Add
- `go.mod`: Mod initialization.
- `rates.json`: The database of PG&E rates.
- `pkg/domain/types.go`: Core domain structures.
- `pkg/domain/calculator.go`: Math and weight equations for calculating costs 24/7.
- `pkg/domain/calculator_test.go`: Red/green test suite.
- `pkg/domain/parser.go`: Excel downloader and parser.
- `pkg/domain/parser_test.go`: Test parsing and saving.
- `main.go`: CLI command wrapper, executes the calculation based on flags.

## 8. Runtime Behavior
When the user executes `pge-cost --watts 150 --plan E-TOU-C`, the program:
1. Loads `rates.json` from the current directory or executable directory.
2. Looks up the `E-TOU-C` plan.
3. Computes the weighted average rate using the following 2026 calendar constants:
   - **June-September (122 days)**:
     - 5 peak hours everyday.
     - 19 off-peak hours everyday.
   - **October-May (243 days)**:
     - 5 peak hours everyday.
     - 19 off-peak hours everyday.
4. Performs the calculations:
   - Daily kWh = $150 \times 24 / 1000 = 3.6$ kWh.
   - Daily Cost = Daily kWh $\times$ Weighted Average Rate.
   - Monthly Cost = Daily Cost $\times (365 / 12)$.
   - Annual Cost = Daily Cost $\times 365$.
5. Prints the results in a clean table.

### Calendar Weights (2026 Non-Leap Year)

#### For E-TOU-C:
- Summer Peak hours: $122 \times 5 = 610$
- Summer Off-Peak hours: $122 \times 19 = 2318$
- Winter Peak hours: $243 \times 5 = 1215$
- Winter Off-Peak hours: $243 \times 19 = 4617$
- **Weighted Avg Rate Formula**:
  $$\text{Rate}_{\text{avg}} = \frac{610 \cdot \text{SummerPeak} + 2318 \cdot \text{SummerOff} + 1215 \cdot \text{WinterPeak} + 4617 \cdot \text{WinterOff}}{8760}$$

#### For E-TOU-D:
- Summer Peak hours (weekdays only, no holidays): $86 \times 3 = 258$
- Summer Off-Peak hours: $122 \times 24 - 258 = 2670$
- Winter Peak hours (weekdays only, no holidays): $167 \times 3 = 501$
- Winter Off-Peak hours: $243 \times 24 - 501 = 5331$
- **Weighted Avg Rate Formula**:
  $$\text{Rate}_{\text{avg}} = \frac{258 \cdot \text{SummerPeak} + 2670 \cdot \text{SummerOff} + 501 \cdot \text{WinterPeak} + 5331 \cdot \text{WinterOff}}{8760}$$

#### For EV2 / E-ELEC:
- Summer Peak hours: $122 \times 5 = 610$
- Summer Partial-Peak hours: $122 \times 4 = 488$
- Summer Off-Peak hours: $122 \times 15 = 1830$
- Winter Peak hours: $243 \times 5 = 1215$
- Winter Partial-Peak hours: $243 \times 4 = 972$
- Winter Off-Peak hours: $243 \times 15 = 3645$
- **Weighted Avg Rate Formula**:
  $$\text{Rate}_{\text{avg}} = \frac{610 \cdot \text{Peak}_s + 488 \cdot \text{Part}_s + 1830 \cdot \text{Off}_s + 1215 \cdot \text{Peak}_w + 972 \cdot \text{Part}_w + 3645 \cdot \text{Off}_w}{8760}$$

#### For EV-B (Summer May-Oct, Winter Nov-Apr):
- Summer Weekdays: 128 days. Winter Weekdays: 125 days.
- Summer Weekends/Hols: 56 days. Winter Weekends/Hols: 56 days.
- Summer Peak: $128 \times 7 + 56 \times 4 = 1120$ hours
- Summer Partial-Peak: $128 \times 9 = 1152$ hours
- Summer Off-Peak: $184 \times 24 - 1120 - 1152 = 2144$ hours
- Winter Peak: $125 \times 7 + 56 \times 4 = 1099$ hours
- Winter Partial-Peak: $125 \times 9 = 1125$ hours
- Winter Off-Peak: $181 \times 24 - 1099 - 1125 = 2120$ hours
- **Weighted Avg Rate Formula**:
  $$\text{Rate}_{\text{avg}} = \frac{1120 \cdot \text{Peak}_s + 1152 \cdot \text{Part}_s + 2144 \cdot \text{Off}_s + 1099 \cdot \text{Peak}_w + 1125 \cdot \text{Part}_w + 2120 \cdot \text{Off}_w}{8760}$$

#### For E-1 (Tiered):
- Since E-1 has no hourly variance, the calculation depends on the selected tier.
- If the user selects `--tier 1`, the rate is simply `tier_1`.
- If the user selects `--tier 2`, the rate is simply `tier_2`.
- Default: `tier_2` (representing marginal cost of adding a device to an existing household).

## 9. Validation Rules
- `--watts` must be a positive float greater than zero.
- `--plan` must be one of: `E-1`, `E-TOU-C`, `E-TOU-D`, `EV2`, `E-ELEC`, `EV-B`.
- `rates.json` must be present and parse successfully. If it is not present, the CLI must automatically fallback to a built-in embedded version of the current 2026 rates so it runs "out of the box".

## 10. Exact API Surface

### Command Line Interface

```bash
# Calculate cost for a 150-watt device using E-TOU-C
pge-cost --watts 150 --plan E-TOU-C

# Calculate cost using E-1 Tier 1 (using baseline)
pge-cost --watts 150 --plan E-1 --tier 1

# Update rates database from PG&E's website
pge-cost fetch
```

### Outputs
Prints a beautifully formatted table to stdout:

```
==================================================
PG&E 24/7 DEVICE RUNNING COST ESTIMATOR (2026)
==================================================
Device Wattage : 150.0 W (0.15 kW)
Selected Plan  : E-TOU-C - Residential Time-Of-Use (4-9 PM Everyday)
Effective Rate : $0.39092 per kWh (Weighted 24/7 Average)
Daily Energy   : 3.60 kWh
--------------------------------------------------
ESTIMATED RUNNING COSTS:
Daily Cost     : $1.41
Monthly Cost   : $42.82  (30.42 days average)
Annual Cost    : $513.71
--------------------------------------------------
Note: Marginal calculations based on total bundled rates.
==================================================
```

## 11. Implementation Order
1. **Initialize module**: Run `go mod init pge-cost`.
2. **Implement core types & embedded rates**: Create `pkg/domain/types.go` and embed the initial `rates.json`.
3. **Implement calculator & weights math**: Create `pkg/domain/calculator.go`.
4. **Write unit tests for calculator**: Create `pkg/domain/calculator_test.go` and run tests.
5. **Implement PG&E spreadsheet parser**: Create `pkg/domain/parser.go`.
6. **Write tests for parser**: Create `pkg/domain/parser_test.go`.
7. **Implement main CLI app**: Create `main.go`.
8. **Final test pass and validation**.

## 12. Testing Plan
- Test E-TOU-C, E-TOU-D, EV2, E-ELEC, and E-1 calculations with known rates and compare against hard mathematical verifications.
- Test that missing `rates.json` correctly falls back to embedded rates.
- Test invalid inputs (negative wattage, unknown plans).

## 13. Out-of-Scope / Rejection List
- Multi-tier baseline calculations with dynamically calculated baseline usage budgets (rejected because it depends on the customer's household baseline usage and requires full monthly billing data, which is out of scope for a simple device-specific continuous calculator).

## 14. Definition of Done
- The command `go test ./...` passes with 100% success.
- The command `go vet ./...` is completely clean.
- Executing `pge-cost --watts 100 --plan E-TOU-C` displays accurate, correct, and readable outputs.
- Executing `pge-cost fetch` successfully grabs the spreadsheet, extracts the rates, updates `rates.json`, and subsequent runs use the newly fetched rates.
