package domain

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseSpreadsheet_Success(t *testing.T) {
	// Create a mock excel file
	f := excelize.NewFile()
	defer f.Close()

	// 1. Create main sheet
	const mainSheet = "Res Inclu TOU_260301-Present"
	_, err := f.NewSheet(mainSheet)
	if err != nil {
		t.Fatalf("failed to create main sheet: %v", err)
	}

	// Set up E-1 (tiered) data
	// Row 2 is 1-indexed, meaning row 3 in Excel. Wait, getRows is 0-indexed slice.
	// Rows:
	// row 0: headers
	// row 1: headers
	// row 2: [Col 0: Residential Schedules, Col 8: 0.32561, Col 9: 0.40702]
	// row 3: [Col 0: E1, ESR, ES,  ET]
	f.SetCellValue(mainSheet, "A3", "Residential Schedules:")
	f.SetCellValue(mainSheet, "I3", "0.32561") // Column I is index 8
	f.SetCellValue(mainSheet, "J3", "0.40702") // Column J is index 9
	f.SetCellValue(mainSheet, "A4", "E1, ESR, ES,  ET")

	// Set up E-TOU-C data (starts at row 8, which is row index 8, i.e. cell row 9)
	f.SetCellValue(mainSheet, "A9", "Residential Time-of-Use\r\nRate Schedule E-TOU-C")
	f.SetCellValue(mainSheet, "H9", "Summer")
	f.SetCellValue(mainSheet, "I9", "Peak")
	f.SetCellValue(mainSheet, "J9", "0.52240") // Col J is index 9

	f.SetCellValue(mainSheet, "I10", "Off-Peak")
	f.SetCellValue(mainSheet, "J10", "0.39940")

	f.SetCellValue(mainSheet, "H11", "Winter")
	f.SetCellValue(mainSheet, "I11", "Peak")
	f.SetCellValue(mainSheet, "J11", "0.39757")

	f.SetCellValue(mainSheet, "I12", "Off-Peak")
	f.SetCellValue(mainSheet, "J12", "0.36757")

	// Set up E-TOU-D data (starts at row 12, row index 12, i.e. cell row 13)
	f.SetCellValue(mainSheet, "A13", "Residential Time-of-Use\r\nRate Schedule E-TOU-D")
	f.SetCellValue(mainSheet, "H13", "Summer")
	f.SetCellValue(mainSheet, "I13", "Peak")
	f.SetCellValue(mainSheet, "J13", "0.47708")

	f.SetCellValue(mainSheet, "I14", "Off-Peak")
	f.SetCellValue(mainSheet, "J14", "0.34212")

	f.SetCellValue(mainSheet, "H15", "Winter")
	f.SetCellValue(mainSheet, "I15", "Peak")
	f.SetCellValue(mainSheet, "J15", "0.38747")

	f.SetCellValue(mainSheet, "I16", "Off-Peak")
	f.SetCellValue(mainSheet, "J16", "0.34886")

	// 2. Create tech sheet
	const techSheet = "ElecVehicle&Tech_260301-Present"
	_, err = f.NewSheet(techSheet)
	if err != nil {
		t.Fatalf("failed to create tech sheet: %v", err)
	}

	// EV-B (starts at row index 1, i.e. row 2)
	f.SetCellValue(techSheet, "A2", "Rate Schedule EV, Rate B")
	f.SetCellValue(techSheet, "G2", "Summer")
	f.SetCellValue(techSheet, "H2", "Peak")
	f.SetCellValue(techSheet, "I2", "0.62131") // Col I is index 8

	f.SetCellValue(techSheet, "H3", "Part-Peak")
	f.SetCellValue(techSheet, "I3", "0.37720")

	f.SetCellValue(techSheet, "H4", "Off-Peak")
	f.SetCellValue(techSheet, "I4", "0.26465")

	f.SetCellValue(techSheet, "G5", "Winter")
	f.SetCellValue(techSheet, "H5", "Peak")
	f.SetCellValue(techSheet, "I5", "0.43878")

	f.SetCellValue(techSheet, "H6", "Part-Peak")
	f.SetCellValue(techSheet, "I6", "0.30677")

	f.SetCellValue(techSheet, "H7", "Off-Peak")
	f.SetCellValue(techSheet, "I7", "0.23504")

	// EV2 (starts at row index 7, i.e. row 8)
	f.SetCellValue(techSheet, "A8", "Rate Schedule EV2")
	f.SetCellValue(techSheet, "G8", "Summer")
	f.SetCellValue(techSheet, "H8", "Peak")
	f.SetCellValue(techSheet, "I8", "0.53809")

	f.SetCellValue(techSheet, "H9", "Part-Peak")
	f.SetCellValue(techSheet, "I9", "0.42760")

	f.SetCellValue(techSheet, "H10", "Off-Peak")
	f.SetCellValue(techSheet, "I10", "0.22558")

	f.SetCellValue(techSheet, "G11", "Winter")
	f.SetCellValue(techSheet, "H11", "Peak")
	f.SetCellValue(techSheet, "I11", "0.41099")

	f.SetCellValue(techSheet, "H12", "Part-Peak")
	f.SetCellValue(techSheet, "I12", "0.39428")

	f.SetCellValue(techSheet, "H13", "Off-Peak")
	f.SetCellValue(techSheet, "I13", "0.22558")

	// E-ELEC (starts at row index 13, i.e. row 14)
	f.SetCellValue(techSheet, "A14", "Rate Schedule E-ELEC")
	f.SetCellValue(techSheet, "G14", "Summer")
	f.SetCellValue(techSheet, "H14", "Peak")
	f.SetCellValue(techSheet, "I14", "0.55214")

	f.SetCellValue(techSheet, "H15", "Part-Peak")
	f.SetCellValue(techSheet, "I15", "0.39026")

	f.SetCellValue(techSheet, "H16", "Off-Peak")
	f.SetCellValue(techSheet, "I16", "0.33358")

	f.SetCellValue(techSheet, "G17", "Winter")
	f.SetCellValue(techSheet, "H17", "Peak")
	f.SetCellValue(techSheet, "I17", "0.32063")

	f.SetCellValue(techSheet, "H18", "Part-Peak")
	f.SetCellValue(techSheet, "I18", "0.29854")

	f.SetCellValue(techSheet, "H19", "Off-Peak")
	f.SetCellValue(techSheet, "I19", "0.28468")

	// Save to temporary file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-pge-rates.xlsx")
	if err := f.SaveAs(filePath); err != nil {
		t.Fatalf("failed to save mock sheet: %v", err)
	}

	// Prepare base DB for copying schema structures
	baseDb := loadTestDatabase(t)

	// Execute Parser
	parsedDb, err := ParseSpreadsheet(filePath, baseDb)
	if err != nil {
		t.Fatalf("ParseSpreadsheet returned unexpected error: %v", err)
	}

	// Assert rates were updated correctly
	// E-1 tiered
	e1 := parsedDb.Plans["E-1"]
	if e1.Tier1 != 0.32561 || e1.Tier2 != 0.40702 {
		t.Errorf("E-1 tiered rates parsed incorrectly: got tier1=%f, tier2=%f; want 0.32561, 0.40702", e1.Tier1, e1.Tier2)
	}

	// E-TOU-C
	etouc := parsedDb.Plans["E-TOU-C"]
	if etouc.Rates == nil {
		t.Fatal("E-TOU-C rates are nil")
	}
	if etouc.Rates.Summer.Peak != 0.52240 || etouc.Rates.Summer.OffPeak != 0.39940 ||
		etouc.Rates.Winter.Peak != 0.39757 || etouc.Rates.Winter.OffPeak != 0.36757 {
		t.Errorf("E-TOU-C rates parsed incorrectly: got %+v", etouc.Rates)
	}

	// E-TOU-D
	etoud := parsedDb.Plans["E-TOU-D"]
	if etoud.Rates == nil {
		t.Fatal("E-TOU-D rates are nil")
	}
	if etoud.Rates.Summer.Peak != 0.47708 || etoud.Rates.Summer.OffPeak != 0.34212 ||
		etoud.Rates.Winter.Peak != 0.38747 || etoud.Rates.Winter.OffPeak != 0.34886 {
		t.Errorf("E-TOU-D rates parsed incorrectly: got %+v", etoud.Rates)
	}

	// EV-B
	evb := parsedDb.Plans["EV-B"]
	if evb.Rates == nil {
		t.Fatal("EV-B rates are nil")
	}
	if evb.Rates.Summer.Peak != 0.62131 || evb.Rates.Summer.PartialPeak != 0.37720 || evb.Rates.Summer.OffPeak != 0.26465 ||
		evb.Rates.Winter.Peak != 0.43878 || evb.Rates.Winter.PartialPeak != 0.30677 || evb.Rates.Winter.OffPeak != 0.23504 {
		t.Errorf("EV-B rates parsed incorrectly: got %+v", evb.Rates)
	}

	// EV2
	ev2 := parsedDb.Plans["EV2"]
	if ev2.Rates == nil {
		t.Fatal("EV2 rates are nil")
	}
	if ev2.Rates.Summer.Peak != 0.53809 || ev2.Rates.Summer.PartialPeak != 0.42760 || ev2.Rates.Summer.OffPeak != 0.22558 ||
		ev2.Rates.Winter.Peak != 0.41099 || ev2.Rates.Winter.PartialPeak != 0.39428 || ev2.Rates.Winter.OffPeak != 0.22558 {
		t.Errorf("EV2 rates parsed incorrectly: got %+v", ev2.Rates)
	}

	// E-ELEC
	eelec := parsedDb.Plans["E-ELEC"]
	if eelec.Rates == nil {
		t.Fatal("E-ELEC rates are nil")
	}
	if eelec.Rates.Summer.Peak != 0.55214 || eelec.Rates.Summer.PartialPeak != 0.39026 || eelec.Rates.Summer.OffPeak != 0.33358 ||
		eelec.Rates.Winter.Peak != 0.32063 || eelec.Rates.Winter.PartialPeak != 0.29854 || eelec.Rates.Winter.OffPeak != 0.28468 {
		t.Errorf("E-ELEC rates parsed incorrectly: got %+v", eelec.Rates)
	}
}
