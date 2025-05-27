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

func NewZDetector(n int) *ZDetector {
	return &ZDetector{
		window: make([]float64, 0, n),
		size:   n,
	}
}

func (d *ZDetector) Add(val float64) bool {
	if len(d.window) < d.size {
		d.window = append(d.window, val)
		return false
	}

	// Compute mean
	mean, err := stats.Mean(d.window)
	if err != nil {
		fmt.Printf("warning: could not compute mean: %v\n", err)
		d.window = append(d.window[1:], val)
		return false
	}
	// Compute standard deviation
	std, err := stats.StandardDeviation(d.window)
	if err != nil {
		fmt.Printf("warning: could not compute stddev: %v\n", err)
		d.window = append(d.window[1:], val)
		return false
	}

	isAnom := std > 0 && math.Abs(val-mean)/std > 0.5
	d.window = append(d.window[1:], val)
	return isAnom
}
