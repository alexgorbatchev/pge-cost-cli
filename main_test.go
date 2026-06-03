package main

import "testing"

func TestNormalizePlan(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"E-1", "E-1"},
		{"e1", "E-1"},
		{"E1", "E-1"},
		{"E-TOU-C", "E-TOU-C"},
		{"etouc", "E-TOU-C"},
		{"tou-c", "E-TOU-C"},
		{"EV2", "EV2"},
		{"ev2a", "EV2"},
		{"EV2-A", "EV2"},
		{"EV-2A", "EV2"},
		{"eelec", "E-ELEC"},
		{"elec", "E-ELEC"},
		{"evb", "EV-B"},
		{"unknown-plan", "UNKNOWN-PLAN"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizePlan(tt.input)
			if got != tt.want {
				t.Errorf("normalizePlan(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}
