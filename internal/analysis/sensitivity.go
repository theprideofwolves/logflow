package analysis

import "time"

// SensitivityCell describes support from retained trigger evidence, not a rerun
// of the complete input. Higher thresholds may need events we did not retain.
type SensitivityCell struct {
	Threshold   int
	Window      time.Duration
	WindowLabel string
	Peak        int
	Supported   bool
}

// Sensitivity compares a bounded 3x3 neighborhood of the configured rule.
// It never changes the alert, its identity, or its evidence.
func Sensitivity(evidence []Evidence, o Options) []SensitivityCell {
	thresholds := []int{o.Threshold - 1, o.Threshold, o.Threshold + 1}
	windows := []time.Duration{o.Window / 2, o.Window, o.Window * 2}
	cells := make([]SensitivityCell, 0, 9)
	for _, threshold := range thresholds {
		if threshold < 2 || threshold > 100 {
			continue
		}
		for _, window := range windows {
			if window < time.Second || window > 24*time.Hour {
				continue
			}
			left, peak := 0, 0
			for right, e := range evidence {
				cutoff := e.Timestamp.Add(-window)
				for left <= right && evidence[left].Timestamp.Before(cutoff) {
					left++
				}
				if count := right - left + 1; count > peak {
					peak = count
				}
			}
			cells = append(cells, SensitivityCell{threshold, window, window.String(), peak, peak >= threshold})
		}
	}
	return cells
}
