package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"pge-cost/pkg/domain"
	"strings"
)

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

	fmt.Println("==================================================")
	fmt.Printf(fmt.Sprintf("PG%%-%ds DEVICE RUNNING COST ESTIMATOR (%%d)\n", 22), "E 24/7", *yearFlag)
	fmt.Println("==================================================")
	fmt.Fprintf(os.Stdout, "Device Wattage : %.1f W (%.3f kW)\n", result.Watts, result.Watts/1000.0)
	fmt.Fprintf(os.Stdout, "Selected Plan  : %s - %s\n", result.PlanID, planDesc)
	fmt.Fprintf(os.Stdout, "Effective Rate : $%.5f per kWh (Weighted 24/7 Average)\n", result.EffectiveRate)
	fmt.Fprintf(os.Stdout, "Daily Energy   : %.2f kWh\n", result.DailyEnergy)
	fmt.Println("--------------------------------------------------")
	fmt.Println("ESTIMATED RUNNING COSTS:")
	fmt.Fprintf(os.Stdout, "Daily Cost     : $%.2f\n", result.DailyCost)
	fmt.Fprintf(os.Stdout, "Monthly Cost   : $%.2f  (30.42 days average)\n", result.MonthlyCost)
	fmt.Fprintf(os.Stdout, "Annual Cost    : $%.2f\n", result.AnnualCost)
	fmt.Println("--------------------------------------------------")
	if !loadedFromDisk {
		fmt.Println("Note: Using embedded 2026 rates fallback database.")
	} else {
		fmt.Fprintf(os.Stdout, "Note: Using database %q (last updated %s).\n", *dbFlag, db.LastUpdated)
	}
	fmt.Println("Marginal calculations based on total bundled rates.")
	fmt.Println("==================================================")
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

	fmt.Println("\n==================================================")
	fmt.Println("PG&E RATES DATABASE SYNCHRONIZATION SUCCESS")
	fmt.Println("==================================================")
	fmt.Fprintf(os.Stdout, "Source URL   : %s\n", *urlFlag)
	fmt.Fprintf(os.Stdout, "Destination  : %s\n", *dbFlag)
	fmt.Fprintf(os.Stdout, "Last Updated : %s\n", updatedDb.LastUpdated)
	fmt.Println("--------------------------------------------------")
	fmt.Println("UPDATED RATES SUMMARY:")
	fmt.Fprintf(os.Stdout, "- E-1      (Tiered)     : Tier 1 = $%.5f, Tier 2 = $%.5f\n", e1.Tier1, e1.Tier2)
	fmt.Fprintf(os.Stdout, "- E-TOU-C  (TOU Everyday) : Summer Peak = $%.5f, Winter Peak = $%.5f\n", etouc.Rates.Summer.Peak, etouc.Rates.Winter.Peak)
	fmt.Fprintf(os.Stdout, "- E-TOU-D  (TOU Weekdays) : Summer Peak = $%.5f, Winter Peak = $%.5f\n", etoud.Rates.Summer.Peak, etoud.Rates.Winter.Peak)
	fmt.Fprintf(os.Stdout, "- EV2      (Everyday EV): Summer Peak = $%.5f, Winter Peak = $%.5f\n", ev2.Rates.Summer.Peak, ev2.Rates.Winter.Peak)
	fmt.Fprintf(os.Stdout, "- E-ELEC   (Elect Home) : Summer Peak = $%.5f, Winter Peak = $%.5f\n", eelec.Rates.Summer.Peak, eelec.Rates.Winter.Peak)
	fmt.Fprintf(os.Stdout, "- EV-B     (Metered EV) : Summer Peak = $%.5f, Winter Peak = $%.5f\n", evb.Rates.Summer.Peak, evb.Rates.Winter.Peak)
	fmt.Println("==================================================")
}
