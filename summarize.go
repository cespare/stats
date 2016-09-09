package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/cespare/argf"
	"github.com/cespare/stats/b"
)

func summarize(args []string) {
	fs := flag.NewFlagSet("summarize", flag.ExitOnError)
	quantStr := fs.String("quantiles", "0.5,0.9,0.99", "Quantiles to record")
	printHist := fs.Bool("hist", false, "Print a histogram")
	histBuckets := fs.Int("buckets", 10, "How many buckets for the histogram")
	fs.Parse(args)

	if *histBuckets <= 1 {
		log.Fatalf("%d is an invalid number of buckets", *histBuckets)
	}

	var quants []float64
	for _, qs := range strings.Split(*quantStr, ",") {
		qs = strings.TrimSpace(qs)
		f, err := strconv.ParseFloat(qs, 64)
		if err != nil {
			log.Fatal(err)
		}
		if f <= 0 || f >= 1 {
			log.Fatalf("quantile values must be in (0, 1); got %g", f)
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
		log.Fatal(err)
	}
	if nonNumericFound > 0 {
		log.Printf("warning: found %d non-numeric lines of input", nonNumericFound)
	}
	if btree.Len() == 0 {
		log.Println("no numbers given")
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

type Bucket struct {
	start float64
	count uint64
}

type Hist struct {
	bucketSize float64
	buckets    []Bucket
}

func (s *Stats) Hist(n int) *Hist {
	h := &Hist{buckets: make([]Bucket, n)}
	rnge := s.max - s.min
	h.bucketSize = rnge / float64(n)
	i := 0
	limit := s.min + h.bucketSize
	h.buckets[0].start = s.min
	for j := 0; j < len(s.sorted); {
		v := s.sorted[j]
		if v >= limit && i < len(h.buckets)-1 {
			i++
			h.buckets[i].start = limit
			limit = s.min + float64(i+1)*(rnge/float64(n))
			continue
		}
		h.buckets[i].count++
		j++
	}
	return h
}

const histBlocks = 70

func (h *Hist) String() string {
	// TODO: if the range is large, expand the bucketsize and start/end a bit to get integer boundaries.
	labels := make([]string, len(h.buckets))
	labelSpaceBefore := 0
	labelSpaceAfter := 0
	var maxCount, sum float64
	for i, b := range h.buckets {
		sum += float64(b.count)
		s := "<"
		if i == len(h.buckets)-1 {
			s = "≤"
		}
		label := fmt.Sprintf("%.3g ≤ x %s %.3g", b.start, s, b.start+h.bucketSize)
		xPos := runeIndex(label, 'x')
		if xPos > labelSpaceBefore {
			labelSpaceBefore = xPos
		}
		if after := runeLen(label) - xPos - 1; after > labelSpaceAfter {
			labelSpaceAfter = after
		}
		labels[i] = label
		if f := float64(b.count); f > maxCount {
			maxCount = f
		}
	}

	var buf bytes.Buffer
	for i, b := range h.buckets {
		xPos := runeIndex(labels[i], 'x')
		before := labelSpaceBefore - xPos
		after := labelSpaceAfter - runeLen(labels[i]) + xPos + 1
		fmt.Fprintf(&buf, " %*s%s%*s │", before, "", labels[i], after, "")
		fmt.Fprint(&buf, makeBar((float64(b.count)/float64(maxCount))*histBlocks))
		fmt.Fprintf(&buf, " %d (%.3f%%)\n", b.count, 100*float64(b.count)/sum)
	}
	b := buf.Bytes()
	return string(b[:len(b)-1]) // drop the \n
}

func runeLen(s string) int { return len([]rune(s)) }

func runeIndex(s string, r rune) int {
	for i, r2 := range []rune(s) {
		if r2 == r {
			return i
		}
	}
	return -1
}

var barEighths = [9]rune{
	' ', // empty
	'▏',
	'▎',
	'▍',
	'▌',
	'▋',
	'▊',
	'▉',
	'█', // full
}

func makeBar(n float64) string {
	eighths := round(n * 8)
	full := eighths / 8
	rem := eighths % 8
	return strings.Repeat(string(barEighths[8]), full) + string(barEighths[rem])
}
