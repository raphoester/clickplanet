package cpscripts

import (
	_ "embed"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

//go:embed historic.txt
var vendored string

const minRanges = 300

var historic = mustTableOf(vendored)

func Historic(r rune) bool {
	return unicode.Is(historic, r)
}

func mustTableOf(data string) *unicode.RangeTable {
	table, err := tableOf(data)
	if err != nil {
		panic(fmt.Sprintf("historic.txt: %v: run make historic-scripts", err))
	}
	return table
}

func tableOf(data string) (*unicode.RangeTable, error) {
	table := &unicode.RangeTable{}
	ranges := 0
	last := rune(-1)
	for line := range strings.Lines(data) {
		fields, _, _ := strings.Cut(line, "#")
		if strings.TrimSpace(fields) == "" {
			continue
		}
		span, types, found := strings.Cut(fields, ";")
		if !found {
			return nil, fmt.Errorf("%q has no type", line)
		}
		if !slices.Contains(strings.Fields(types), "Exclusion") {
			return nil, fmt.Errorf("%q is not Exclusion", line)
		}
		lo, hi, err := spanOf(strings.TrimSpace(span))
		if err != nil {
			return nil, fmt.Errorf("%q: %w", line, err)
		}
		if lo <= last {
			return nil, fmt.Errorf("%q is out of order", line)
		}
		add(table, lo, hi)
		last = hi
		ranges++
	}
	if ranges < minRanges {
		return nil, fmt.Errorf("%d ranges, under %d", ranges, minRanges)
	}
	return table, nil
}

func spanOf(value string) (rune, rune, error) {
	first, end, isRange := strings.Cut(value, "..")
	lo, err := codePointOf(first)
	if err != nil {
		return 0, 0, err
	}
	if !isRange {
		return lo, lo, nil
	}
	hi, err := codePointOf(end)
	if err != nil {
		return 0, 0, err
	}
	if hi < lo {
		return 0, 0, fmt.Errorf("%X is before %X", hi, lo)
	}
	return lo, hi, nil
}

func codePointOf(value string) (rune, error) {
	n, err := strconv.ParseUint(value, 16, 32)
	if err != nil || n > unicode.MaxRune {
		return 0, fmt.Errorf("%q is not a code point", value)
	}
	return rune(n), nil
}

func add(table *unicode.RangeTable, lo, hi rune) {
	if lo <= 0xFFFF {
		table.R16 = append(table.R16, unicode.Range16{Lo: uint16(lo), Hi: uint16(min(hi, 0xFFFF)), Stride: 1})
		lo = 0x10000
	}
	if hi >= lo {
		table.R32 = append(table.R32, unicode.Range32{Lo: uint32(lo), Hi: uint32(hi), Stride: 1})
	}
}
