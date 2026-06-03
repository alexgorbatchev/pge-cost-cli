package domain

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ParseSpreadsheet parses the PG&E residential spreadsheet and returns updated RatePlan structures.
func ParseSpreadsheet(filePath string, baseDb RatesDatabase) (RatesDatabase, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return RatesDatabase{}, fmt.Errorf("opening excel file: %w", err)
	}
	defer func() {
		_ = f.Close() // best-effort file close
	}()

	updatedDb := RatesDatabase{
		LastUpdated: time.Now().Format("2006-01-02"),
		Plans:       make(map[string]RatePlan),
	}

	// Copy baseline configurations (name, seasons, schedule) from the fallback/existing database
	for id, plan := range baseDb.Plans {
		updatedDb.Plans[id] = RatePlan{
			Name:     plan.Name,
			Seasons:  plan.Seasons,
			Schedule: plan.Schedule,
			Rates:    plan.Rates, // Will overlay parsed rates below
		}
	}

	// 1. Parse Res Inclu TOU sheet for E-1, E-TOU-C, and E-TOU-D
	const mainSheet = "Res Inclu TOU_260301-Present"
	mainRows, err := f.GetRows(mainSheet)
	if err != nil {
		return RatesDatabase{}, fmt.Errorf("getting rows from %s: %w", mainSheet, err)
	}

	if err := parseMainSheetRates(mainRows, updatedDb.Plans); err != nil {
		return RatesDatabase{}, fmt.Errorf("parsing main sheet rates: %w", err)
	}

	// 2. Parse ElecVehicle&Tech sheet for EV-B, EV2, and E-ELEC
	const techSheet = "ElecVehicle&Tech_260301-Present"
	techRows, err := f.GetRows(techSheet)
	if err != nil {
		return RatesDatabase{}, fmt.Errorf("getting rows from %s: %w", techSheet, err)
	}

	if err := parseTechSheetRates(techRows, updatedDb.Plans); err != nil {
		return RatesDatabase{}, fmt.Errorf("parsing tech sheet rates: %w", err)
	}

	return updatedDb, nil
}

func parseMainSheetRates(rows [][]string, plans map[string]RatePlan) error {
	// Find E-1 (Tiered) rates
	// E-1 rates are in row where Col 0 contains "E1, ESR, ES,  ET" or around row 2/3.
	// We'll search for the row containing "E1, ESR, ES,  ET" and search around it for the Tier 1 & Tier 2 prices (usually cols 8 & 9).
	e1RowIdx := -1
	for idx, row := range rows {
		if len(row) > 0 && strings.Contains(row[0], "E1, ESR, ES,  ET") {
			e1RowIdx = idx
			break
		}
	}

	if e1RowIdx != -1 {
		// Try to read rates from the row above or same row
		tier1, tier2 := 0.0, 0.0

		// Look up to 2 rows up or down to find the row with numeric tiered prices in columns 8 & 9
		for i := -1; i <= 1; i++ {
			targetRow := e1RowIdx + i
			if targetRow >= 0 && targetRow < len(rows) {
				r := rows[targetRow]
				if len(r) > 9 && r[8] != "" && r[9] != "" {
					t1Val, e1 := parseRateValue(r[8])
					t2Val, e2 := parseRateValue(r[9])
					if e1 == nil && e2 == nil && t1Val > 0 && t2Val > 0 {
						tier1 = t1Val
						tier2 = t2Val
						break
					}
				}
			}
		}

		if tier1 > 0 && tier2 > 0 {
			p := plans["E-1"]
			p.Tier1 = tier1
			p.Tier2 = tier2
			plans["E-1"] = p
		} else {
			return fmt.Errorf("could not locate numeric E-1 tiered rates around row %d", e1RowIdx)
		}
	} else {
		return fmt.Errorf("could not find row containing E-1 schedule identifier")
	}

	// Find E-TOU-C rates
	cRowIdx := -1
	for idx, row := range rows {
		if len(row) > 0 && strings.Contains(row[0], "Rate Schedule E-TOU-C") {
			cRowIdx = idx
			break
		}
	}

	if cRowIdx != -1 {
		// Structure:
		// row cRowIdx: Summer Peak
		// row cRowIdx+1: Summer Off-Peak
		// row cRowIdx+2: Winter Peak
		// row cRowIdx+3: Winter Off-Peak
		// Rates are in column 9 (index 9)
		p := plans["E-TOU-C"]
		if p.Rates == nil {
			p.Rates = &PlanRates{}
		}

		sumPeak, err := getRowColumnValue(rows, cRowIdx, 9)
		if err != nil {
			return fmt.Errorf("E-TOU-C summer peak rate: %w", err)
		}
		sumOff, err := getRowColumnValue(rows, cRowIdx+1, 9)
		if err != nil {
			return fmt.Errorf("E-TOU-C summer off-peak rate: %w", err)
		}
		winPeak, err := getRowColumnValue(rows, cRowIdx+2, 9)
		if err != nil {
			return fmt.Errorf("E-TOU-C winter peak rate: %w", err)
		}
		winOff, err := getRowColumnValue(rows, cRowIdx+3, 9)
		if err != nil {
			return fmt.Errorf("E-TOU-C winter off-peak rate: %w", err)
		}

		p.Rates.Summer.Peak = sumPeak
		p.Rates.Summer.OffPeak = sumOff
		p.Rates.Winter.Peak = winPeak
		p.Rates.Winter.OffPeak = winOff
		plans["E-TOU-C"] = p
	} else {
		return fmt.Errorf("could not find E-TOU-C schedule row")
	}

	// Find E-TOU-D rates
	dRowIdx := -1
	for idx, row := range rows {
		if len(row) > 0 && strings.Contains(row[0], "Rate Schedule E-TOU-D") {
			dRowIdx = idx
			break
		}
	}

	if dRowIdx != -1 {
		p := plans["E-TOU-D"]
		if p.Rates == nil {
			p.Rates = &PlanRates{}
		}

		sumPeak, err := getRowColumnValue(rows, dRowIdx, 9)
		if err != nil {
			return fmt.Errorf("E-TOU-D summer peak rate: %w", err)
		}
		sumOff, err := getRowColumnValue(rows, dRowIdx+1, 9)
		if err != nil {
			return fmt.Errorf("E-TOU-D summer off-peak rate: %w", err)
		}
		winPeak, err := getRowColumnValue(rows, dRowIdx+2, 9)
		if err != nil {
			return fmt.Errorf("E-TOU-D winter peak rate: %w", err)
		}
		winOff, err := getRowColumnValue(rows, dRowIdx+3, 9)
		if err != nil {
			return fmt.Errorf("E-TOU-D winter off-peak rate: %w", err)
		}

		p.Rates.Summer.Peak = sumPeak
		p.Rates.Summer.OffPeak = sumOff
		p.Rates.Winter.Peak = winPeak
		p.Rates.Winter.OffPeak = winOff
		plans["E-TOU-D"] = p
	} else {
		return fmt.Errorf("could not find E-TOU-D schedule row")
	}

	return nil
}

func parseTechSheetRates(rows [][]string, plans map[string]RatePlan) error {
	// Find EV-B rates
	evbRowIdx := -1
	for idx, row := range rows {
		if len(row) > 0 && strings.Contains(row[0], "Rate Schedule EV, Rate B") {
			evbRowIdx = idx
			break
		}
	}

	if evbRowIdx != -1 {
		p := plans["EV-B"]
		if p.Rates == nil {
			p.Rates = &PlanRates{}
		}

		// Rates in column 8 (index 8)
		sumPeak, err := getRowColumnValue(rows, evbRowIdx, 8)
		if err != nil {
			return fmt.Errorf("EV-B summer peak rate: %w", err)
		}
		sumPart, err := getRowColumnValue(rows, evbRowIdx+1, 8)
		if err != nil {
			return fmt.Errorf("EV-B summer partial peak rate: %w", err)
		}
		sumOff, err := getRowColumnValue(rows, evbRowIdx+2, 8)
		if err != nil {
			return fmt.Errorf("EV-B summer off-peak rate: %w", err)
		}
		winPeak, err := getRowColumnValue(rows, evbRowIdx+3, 8)
		if err != nil {
			return fmt.Errorf("EV-B winter peak rate: %w", err)
		}
		winPart, err := getRowColumnValue(rows, evbRowIdx+4, 8)
		if err != nil {
			return fmt.Errorf("EV-B winter partial peak rate: %w", err)
		}
		winOff, err := getRowColumnValue(rows, evbRowIdx+5, 8)
		if err != nil {
			return fmt.Errorf("EV-B winter off-peak rate: %w", err)
		}

		p.Rates.Summer.Peak = sumPeak
		p.Rates.Summer.PartialPeak = sumPart
		p.Rates.Summer.OffPeak = sumOff
		p.Rates.Winter.Peak = winPeak
		p.Rates.Winter.PartialPeak = winPart
		p.Rates.Winter.OffPeak = winOff
		plans["EV-B"] = p
	} else {
		return fmt.Errorf("could not find EV-B schedule row")
	}

	// Find EV2 rates
	ev2RowIdx := -1
	for idx, row := range rows {
		if len(row) > 0 && strings.Contains(row[0], "Rate Schedule EV2") {
			ev2RowIdx = idx
			break
		}
	}

	if ev2RowIdx != -1 {
		p := plans["EV2"]
		if p.Rates == nil {
			p.Rates = &PlanRates{}
		}

		sumPeak, err := getRowColumnValue(rows, ev2RowIdx, 8)
		if err != nil {
			return fmt.Errorf("EV2 summer peak rate: %w", err)
		}
		sumPart, err := getRowColumnValue(rows, ev2RowIdx+1, 8)
		if err != nil {
			return fmt.Errorf("EV2 summer partial peak rate: %w", err)
		}
		sumOff, err := getRowColumnValue(rows, ev2RowIdx+2, 8)
		if err != nil {
			return fmt.Errorf("EV2 summer off-peak rate: %w", err)
		}
		winPeak, err := getRowColumnValue(rows, ev2RowIdx+3, 8)
		if err != nil {
			return fmt.Errorf("EV2 winter peak rate: %w", err)
		}
		winPart, err := getRowColumnValue(rows, ev2RowIdx+4, 8)
		if err != nil {
			return fmt.Errorf("EV2 winter partial peak rate: %w", err)
		}
		winOff, err := getRowColumnValue(rows, ev2RowIdx+5, 8)
		if err != nil {
			return fmt.Errorf("EV2 winter off-peak rate: %w", err)
		}

		p.Rates.Summer.Peak = sumPeak
		p.Rates.Summer.PartialPeak = sumPart
		p.Rates.Summer.OffPeak = sumOff
		p.Rates.Winter.Peak = winPeak
		p.Rates.Winter.PartialPeak = winPart
		p.Rates.Winter.OffPeak = winOff
		plans["EV2"] = p
	} else {
		return fmt.Errorf("could not find EV2 schedule row")
	}

	// Find E-ELEC rates
	elecRowIdx := -1
	for idx, row := range rows {
		if len(row) > 0 && strings.Contains(row[0], "Rate Schedule E-ELEC") {
			elecRowIdx = idx
			break
		}
	}

	if elecRowIdx != -1 {
		p := plans["E-ELEC"]
		if p.Rates == nil {
			p.Rates = &PlanRates{}
		}

		sumPeak, err := getRowColumnValue(rows, elecRowIdx, 8)
		if err != nil {
			return fmt.Errorf("E-ELEC summer peak rate: %w", err)
		}
		sumPart, err := getRowColumnValue(rows, elecRowIdx+1, 8)
		if err != nil {
			return fmt.Errorf("E-ELEC summer partial peak rate: %w", err)
		}
		sumOff, err := getRowColumnValue(rows, elecRowIdx+2, 8)
		if err != nil {
			return fmt.Errorf("E-ELEC summer off-peak rate: %w", err)
		}
		winPeak, err := getRowColumnValue(rows, elecRowIdx+3, 8)
		if err != nil {
			return fmt.Errorf("E-ELEC winter peak rate: %w", err)
		}
		winPart, err := getRowColumnValue(rows, elecRowIdx+4, 8)
		if err != nil {
			return fmt.Errorf("E-ELEC winter partial peak rate: %w", err)
		}
		winOff, err := getRowColumnValue(rows, elecRowIdx+5, 8)
		if err != nil {
			return fmt.Errorf("E-ELEC winter off-peak rate: %w", err)
		}

		p.Rates.Summer.Peak = sumPeak
		p.Rates.Summer.PartialPeak = sumPart
		p.Rates.Summer.OffPeak = sumOff
		p.Rates.Winter.Peak = winPeak
		p.Rates.Winter.PartialPeak = winPart
		p.Rates.Winter.OffPeak = winOff
		plans["E-ELEC"] = p
	} else {
		return fmt.Errorf("could not find E-ELEC schedule row")
	}

	return nil
}

func getRowColumnValue(rows [][]string, rowIdx int, colIdx int) (float64, error) {
	if rowIdx < 0 || rowIdx >= len(rows) {
		return 0, fmt.Errorf("row index %d out of bounds (total rows %d)", rowIdx, len(rows))
	}
	row := rows[rowIdx]
	if colIdx < 0 || colIdx >= len(row) {
		return 0, fmt.Errorf("column index %d out of bounds for row %d (columns %d)", colIdx, rowIdx, len(row))
	}
	valStr := row[colIdx]
	if valStr == "" {
		return 0, fmt.Errorf("cell value at row %d, col %d is empty", rowIdx, colIdx)
	}
	return parseRateValue(valStr)
}

func parseRateValue(s string) (float64, error) {
	s = strings.TrimSpace(s)
	// Remove any leading currency signs or trailing commentary
	s = strings.TrimPrefix(s, "$")
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse rate %q: %w", s, err)
	}
	return val, nil
}

// DownloadFile retrieves a remote URL and writes it to the local destination file.
func DownloadFile(url string, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http get %s: %w", url, err)
	}
	defer func() {
		_ = resp.Body.Close() // best-effort body close
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad HTTP status: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("create file %s: %w", destPath, err)
	}
	defer func() {
		_ = out.Close() // best-effort cleanup on early return
	}()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("writing body to %s: %w", destPath, err)
	}

	if err := out.Close(); err != nil {
		return fmt.Errorf("closing output file %s: %w", destPath, err)
	}

	return nil
}
