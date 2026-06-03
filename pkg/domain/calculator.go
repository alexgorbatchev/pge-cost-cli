package domain

import (
	"errors"
	"fmt"
	"time"
)

// CalculationResult holds the calculated cost breakdown for running a device.
type CalculationResult struct {
	PlanID        string  `json:"plan_id"`
	PlanName      string  `json:"plan_name"`
	Watts         float64 `json:"watts"`
	EffectiveRate float64 `json:"effective_rate"`
	DailyEnergy   float64 `json:"daily_energy_kwh"`
	DailyCost     float64 `json:"daily_cost"`
	MonthlyCost   float64 `json:"monthly_cost"`
	AnnualCost    float64 `json:"annual_cost"`
}

// Calculate computes running costs of a continuous device on a given plan for a specific year (default 2026).
func Calculate(planID string, plan RatePlan, watts float64, tier int, year int) (CalculationResult, error) {
	if watts <= 0 {
		return CalculationResult{}, errors.New("wattage must be greater than zero")
	}
	if year <= 0 {
		year = 2026
	}

	rate, err := CalculateWeightedRate(planID, plan, tier, year)
	if err != nil {
		return CalculationResult{}, fmt.Errorf("calculating rate: %w", err)
	}

	dailyEnergy := (watts * 24.0) / 1000.0
	dailyCost := dailyEnergy * rate
	annualCost := dailyCost * 365.0
	monthlyCost := dailyCost * (365.0 / 12.0)

	return CalculationResult{
		PlanID:        planID,
		PlanName:      plan.Name,
		Watts:         watts,
		EffectiveRate: rate,
		DailyEnergy:   dailyEnergy,
		DailyCost:     dailyCost,
		MonthlyCost:   monthlyCost,
		AnnualCost:    annualCost,
	}, nil
}

// CalculateWeightedRate returns the mathematically weighted rate for a plan over a given year.
func CalculateWeightedRate(planID string, plan RatePlan, tier int, year int) (float64, error) {
	if planID == "E-1" {
		if tier <= 0 {
			tier = 2 // Default to Tier 2 for marginal continuous draw.
		}
		switch tier {
		case 1:
			if plan.Tier1 <= 0 {
				return 0, fmt.Errorf("plan E-1 tier 1 rate is not defined or invalid: %f", plan.Tier1)
			}
			return plan.Tier1, nil
		case 2:
			if plan.Tier2 <= 0 {
				return 0, fmt.Errorf("plan E-1 tier 2 rate is not defined or invalid: %f", plan.Tier2)
			}
			return plan.Tier2, nil
		default:
			return 0, fmt.Errorf("invalid tier %d specified for E-1 (must be 1 or 2)", tier)
		}
	}

	if plan.Seasons == nil {
		return 0, errors.New("plan seasons config is missing")
	}
	if plan.Schedule == nil {
		return 0, errors.New("plan schedule config is missing")
	}
	if plan.Rates == nil {
		return 0, errors.New("plan rates config is missing")
	}

	hoursMap, err := CalculateTOUHours(year, plan.Seasons, plan.Schedule)
	if err != nil {
		return 0, fmt.Errorf("calculating TOU hours: %w", err)
	}

	sumPeak := hoursMap["summer"]["peak"]
	sumPart := hoursMap["summer"]["partial_peak"]
	sumOff := hoursMap["summer"]["off_peak"]

	winPeak := hoursMap["winter"]["peak"]
	winPart := hoursMap["winter"]["partial_peak"]
	winOff := hoursMap["winter"]["off_peak"]

	totalHours := sumPeak + sumPart + sumOff + winPeak + winPart + winOff
	if totalHours <= 0 {
		return 0, fmt.Errorf("total hours calculated is zero or negative: %f", totalHours)
	}

	numerator := (sumPeak * plan.Rates.Summer.Peak) +
		(sumPart * plan.Rates.Summer.PartialPeak) +
		(sumOff * plan.Rates.Summer.OffPeak) +
		(winPeak * plan.Rates.Winter.Peak) +
		(winPart * plan.Rates.Winter.PartialPeak) +
		(winOff * plan.Rates.Winter.OffPeak)

	return numerator / totalHours, nil
}

// CalculateTOUHours iterates hour-by-hour through the specified year to count
// how many hours fall into each TOU period for Summer and Winter.
func CalculateTOUHours(year int, seasons *SeasonConfigs, schedule *PlanSchedule) (map[string]map[string]float64, error) {
	hoursMap := map[string]map[string]float64{
		"summer": {
			"peak":         0,
			"partial_peak": 0,
			"off_peak":     0,
		},
		"winter": {
			"peak":         0,
			"partial_peak": 0,
			"off_peak":     0,
		},
	}

	holidays := GetHolidays(year)

	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(year, 12, 31, 23, 0, 0, 0, time.UTC)

	for t := start; !t.After(end); t = t.Add(time.Hour) {
		// 1. Determine Season
		var season string
		if IsInSeason(t, seasons.Summer) {
			season = "summer"
		} else if IsInSeason(t, seasons.Winter) {
			season = "winter"
		} else {
			season = "winter"
		}

		// 2. Select appropriate day schedule (Everyday vs Weekdays vs Weekends/Holidays)
		var ds *DaySchedule
		if schedule.Everyday != nil {
			ds = schedule.Everyday
		} else {
			isWeekend := t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
			isHoliday := holidays[t.Format("2006-01-02")]
			if isWeekend || isHoliday {
				ds = schedule.WeekendsHolidays
			} else {
				ds = schedule.Weekdays
			}
		}

		if ds == nil {
			return nil, fmt.Errorf("no day schedule found for %s", t.Format("2006-01-02"))
		}

		// 3. Find period for current hour
		hour := t.Hour()
		period := ""

		if isHourInRange(hour, ds.Peak) {
			period = "peak"
		} else if isHourInRange(hour, ds.PartialPeak) {
			period = "partial_peak"
		} else if isHourInRange(hour, ds.OffPeak) {
			period = "off_peak"
		} else {
			period = "off_peak"
		}

		hoursMap[season][period]++
	}

	return hoursMap, nil
}

// IsInSeason checks if date t falls in the configured season.
func IsInSeason(t time.Time, s SeasonConfig) bool {
	m := int(t.Month())
	d := t.Day()

	if s.StartMonth <= s.EndMonth {
		if m > s.StartMonth && m < s.EndMonth {
			return true
		}
		if m == s.StartMonth && d >= s.StartDay {
			return true
		}
		if m == s.EndMonth && d <= s.EndDay {
			return true
		}
		return false
	} else {
		if m > s.StartMonth || m < s.EndMonth {
			return true
		}
		if m == s.StartMonth && d >= s.StartDay {
			return true
		}
		if m == s.EndMonth && d <= s.EndDay {
			return true
		}
		return false
	}
}

func isHourInRange(hour int, ranges []TimeRange) bool {
	for _, r := range ranges {
		if hour >= r.Start && hour < r.End {
			return true
		}
	}
	return false
}

// GetHolidays returns a set of YYYY-MM-DD formatted strings for standard PG&E observed holidays.
func GetHolidays(year int) map[string]bool {
	holidays := make(map[string]bool)

	addObserved := func(t time.Time) {
		observed := t
		if t.Weekday() == time.Saturday {
			observed = t.AddDate(0, 0, -1)
		} else if t.Weekday() == time.Sunday {
			observed = t.AddDate(0, 0, 1)
		}
		holidays[observed.Format("2006-01-02")] = true
	}

	// 1. New Year's Day
	addObserved(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC))

	// 2. President's Day (Third Monday in Feb)
	pres := findNthWeekday(year, time.February, time.Monday, 3)
	holidays[pres.Format("2006-01-02")] = true

	// 3. Memorial Day (Last Monday in May)
	mem := findLastWeekday(year, time.May, time.Monday)
	holidays[mem.Format("2006-01-02")] = true

	// 4. Independence Day (July 4)
	addObserved(time.Date(year, 7, 4, 0, 0, 0, 0, time.UTC))

	// 5. Labor Day (First Monday in Sep)
	lab := findNthWeekday(year, time.September, time.Monday, 1)
	holidays[lab.Format("2006-01-02")] = true

	// 6. Veterans Day (Nov 11)
	addObserved(time.Date(year, 11, 11, 0, 0, 0, 0, time.UTC))

	// 7. Thanksgiving Day (Fourth Thursday in Nov)
	thg := findNthWeekday(year, time.November, time.Thursday, 4)
	holidays[thg.Format("2006-01-02")] = true

	// 8. Christmas Day (Dec 25)
	addObserved(time.Date(year, 12, 25, 0, 0, 0, 0, time.UTC))

	return holidays
}

func findNthWeekday(year int, month time.Month, weekday time.Weekday, n int) time.Time {
	count := 0
	for d := 1; d <= 31; d++ {
		t := time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
		if t.Month() != month {
			break
		}
		if t.Weekday() == weekday {
			count++
			if count == n {
				return t
			}
		}
	}
	return time.Time{}
}

func findLastWeekday(year int, month time.Month, weekday time.Weekday) time.Time {
	// Start at the first day of the next month
	t := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC)
	// Subtract days until we get the target weekday
	for {
		t = t.AddDate(0, 0, -1)
		if t.Weekday() == weekday {
			return t
		}
	}
}
