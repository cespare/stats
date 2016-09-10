package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

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

	sr := newSummarizer(quants, *histBuckets)
	var nonNumeric int64
	argf.Init(flag.Args())
	for argf.Scan() {
		s := argf.String()
		if s == "" {
			continue
		}
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			nonNumeric++
			continue
		}
		sr.add(v)
	}
	if err := argf.Error(); err != nil {
		log.Fatal(err)
	}
	if nonNumeric > 0 {
		log.Printf("warning: found %d non-numeric lines of input", nonNumeric)
	}
	if sr.count == 0 {
		log.Println("no numbers given")
		return
	}
	s := sr.summarize()
	fmt.Println(s)
	if *printHist {
		fmt.Println(&s.hist)
	}
}

type summarizer struct {
	summary
	btree *b.Tree
}

func newSummarizer(quants []float64, numBuckets int) *summarizer {
	btree := b.TreeNew(func(a, b float64) int {
		if a < b {
			return -1
		}
		if a == b {
			return 0
		}
		return 1
	})
	sr := &summarizer{
		btree: btree,
		summary: summary{
			quants: make([]quantile, len(quants)),
			hist:   hist{buckets: make([]histBucket, numBuckets)},
		},
	}
	sort.Float64s(quants)
	for i, q := range quants {
		sr.quants[i].q = q
	}
	return sr
}

func (sr *summarizer) add(v float64) {
	if sr.count == 0 || v < sr.min {
		sr.min = v
	}
	if sr.count == 0 || v > sr.max {
		sr.max = v
	}
	sr.btree.Put(v, func(c int64, _ bool) (int64, bool) { return c + 1, true })
	sr.count++
}

func (sr *summarizer) summarize() *summary {
	it, err := sr.btree.SeekFirst()
	if err != nil {
		panic(err)
	}
	for i, q := range sr.quants {
		sr.quants[i].i = round(q.q * float64(sr.count-1))
	}
	var (
		qi   int
		bi   int
		i    int64
		rnge = sr.max - sr.min
	)
	// TODO: If the range is large, expand the bucketsize and start/end a
	// little bit to obtain integer boundaries.
	sr.bucketSize = rnge / float64(len(sr.buckets))
	sr.buckets[0].start = sr.min
	bucketLimit := sr.min + sr.bucketSize
	for {
		v, c, err := it.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		for j := 0; j < int(c); j++ {
			sr.sum += v
			sr.sumSquares += v * v
			for qi < len(sr.quants) && i == sr.quants[qi].i {
				sr.quants[qi].v = v
				qi++
			}
			i++
		}
		for v >= bucketLimit && bi < len(sr.buckets)-1 {
			bi++
			sr.buckets[bi].start = bucketLimit
			bucketLimit = sr.min + float64(bi+1)*sr.bucketSize
		}
		sr.buckets[bi].count += c
	}
	return &sr.summary
}

type summary struct {
	count      int64
	min        float64
	max        float64
	sum        float64
	sumSquares float64
	quants     []quantile
	hist
}

type quantile struct {
	q float64 // e.g., 0.9 for 90th percentile
	i int64   // index of quantile value
	v float64 // quantile value
}

type histBucket struct {
	start float64
	count int64
}

type hist struct {
	bucketSize float64
	buckets    []histBucket
}

func (s *summary) String() string {
	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 0, 0, 4, ' ', 0)

	n := float64(s.count)
	mean := s.sum / n
	stdev := math.Sqrt(n*s.sumSquares-(s.sum*s.sum)) / n

	fmt.Fprintf(tw, "count\t%d\n", s.count)
	fmt.Fprintf(tw, "min\t%g\n", s.min)
	fmt.Fprintf(tw, "max\t%g\n", s.max)
	fmt.Fprintf(tw, "mean\t%g\n", mean)
	fmt.Fprintf(tw, "std. dev.\t%g\n", stdev)
	for _, q := range s.quants {
		fmt.Fprintf(tw, "quantile %g\t%g\n", q.q, q.v)
	}

	tw.Flush()
	b := buf.Bytes()
	return string(b[:len(b)-1]) // drop the \n
}

const histBlocks = 70

func (h *hist) String() string {
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
		if after := utf8.RuneCountInString(label) - xPos - 1; after > labelSpaceAfter {
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
		after := labelSpaceAfter - utf8.RuneCountInString(labels[i]) + xPos + 1
		fmt.Fprintf(&buf, " %*s%s%*s │", before, "", labels[i], after, "")
		fmt.Fprint(&buf, bar((float64(b.count)/float64(maxCount))*histBlocks))
		fmt.Fprintf(&buf, " %d (%.3f%%)\n", b.count, 100*float64(b.count)/sum)
	}
	b := buf.Bytes()
	return string(b[:len(b)-1]) // drop the \n
}

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

func bar(n float64) string {
	eighths := int(round(n * 8))
	full := eighths / 8
	rem := eighths % 8
	return strings.Repeat(string(barEighths[8]), full) + string(barEighths[rem])
}

// assumes positive v
func round(v float64) int64 {
	return int64(v + 0.5)
}
