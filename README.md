# `pge-cost` — PG&E 24/7 Device Running Cost Estimator

`pge-cost` is a high-precision Go CLI utility that calculates the exact running costs of continuously operating electrical devices (e.g., home servers, network equipment, refrigerators, or security cameras) under various Pacific Gas & Electric (PG&E) residential rate plans.

Rather than relying on naive flat-rate estimates, `pge-cost` simulates the operation of a device hour-by-hour over an entire calendar year, factoring in seasonal shifts, weekdays vs. weekends, and legally observed holidays to compute mathematically precise time-weighted average rates.

---

## Features

- **Dynamic Scheduling Engine**: Simulates all 8,760 hours of a calendar year to determine exact time-of-use (TOU) pricing.
- **Observed Holiday Awareness**: Handles PG&E's 8 standard residential legal holidays (with weekend-observance shifting).
- **Supports All Major PG&E Residential Plans**:
  - **`E-1`**: Tiered General Service (with baseline vs. marginal cost defaults).
  - **`E-TOU-C`**: Peak Pricing 4:00 PM – 9:00 PM Everyday.
  - **`E-TOU-D`**: Peak Pricing 5:00 PM – 8:00 PM Weekdays (no holidays).
  - **`EV2`**: Everyday Electric Vehicle & Battery Storage.
  - **`E-ELEC`**: Everyday Residential Electric Technology Home.
  - **`EV-B`**: Residential EV with a dedicated meter (separate peak windows).
- **Embedded Rates Fallback**: Works completely "out of the box" using baked-in rates, requiring no initial files or internet access.
- **Automated DB Sync**: Includes a command to download, parse, and overlay the latest CPUC-approved rates directly from PG&E's official published tariff spreadsheets.

---

## Usage

`pge-cost` offers cost calculation and rates database updates.

### 1. Cost Calculations
Estimate continuous 24/7 cost for a device by specifying its constant power draw in Watts and the target rate plan.

```bash
# Estimate cost for a 150-watt device on the E-TOU-C plan (4-9 PM Everyday Peak)
./pge-cost --watts 150 --plan E-TOU-C
```

#### Expected Output:
```text
╭──────────────────────────────────────────────────────────────────────────╮
│ PG&E 24/7 DEVICE SPECIFICATIONS (2026)                                   │
├────────────────┬─────────────────────────────────────────────────────────┤
│ PARAMETER      │ VALUE                                                   │
├────────────────┼─────────────────────────────────────────────────────────┤
│ Device Wattage │ 150.0 W (0.150 kW)                                      │
│ Selected Plan  │ E-TOU-C - Residential Time-Of-Use (4-9 PM Everyday) [1] │
│ Effective Rate │ $0.39094 per kWh (Weighted 24/7 Average) [2]            │
╰────────────────┴─────────────────────────────────────────────────────────╯

╭─────────────────────────────────────────────────────────────────╮
│ ESTIMATED RUNNING COSTS                                         │
├──────────────────────────┬─────────────────┬────────────────────┤
│ PERIOD                   │ ENERGY CONSUMED │ ESTIMATED COST [3] │
├──────────────────────────┼─────────────────┼────────────────────┤
│ Daily                    │ 3.60 kWh        │ $1.41              │
│ Monthly (30.42 days avg) │ 109.50 kWh      │ $42.81             │
│ Annual                   │ 1314.00 kWh     │ $513.69            │
╰──────────────────────────┴─────────────────┴────────────────────╯

[1] Selected rates loaded from PG&E database "rates.json" (last updated 2026-06-03).
[2] Calculated using hour-by-hour calendar scheduling weights for year 2026.
[3] Marginal calculations based on total bundled rates.
```

#### E-1 Tiered Calculations
For the tiered `E-1` plan, costs depend on which tier your device's energy draw falls into. By default, calculations use **Tier 2** prices. This represents the true *marginal* cost of adding an additional appliance to an existing household that already consumes past its baseline allowance.

```bash
# Calculate using Tier 1 (within baseline allowance)
./pge-cost --watts 150 --plan E-1 --tier 1

# Calculate using Tier 2 (above baseline allowance - default)
./pge-cost --watts 150 --plan E-1 --tier 2
```

### 2. Synchronization of Rates
You can automatically synchronize local database records with live PG&E rate structures. The `fetch` command downloads PG&E's current residential tariff spreadsheet directly from `pge.com`, parses the latest prices for all 6 schedules, and updates `rates.json`.

```bash
# Download and synchronize the local database
./pge-cost fetch
```

---

## Installation & Build

### 1. Download Precompiled Binaries
You can download the precompiled static binaries for macOS and Linux on both Intel/AMD (`amd64`) and Apple Silicon (`arm64`) architectures directly from the GitHub project's [Releases](https://github.com/alexgorbatchev/pge-cost-cli/releases) page. No Go toolchain is required.

Once downloaded, extract the archive and run the executable:
```bash
# Example for Linux AMD64
tar -xzf pge-cost_v1.0.0_linux_amd64.tar.gz
./pge-cost --watts 100 --plan E-TOU-C
```

### 2. Build from Source

#### Prerequisites
- [Go](https://go.dev/) 1.26 or later
- [Just](https://github.com/casey/just) command runner (optional, but convenient)

#### Build Commands
1. Clone the repository:
   ```bash
   git clone https://github.com/alexgorbatchev/pge-cost-cli.git
   cd pge-cost-cli
   ```
2. Compile the executable binary:
   ```bash
   go build -o pge-cost main.go
   ```
   Or using `just`:
   ```bash
   just build
   ```

---

## What is the "Weighted 24/7 Average"?

For any electrical device that runs continuously 24 hours a day, 7 days a week, the **Weighted 24/7 Average** represents the true mathematical rate paid per kilowatt-hour (kWh) of electricity consumed over an entire calendar year.

Because PG&E rates fluctuate dynamically based on the hour of the day (Peak vs. Off-Peak), the day of the week (Weekdays vs. Weekends), and the season (Summer vs. Winter), lookups on a static rate sheet can be misleading. `pge-cost` blends these shifting rates together.

### The Mathematical Proof
If a device draws a constant wattage of $W$ kilowatts (kW) continuously, it consumes exactly $W$ kWh of energy every single hour.

1. **Total Energy Consumed in a Year**:
   $$\text{Total Energy} = W \times 8,760 \text{ hours (standard calendar year)}$$

2. **Total Cost in a Year**:
   Let $\text{Rate}_{\text{hour}}$ be the price at any specific hour of the year. The total bill is the sum of the hourly costs:
   $$\text{Total Cost} = \sum_{\text{hour}=1}^{8,760} (W \times \text{Rate}_{\text{hour}})$$

3. **Factoring out Wattage**:
   Since the device wattage ($W$) is constant, it factors out of the mathematical summation:
   $$\text{Total Cost} = W \times \sum_{\text{hour}=1}^{8,760} \text{Rate}_{\text{hour}}$$

4. **Calculating the Average Rate per kWh**:
   Dividing the total cost by total energy consumed yields:
   $$\text{Average Rate} = \frac{\text{Total Cost}}{\text{Total Energy}} = \frac{W \times \sum_{\text{hour}=1}^{8,760} \text{Rate}_{\text{hour}}}{W \times 8,760}$$

5. **The Wattage Cancels Out**:
   $$\text{Average Rate} = \frac{\sum_{\text{hour}=1}^{8,760} \text{Rate}_{\text{hour}}}{8,760}$$

This means **the average rate paid per kWh is completely independent of the device's size (wattage)**. It is purely the time-weighted average of all 8,760 hourly rates in the calendar year.

---

### Concrete Calendar Distribution (2026 Non-Leap Year)

#### 1. Summer vs. Winter Splits
- **Plans with June–September Summer (E-TOU-C, E-TOU-D, EV2, E-ELEC)**:
  - **Summer (122 days)**: 86 Weekdays, 34 Weekends, 2 Holidays (Independence Day observed July 3, Labor Day Sept 7).
  - **Winter (243 days)**: 167 Weekdays, 70 Weekends, 6 Holidays.
- **Plans with May–October Summer (EV-B)**:
  - **Summer (184 days)**: 128 Weekdays, 53 Weekends, 3 Holidays.
  - **Winter (181 days)**: 125 Weekdays, 51 Weekends, 5 Holidays.

#### 2. E-TOU-C Hour Weights & Rate Blend
Under E-TOU-C, Peak rates apply from 4:00 PM to 9:00 PM (5 hours/day) everyday, and Off-Peak rates apply for the remaining 19 hours.

| Season | Period | Daily Hours | Annual Days | Annual Hours | PG&E Rate ($/kWh) |
| :--- | :--- | :---: | :---: | :---: | :---: |
| **Summer** | Peak | 5 | 122 | **610** | **$0.52240** |
| **Summer** | Off-Peak | 19 | 122 | **2,318** | **$0.39940** |
| **Winter** | Peak | 5 | 243 | **1,215** | **$0.39757** |
| **Winter** | Off-Peak | 19 | 243 | **4,617** | **$0.36757** |
| **Total Year**| | **24** | **365** | **8,760** | |

#### Weighted Calculation:
$$\text{Average Rate} = \frac{(610 \times 0.52240) + (2,318 \times 0.39940) + (1,215 \times 0.39757) + (4,617 \times 0.36757)}{8,760}$$
$$\text{Average Rate} = \frac{3,424.59144}{8,760} \approx \mathbf{\$0.39094\text{ per kWh}}$$

---

## Comparison of 24/7 Device Rates (CPUC Effective March 1, 2026)

Comparing plans shows that switching a continuously running device from tiered `E-1` or standard `E-TOU-C` to electric vehicle/storage schedules (`EV2` or `E-ELEC`) yields significant savings of **over 20% to 25%** for that appliance:

| Plan ID | Description | 24/7 Weighted Average | Real Cost/Yr (100W draw) |
| :--- | :--- | :---: | :---: |
| **`EV2`** | Electric Vehicle & Storage Everyday | **$0.30303** | $265.45 |
| **`E-ELEC`** | Residential Electric Home | **$0.32593** | $285.52 |
| **`EV-B`** | Residential EV with Dedicated Meter | **$0.34514** | $302.34 |
| **`E-TOU-D`**| 5:00 PM – 8:00 PM Weekdays Peak | **$0.35279** | $309.04 |
| **`E-TOU-C`**| 4:00 PM – 9:00 PM Everyday Peak | **$0.39094** | $342.46 |
| **`E-1 (Tier 2)`**| Tiered (Marginal Draw - Default) | **$0.40702** | $356.55 |
| **`E-1 (Tier 1)`**| Tiered (Within Baseline Allowance) | **$0.32561** | $285.23 |

---

## Development

The project includes a full table-driven test suite validating calendar transitions, holiday definitions, cost functions, and the spreadsheet parser.

```bash
# Run tests and static analysis using just
just test
just vet

# Run tests and analysis manually
go test -v ./...
go vet ./...
```

### PG&E Holiday Policy
The scheduler recognizes exactly 8 standard residential legal holidays:
1. **New Year's Day** (January 1)
2. **Presidents' Day** (Third Monday in February)
3. **Memorial Day** (Last Monday in May)
4. **Independence Day** (July 4)
5. **Labor Day** (First Monday in September)
6. **Veterans Day** (November 11)
7. **Thanksgiving Day** (Fourth Thursday in November)
8. **Christmas Day** (December 25)

When any holiday falls on a Saturday, standard PG&E residential policy shifts the legal observance to the preceding Friday. When any holiday falls on a Sunday, it shifts to the following Monday. These days are fully scheduled using weekend rate plans.
