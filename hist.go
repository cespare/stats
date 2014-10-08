package main

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	histBlocks = 70
)

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
		if v > limit && i < len(h.buckets)-1 {
			i++
			h.buckets[i].start = limit
			limit = s.min + float64(i)*(rnge/float64(n))
			continue
		}
		h.buckets[i].count++
		j++
	}
	return h
}

func (h *Hist) String() string {
	// TODO: if the range is large, expand the bucketsize and start/end a bit to get integer boundaries.
	labels := make([]string, len(h.buckets))
	labelSpaceBefore := 0
	labelSpaceAfter := 0
	var maxCount float64
	for i, b := range h.buckets {
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
		fmt.Fprintf(&buf, " %d\n", b.count)
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
