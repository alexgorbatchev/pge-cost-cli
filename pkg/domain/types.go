package domain

import (
	_ "embed"
)

//go:embed rates_fallback.json
var DefaultRatesJSON []byte

// TimeRange represents a start and end hour (24-hour format).
type TimeRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// DaySchedule contains lists of time ranges for peak, partial-peak, and off-peak.
type DaySchedule struct {
	Peak        []TimeRange `json:"peak,omitempty"`
	PartialPeak []TimeRange `json:"partial_peak,omitempty"`
	OffPeak     []TimeRange `json:"off_peak,omitempty"`
}

// PlanSchedule outlines when TOU periods apply on different days.
type PlanSchedule struct {
	Everyday         *DaySchedule `json:"everyday,omitempty"`
	Weekdays         *DaySchedule `json:"weekdays,omitempty"`
	WeekendsHolidays *DaySchedule `json:"weekends_holidays,omitempty"`
}

// SeasonConfig defines the date bounds (inclusive) of a season.
type SeasonConfig struct {
	StartMonth int `json:"start_month"`
	StartDay   int `json:"start_day"`
	EndMonth   int `json:"end_month"`
	EndDay     int `json:"end_day"`
}

// SeasonConfigs contains summer and winter season specifications.
type SeasonConfigs struct {
	Summer SeasonConfig `json:"summer"`
	Winter SeasonConfig `json:"winter"`
}

// SeasonRates represents the actual electricity rates ($/kWh) for a season.
type SeasonRates struct {
	Peak        float64 `json:"peak"`
	PartialPeak float64 `json:"partial_peak,omitempty"`
	OffPeak     float64 `json:"off_peak"`
}

// PlanRates holds the seasonal prices for TOU plans.
type PlanRates struct {
	Summer SeasonRates `json:"summer"`
	Winter SeasonRates `json:"winter"`
}

// RatePlan is the complete representation of a PG&E residential plan.
type RatePlan struct {
	Name     string         `json:"name"`
	Tier1    float64        `json:"tier_1,omitempty"`
	Tier2    float64        `json:"tier_2,omitempty"`
	Seasons  *SeasonConfigs `json:"seasons,omitempty"`
	Schedule *PlanSchedule  `json:"schedule,omitempty"`
	Rates    *PlanRates     `json:"rates,omitempty"`
}

// RatesDatabase contains the parsed list of PG&E plans.
type RatesDatabase struct {
	LastUpdated string              `json:"last_updated"`
	Plans       map[string]RatePlan `json:"plans"`
}
