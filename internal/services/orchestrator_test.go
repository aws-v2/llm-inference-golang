package service

import (
	"testing"
	"time"
)

func TestParseScheduleInterval(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"manual", 0, false},
		{"0", 0, false},
		{"", 0, false},
		{"none", 0, false},
		{"10", 10 * time.Minute, false},
		{"15", 15 * time.Minute, false},
		{"60", 60 * time.Minute, false},
		{"10m", 10 * time.Minute, false},
		{"15m", 15 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"2h", 2 * time.Hour, false},
		{"hourly", 1 * time.Hour, false},
		{"daily", 24 * time.Hour, false},
		{"weekly", 7 * 24 * time.Hour, false},
		{"15 minutes", 15 * time.Minute, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseScheduleInterval(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseScheduleInterval(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseScheduleInterval(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
