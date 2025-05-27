package monitor

import (
	"fmt"
	"math"

	"github.com/montanaflynn/stats"
)

type ZDetector struct {
	window []float64
	size   int
}

// NewZDetector builds a windowed z-score detector of size n.
func NewZDetector(n int) *ZDetector {
	return &ZDetector{
		window: make([]float64, 0, n),
		size:   n,
	}
}

// Add inserts val into the window and returns true if it's a >3σ anomaly.
func (d *ZDetector) Add(val float64) bool {
	if len(d.window) < d.size {
		d.window = append(d.window, val)
		return false
	}
	mean, std, err := stats.MeanStdDev(d.window)
	if err != nil {
		// Fallback: no anomaly detection if stats calculation fails
		fmt.Printf("warning: could not compute stats: %v\n", err)
		d.window = append(d.window[1:], val)
		return false
	}
	isAnom := std > 0 && math.Abs(val-mean)/std > 3
	// slide window
	d.window = append(d.window[1:], val)
	return isAnom
}
