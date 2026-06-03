package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"pge-cost/pkg/domain"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
)

var version = "dev"

func main() {
	// If the user requests "fetch", route to the fetch subcommand
	if len(os.Args) > 1 && os.Args[1] == "fetch" {
		handleFetch()
		return
	}

	// Main Calculation CLI Flags
	wattsFlag := flag.Float64("watts", 0, "Device wattage in Watts (must be positive, e.g. 150)")
	planFlag := flag.String("plan", "", "PG&E rate plan to use (E-1, E-TOU-C, E-TOU-D, EV2, E-ELEC, EV-B)")
	tierFlag := flag.Int("tier", 2, "Tier to use for E-1 tiered plan (1 or 2, default 2)")
	dbFlag := flag.String("db", "rates.json", "Path to rates JSON database file")
	yearFlag := flag.Int("year", 2026, "The calendar year for scheduling weights (default 2026)")
	versionFlag := flag.Bool("version", false, "Print the version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "PG&E Continuous Running Cost Estimator CLI\n\n")
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --watts <watts> --plan <plan> [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  fetch             Download, parse, and update the local rates database\n")
		fmt.Fprintf(os.Stderr, "                    Usage: %s fetch [--db rates.json] [--url <url>]\n\n", os.Args[0])
	}

	flag.Parse()

	if *versionFlag {
		fmt.Printf("pge-cost version %s\n", version)
		return
	}

	// Validation
	if *wattsFlag <= 0 {
		fmt.Fprintln(os.Stderr, "Error: --watts is required and must be a positive number greater than zero.")
		flag.Usage()
		os.Exit(1)
	}

	if *planFlag == "" {
		fmt.Fprintln(os.Stderr, "Error: --plan is required.")
		flag.Usage()
		os.Exit(1)
	}

	planID := strings.ToUpper(*planFlag)
	validPlans := map[string]bool{
		"E-1":     true,
		"E-TOU-C": true,
		"E-TOU-D": true,
		"EV2":     true,
		"E-ELEC":  true,
		"EV-B":    true,
	}

	if !validPlans[planID] {
		fmt.Fprintf(os.Stderr, "Error: unknown plan %q. Valid options are: E-1, E-TOU-C, E-TOU-D, EV2, E-ELEC, EV-B.\n", *planFlag)
		flag.Usage()
		os.Exit(1)
	}

	if planID == "E-1" && *tierFlag != 1 && *tierFlag != 2 {
		fmt.Fprintf(os.Stderr, "Error: invalid E-1 --tier %d. Must be 1 or 2.\n", *tierFlag)
		flag.Usage()
		os.Exit(1)
	}

	// Load Rates Database
	var db domain.RatesDatabase
	loadedFromDisk := false

	dbBytes, err := os.ReadFile(*dbFlag)
	if err == nil {
		if err := json.Unmarshal(dbBytes, &db); err == nil {
			loadedFromDisk = true
		} else {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse rates database %q: %v. Falling back to embedded rates.\n", *dbFlag, err)
		}
	}

	if !loadedFromDisk {
		if err := json.Unmarshal(domain.DefaultRatesJSON, &db); err != nil {
			fmt.Fprintf(os.Stderr, "Critical Error: failed to load embedded rates fallback: %v\n", err)
			os.Exit(1)
		}
	}

	plan, exists := db.Plans[planID]
	if !exists {
		fmt.Fprintf(os.Stderr, "Error: plan %q not configured in database.\n", planID)
		os.Exit(1)
	}

	// Calculate Costs
	result, err := domain.Calculate(planID, plan, *wattsFlag, *tierFlag, *yearFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: calculation failed: %v\n", err)
		os.Exit(1)
	}

	// Format Output Table
	planDesc := plan.Name
	if planID == "E-1" {
		planDesc = fmt.Sprintf("%s (Tier %d)", planDesc, *tierFlag)
	}

	tw1 := table.NewWriter()
	tw1.SetOutputMirror(os.Stdout)
	tw1.SetStyle(table.StyleRounded)
	tw1.SetTitle("PG&E 24/7 DEVICE SPECIFICATIONS (%d)", *yearFlag)
	tw1.AppendHeader(table.Row{"Parameter", "Value"})
	tw1.AppendRows([]table.Row{
		{"Device Wattage", fmt.Sprintf("%.1f W (%.3f kW)", result.Watts, result.Watts/1000.0)},
		{"Selected Plan", fmt.Sprintf("%s - %s [1]", result.PlanID, planDesc)},
		{"Effective Rate", fmt.Sprintf("$%.5f per kWh (Weighted 24/7 Average) [2]", result.EffectiveRate)},
	})
	tw1.Render()

	fmt.Println()

	tw2 := table.NewWriter()
	tw2.SetOutputMirror(os.Stdout)
	tw2.SetStyle(table.StyleRounded)
	tw2.SetTitle("ESTIMATED RUNNING COSTS")
	tw2.AppendHeader(table.Row{"Period", "Energy Consumed", "Estimated Cost [3]"})
	tw2.AppendRows([]table.Row{
		{"Daily", fmt.Sprintf("%.2f kWh", result.DailyEnergy), fmt.Sprintf("$%.2f", result.DailyCost)},
		{"Monthly (30.42 days avg)", fmt.Sprintf("%.2f kWh", result.DailyEnergy*30.4167), fmt.Sprintf("$%.2f", result.MonthlyCost)},
		{"Annual", fmt.Sprintf("%.2f kWh", result.DailyEnergy*365.0), fmt.Sprintf("$%.2f", result.AnnualCost)},
	})
	tw2.Render()

	fmt.Println()

	var dbSource string
	if !loadedFromDisk {
		dbSource = "embedded 2026 rates fallback database"
	} else {
		dbSource = fmt.Sprintf("database %q (last updated %s)", *dbFlag, db.LastUpdated)
	}

	fmt.Printf("[1] Selected rates loaded from PG&E %s.\n", dbSource)
	fmt.Printf("[2] Calculated using hour-by-hour calendar scheduling weights for year %d.\n", *yearFlag)
	fmt.Println("[3] Marginal calculations based on total bundled rates.")
}

func handleFetch() {
	// Create a new FlagSet for fetch
	fetchCmd := flag.NewFlagSet("fetch", flag.ExitOnError)
	dbFlag := fetchCmd.String("db", "rates.json", "Destination database JSON file")
	urlFlag := fetchCmd.String("url", "https://www.pge.com/assets/rates/tariffs/res-inclu-tou-current.xlsx", "PG&E published residential rates XLSX URL")

	fetchCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, "Fetch and update rates from PG&E spreadsheet.\n\n")
		fmt.Fprintf(os.Stderr, "Usage of %s fetch:\n", os.Args[0])
		fetchCmd.PrintDefaults()
		fmt.Fprintln(os.Stderr)
	}

	// Parse fetch flags (omit "fetch" word)
	if len(os.Args) > 2 {
		fetchCmd.Parse(os.Args[2:])
	} else {
		fetchCmd.Parse([]string{})
	}

	// Step 1: Create local .tmp directory if not exists
	tmpDir := ".tmp"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to create temporary directory: %v\n", err)
		os.Exit(1)
	}

	tmpFile := filepath.Join(tmpDir, "downloaded-rates.xlsx")

	fmt.Printf("Downloading PG&E spreadsheet from:\n  %s\n", *urlFlag)
	fmt.Println("Please wait, downloading...")

	// Step 2: Download the spreadsheet
	if err := domain.DownloadFile(*urlFlag, tmpFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error: download failed: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile) // Best effort cleanup

	fmt.Println("Parsing rates from downloaded spreadsheet...")

	// Step 3: Load baseline structure
	var baseDb domain.RatesDatabase
	if err := json.Unmarshal(domain.DefaultRatesJSON, &baseDb); err != nil {
		fmt.Fprintf(os.Stderr, "Critical Error: failed to load base configurations: %v\n", err)
		os.Exit(1)
	}

	// Step 4: Parse the downloaded spreadsheet
	updatedDb, err := domain.ParseSpreadsheet(tmpFile, baseDb)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: parsing failed: %v\n", err)
		os.Exit(1)
	}

	// Step 5: Save database to disk
	dbBytes, err := json.MarshalIndent(updatedDb, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to marshal updated database: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*dbFlag, dbBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write updated database to %q: %v\n", *dbFlag, err)
		os.Exit(1)
	}

	// Step 6: Print beautiful success summary
	e1 := updatedDb.Plans["E-1"]
	etouc := updatedDb.Plans["E-TOU-C"]
	etoud := updatedDb.Plans["E-TOU-D"]
	ev2 := updatedDb.Plans["EV2"]
	eelec := updatedDb.Plans["E-ELEC"]
	evb := updatedDb.Plans["EV-B"]

	fmt.Println()
	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.SetStyle(table.StyleRounded)
	tw.SetTitle("PG&E RATES DATABASE SYNCHRONIZATION SUCCESS")
	tw.AppendHeader(table.Row{"Sync Detail / Rate Plan", "Value"})
	tw.AppendRows([]table.Row{
		{"Source URL", *urlFlag},
		{"Destination", *dbFlag},
		{"Last Updated", updatedDb.LastUpdated},
	})
	tw.AppendSeparator()
	tw.AppendRows([]table.Row{
		{"Plan E-1 (Tiered)", fmt.Sprintf("Tier 1 = $%.5f, Tier 2 = $%.5f", e1.Tier1, e1.Tier2)},
		{"Plan E-TOU-C (Everyday)", fmt.Sprintf("Summer Peak = $%.5f, Winter Peak = $%.5f", etouc.Rates.Summer.Peak, etouc.Rates.Winter.Peak)},
		{"Plan E-TOU-D (Weekdays)", fmt.Sprintf("Summer Peak = $%.5f, Winter Peak = $%.5f", etoud.Rates.Summer.Peak, etoud.Rates.Winter.Peak)},
		{"Plan EV2 (Everyday EV)", fmt.Sprintf("Summer Peak = $%.5f, Winter Peak = $%.5f", ev2.Rates.Summer.Peak, ev2.Rates.Winter.Peak)},
		{"Plan E-ELEC (Elect Home)", fmt.Sprintf("Summer Peak = $%.5f, Winter Peak = $%.5f", eelec.Rates.Summer.Peak, eelec.Rates.Winter.Peak)},
		{"Plan EV-B (Metered EV)", fmt.Sprintf("Summer Peak = $%.5f, Winter Peak = $%.5f", evb.Rates.Summer.Peak, evb.Rates.Winter.Peak)},
	})
	tw.Render()
	fmt.Println()
}
