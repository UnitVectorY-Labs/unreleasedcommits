package main

import (
	"fmt"
	"math"
	"testing"
)

func TestSpectrumAnchorsRemainUnchanged(t *testing.T) {
	for _, test := range []struct {
		value float64
		color string
	}{{0, "#10b981"}, {0.5, "#fbbf24"}, {1, "#ef4444"}} {
		if got := getColorForValue(test.value); got != test.color {
			t.Errorf("spectrum %v = %s, want %s", test.value, got, test.color)
		}
	}
}

func TestSpectrumTextContrastAcrossRamp(t *testing.T) {
	luminance := func(color string) float64 {
		var r, g, b int
		if _, err := fmt.Sscanf(color, "#%02x%02x%02x", &r, &g, &b); err != nil {
			t.Fatal(err)
		}
		channel := func(v int) float64 {
			x := float64(v) / 255
			if x <= 0.04045 {
				return x / 12.92
			}
			return math.Pow((x+0.055)/1.055, 2.4)
		}
		return .2126*channel(r) + .7152*channel(g) + .0722*channel(b)
	}
	for i := 0; i <= 100; i++ {
		value := float64(i) / 100
		a, b := luminance(getColorForValue(value)), luminance(getTextColor(value))
		if a < b {
			a, b = b, a
		}
		if ratio := (a + .05) / (b + .05); ratio < 4.5 {
			t.Errorf("spectrum %.2f text contrast %.2f", value, ratio)
		}
	}
}
