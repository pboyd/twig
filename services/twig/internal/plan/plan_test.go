package plan_test

import (
	"testing"

	"github.com/pboyd/twig/services/twig/internal/plan"
)

func TestOverlap(t *testing.T) {
	cases := []struct {
		name   string
		aStart int
		aDur   int
		bStart int
		bDur   int
		want   bool
	}{
		{"full overlap B inside A", 10, 60, 20, 20, true},
		{"full overlap A inside B", 20, 20, 10, 60, true},
		{"partial overlap front", 10, 30, 25, 30, true},
		{"partial overlap back", 25, 30, 10, 30, true},
		{"touching A end equals B start", 10, 30, 40, 30, false},
		{"touching B end equals A start", 40, 30, 10, 30, false},
		{"disjoint A before B", 0, 30, 60, 30, false},
		{"disjoint B before A", 60, 30, 0, 30, false},
		{"same interval", 60, 30, 60, 30, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := plan.Overlap(c.aStart, c.aDur, c.bStart, c.bDur)
			if got != c.want {
				t.Errorf("Overlap(%d,%d,%d,%d) = %v, want %v", c.aStart, c.aDur, c.bStart, c.bDur, got, c.want)
			}
		})
	}
}
