package domain

import (
	"encoding/json"
	"math"
	"testing"
)

// Helper to load the embedded fallback database.
func loadTestDatabase(t *testing.T) RatesDatabase {
	t.Helper()
	var db RatesDatabase
	if err := json.Unmarshal(DefaultRatesJSON, &db); err != nil {
		t.Fatalf("failed to unmarshal fallback JSON: %v", err)
	}
	return db
}

func TestCalculateWeightedRate_2026(t *testing.T) {
	db := loadTestDatabase(t)

	tests := []struct {
		planID      string
		tier        int
		expectedMin float64
		expectedMax float64
	}{
		{"E-1", 1, 0.32561, 0.32561},
		{"E-1", 2, 0.40702, 0.40702},
		{"E-1", 0, 0.40702, 0.40702}, // default tier 2
		{"E-TOU-C", 0, 0.39093, 0.39094}, // mathematically precise: 0.390935
		{"E-TOU-D", 0, 0.35278, 0.35280}, // mathematically precise: 0.352790
		{"EV2", 0, 0.30302, 0.30304},     // mathematically precise: 0.30303
		{"E-ELEC", 0, 0.32591, 0.32594},  // mathematically precise: 0.32593
		{"EV-B", 0, 0.34513, 0.34515},    // mathematically precise: 0.345141
	}

	for _, tt := range tests {
		t.Run(tt.planID, func(t *testing.T) {
			plan, ok := db.Plans[tt.planID]
			if !ok {
				t.Fatalf("plan %s not found in database", tt.planID)
			}

			rate, err := CalculateWeightedRate(tt.planID, plan, tt.tier, 2026)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tt.planID, err)
			}

			if plan.Seasons != nil && plan.Schedule != nil {
				hours, _ := CalculateTOUHours(2026, plan.Seasons, plan.Schedule)
				t.Logf("Plan %s hours in 2026: Summer(Peak: %f, Part: %f, Off: %f), Winter(Peak: %f, Part: %f, Off: %f)",
					tt.planID, hours["summer"]["peak"], hours["summer"]["partial_peak"], hours["summer"]["off_peak"],
					hours["winter"]["peak"], hours["winter"]["partial_peak"], hours["winter"]["off_peak"])
				if plan.Rates != nil {
					t.Logf("Plan %s rates: Summer(Peak: %f, Part: %f, Off: %f), Winter(Peak: %f, Part: %f, Off: %f)",
						tt.planID, plan.Rates.Summer.Peak, plan.Rates.Summer.PartialPeak, plan.Rates.Summer.OffPeak,
						plan.Rates.Winter.Peak, plan.Rates.Winter.PartialPeak, plan.Rates.Winter.OffPeak)
				}
			}

			if rate < tt.expectedMin || rate > tt.expectedMax {
				t.Errorf("CalculateWeightedRate(%s) = %f; want value in range [%f, %f]",
					tt.planID, rate, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestCalculate(t *testing.T) {
	db := loadTestDatabase(t)

	t.Run("Valid Calculation E-TOU-C", func(t *testing.T) {
		plan := db.Plans["E-TOU-C"]
		watts := 150.0

		result, err := Calculate("E-TOU-C", plan, watts, 0, 2026)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify structural correctness of output
		if result.PlanID != "E-TOU-C" {
			t.Errorf("expected plan ID E-TOU-C, got %s", result.PlanID)
		}
		if result.Watts != watts {
			t.Errorf("expected watts %f, got %f", watts, result.Watts)
		}

		// Daily energy check: (150 * 24) / 1000 = 3.6 kWh
		expectedDailyEnergy := 3.6
		if math.Abs(result.DailyEnergy-expectedDailyEnergy) > 1e-9 {
			t.Errorf("expected daily energy %f, got %f", expectedDailyEnergy, result.DailyEnergy)
		}

		// Effective rate check: should be around 0.39092
		expectedRate := 0.39092
		if math.Abs(result.EffectiveRate-expectedRate) > 0.0001 {
			t.Errorf("expected effective rate close to %f, got %f", expectedRate, result.EffectiveRate)
		}

		// Cost checks
		expectedDailyCost := expectedDailyEnergy * result.EffectiveRate
		if math.Abs(result.DailyCost-expectedDailyCost) > 1e-9 {
			t.Errorf("expected daily cost %f, got %f", expectedDailyCost, result.DailyCost)
		}

		expectedAnnualCost := expectedDailyCost * 365.0
		if math.Abs(result.AnnualCost-expectedAnnualCost) > 1e-9 {
			t.Errorf("expected annual cost %f, got %f", expectedAnnualCost, result.AnnualCost)
		}
	})

	t.Run("Invalid Wattage", func(t *testing.T) {
		plan := db.Plans["E-TOU-C"]
		_, err := Calculate("E-TOU-C", plan, -10.0, 0, 2026)
		if err == nil {
			t.Error("expected error for negative wattage, got nil")
		}
	})

	t.Run("Invalid Plan ID", func(t *testing.T) {
		_, err := Calculate("UNKNOWN", RatePlan{}, 100.0, 0, 2026)
		if err == nil {
			t.Error("expected error for unknown plan ID, got nil")
		}
	})
}
