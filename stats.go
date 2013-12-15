package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/bmizerany/perks/quantile"
	"github.com/cespare/argf"
)

var quants = []float64{0.5, 0.9, 0.99}

type Stats struct {
	Count      float64
	Sum        float64
	SumSquares float64
	Quantiles  *quantile.Stream
}

func (s *Stats) Insert(v float64) {
	s.Count++
	s.Sum += v
	s.SumSquares += (v * v)
	s.Quantiles.Insert(v)
}

func (s *Stats) Mean() float64 {
	return s.Sum / s.Count
}

func (s *Stats) Stdev() float64 {
	return math.Sqrt(s.Count*s.SumSquares-(s.Sum*s.Sum)) / s.Count
}

func printStat(name string, value float64) {
	fmt.Printf("%-15s %7.3f\n", name, value)
}

func main() {
	stats := &Stats{
		Quantiles: quantile.NewTargeted(quants...),
	}
	var nonNumericFound int64
	for argf.Scan() {
		s := argf.String()
		if s == "" {
			continue
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			nonNumericFound++
			continue
		}
		stats.Insert(v)
	}
	if err := argf.Error(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if nonNumericFound > 0 {
		fmt.Fprintf(os.Stderr, "Warning: found %d non-numeric lines of input\n", nonNumericFound)
		return
	}
	if stats.Count == 0 {
		fmt.Fprintln(os.Stderr, "Warning: no numbers given")
		return
	}
	printStat("count", stats.Count)
	printStat("mean", stats.Mean())
	printStat("std. dev.", stats.Stdev())
	for _, q := range quants {
		name := fmt.Sprintf("quantile %f", q)
		name = strings.TrimRight(name, "0")
		printStat(name, stats.Quantiles.Query(q))
	}
}
