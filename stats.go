package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/cespare/argf"
	"github.com/cespare/stats/b"
)

type Stats struct {
	Count      float64
	min        float64
	max        float64
	sum        float64
	sumSquares float64
	sorted     []float64
}

func (s *Stats) Mean() float64 {
	return s.sum / s.Count
}

func (s *Stats) Stdev() float64 {
	return math.Sqrt(s.Count*s.sumSquares-(s.sum*s.sum)) / s.Count
}

func (s *Stats) Quant(q float64) float64 {
	if q < 0 || q > 1 {
		panic("bad quantile")
	}
	i := round((s.Count - 1) * q)
	return s.sorted[i]
}

func round(f float64) int { return int(f + 0.5) }

func printStat(name string, value float64) {
	fmt.Printf("%-15s %7.3f\n", name, value)
}

func StatsFromBtree(btree *b.Tree) *Stats {
	enum, err := btree.SeekFirst()
	if err != nil {
		panic(err)
	}
	s := &Stats{sorted: make([]float64, 0, btree.Len())}
	for {
		k, c, err := enum.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		if s.Count == 0 || k < s.min {
			s.min = k
		}
		if s.Count == 0 || k > s.max {
			s.max = k
		}
		for i := 0; i < int(c); i++ {
			s.sorted = append(s.sorted, k)
			s.Count++
			s.sum += k
			s.sumSquares += k * k
		}
	}
	return s
}

func main() {
	quantStr := flag.String("quantiles", "0.5,0.9,0.99", "Quantiles to record")
	printHist := flag.Bool("hist", false, "Print a histogram")
	histBuckets := flag.Int("buckets", 10, "How many buckets for the histogram")
	flag.Parse()

	if *histBuckets <= 1 {
		fmt.Fprintf(os.Stderr, "%d is an invalid number of buckets\n", *histBuckets)
	}

	var quants []float64
	for _, qs := range strings.Split(*quantStr, ",") {
		qs = strings.TrimSpace(qs)
		f, err := strconv.ParseFloat(qs, 64)
		if err != nil {
			fatal(err)
		}
		if f <= 0 || f >= 1 {
			fatal(fmt.Errorf("quantile values must be in (0, 1); got %f", f))
		}
		quants = append(quants, f)
	}

	btree := NewBTree()
	var nonNumericFound int64
	argf.Init(flag.Args())
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
		btree.Put(v, func(c uint, _ bool) (newC uint, write bool) { return c + 1, true })
	}
	if err := argf.Error(); err != nil {
		fatal(err)
	}
	if nonNumericFound > 0 {
		fmt.Fprintf(os.Stderr, "Warning: found %d non-numeric lines of input\n", nonNumericFound)
	}
	if btree.Len() == 0 {
		fmt.Fprintln(os.Stderr, "No numbers given")
		return
	}
	stats := StatsFromBtree(btree)
	printStat("count", stats.Count)
	printStat("min", stats.min)
	printStat("max", stats.max)
	printStat("mean", stats.Mean())
	printStat("std. dev.", stats.Stdev())
	for _, q := range quants {
		name := fmt.Sprintf("quantile %f", q)
		name = strings.TrimRight(name, "0")
		printStat(name, stats.Quant(q))
	}
	if *printHist {
		fmt.Println(stats.Hist(*histBuckets))
	}
}

func fatal(err error) {
	fmt.Println(err)
	os.Exit(1)
}

func NewBTree() *b.Tree {
	return b.TreeNew(func(a, b float64) int {
		if a < b {
			return -1
		}
		if a == b {
			return 0
		}
		return 1
	})
}
