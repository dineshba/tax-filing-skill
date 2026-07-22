// Package fa parses Fidelity Open/Closed Lots CSV exports and computes the
// Indian ITR-2 Schedule FA (A3) foreign-equity holdings schedule.
package fa

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// OpenLot is a single row from the Fidelity "View Open Lots" CSV. It represents
// shares still held as of the report (snapshot) date.
type OpenLot struct {
	DateAcquired      time.Time
	Quantity          float64
	CostBasis         float64 // total USD cost basis of the lot
	CostBasisPerShare float64 // USD cost per share
	Value             float64 // current market value (USD) at report date
	GrantDate         *time.Time
	ShareSource       string // e.g. "SP" (ESPP) or "DO" (RSU)
	HoldingPeriod     string // "Short" / "Long"
}

// ClosedLot is a single row from the Fidelity "View Closed Lots" CSV. It
// represents a quantity of shares that was sold or transferred.
type ClosedLot struct {
	DateAcquired time.Time
	DateSold     time.Time
	Quantity     float64
	Proceeds     float64 // USD proceeds
	CostBasis    float64 // USD cost basis of the sold quantity
	GainLoss     float64
	Term         string // "LONG" / "SHORT"
}

// openLotDateLayout matches dates like "Jun-15-2026".
const openLotDateLayout = "Jan-02-2006"

// closedLotDateLayout matches dates like "May/15/2023" (after case normalisation).
const closedLotDateLayout = "Jan/02/2006"

// ParseOpenLots reads the Open Lots CSV from r.
func ParseOpenLots(r io.Reader) ([]OpenLot, error) {
	records, err := readCSV(r)
	if err != nil {
		return nil, err
	}
	var lots []OpenLot
	for i, rec := range records {
		if isSkippableRow(rec) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(rec[0]), "Date acquired") {
			continue // header
		}
		if len(rec) < 11 {
			return nil, fmt.Errorf("open lots row %d: expected 11 columns, got %d", i+1, len(rec))
		}
		acq, err := parseDate(rec[0], openLotDateLayout)
		if err != nil {
			return nil, fmt.Errorf("open lots row %d: date acquired: %w", i+1, err)
		}
		qty, err := parseFloat(rec[1])
		if err != nil {
			return nil, fmt.Errorf("open lots row %d: quantity: %w", i+1, err)
		}
		cost, _ := parseFloat(rec[2])
		costPS, _ := parseFloat(rec[3])
		val, _ := parseFloat(rec[4])
		var grant *time.Time
		if g, err := parseDate(rec[8], openLotDateLayout); err == nil {
			grant = &g
		}
		lots = append(lots, OpenLot{
			DateAcquired:      acq,
			Quantity:          qty,
			CostBasis:         cost,
			CostBasisPerShare: costPS,
			Value:             val,
			GrantDate:         grant,
			ShareSource:       strings.TrimSpace(rec[9]),
			HoldingPeriod:     strings.TrimSpace(rec[10]),
		})
	}
	return lots, nil
}

// ParseClosedLots reads the Closed Lots CSV from r.
func ParseClosedLots(r io.Reader) ([]ClosedLot, error) {
	records, err := readCSV(r)
	if err != nil {
		return nil, err
	}
	var lots []ClosedLot
	for i, rec := range records {
		if isSkippableRow(rec) {
			continue
		}
		// The Fidelity export wraps the "Date sold" header in HTML markup, so
		// detect the header by its first cell instead of an exact match.
		if strings.EqualFold(strings.TrimSpace(rec[0]), "Date acquired") {
			continue // header
		}
		if len(rec) < 7 {
			return nil, fmt.Errorf("closed lots row %d: expected 7 columns, got %d", i+1, len(rec))
		}
		acq, err := parseDate(rec[0], closedLotDateLayout)
		if err != nil {
			return nil, fmt.Errorf("closed lots row %d: date acquired: %w", i+1, err)
		}
		sold, err := parseDate(rec[2], closedLotDateLayout)
		if err != nil {
			return nil, fmt.Errorf("closed lots row %d: date sold: %w", i+1, err)
		}
		qty, err := parseFloat(rec[1])
		if err != nil {
			return nil, fmt.Errorf("closed lots row %d: quantity: %w", i+1, err)
		}
		proceeds, _ := parseFloat(rec[3])
		cost, _ := parseFloat(rec[4])
		gain, _ := parseFloat(rec[5])
		lots = append(lots, ClosedLot{
			DateAcquired: acq,
			DateSold:     sold,
			Quantity:     qty,
			Proceeds:     proceeds,
			CostBasis:    cost,
			GainLoss:     gain,
			Term:         strings.TrimSpace(rec[6]),
		})
	}
	return lots, nil
}

// readCSV reads all records allowing a variable number of fields per row, since
// the Fidelity exports contain trailing junk rows with a different arity.
func readCSV(r io.Reader) ([][]string, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	// The Fidelity Closed Lots export wraps a header cell in raw HTML that
	// contains bare double quotes, so tolerate non-standard quoting.
	cr.LazyQuotes = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	// Fidelity exports are UTF-8 with a leading byte-order mark; strip it from
	// the first cell so header detection and date parsing work.
	if len(records) > 0 && len(records[0]) > 0 {
		records[0][0] = strings.TrimPrefix(records[0][0], "\ufeff")
	}
	return records, nil
}

// isSkippableRow reports whether a CSV row is empty, blank, or the trailing
// "The values are displayed in USD" footer line.
func isSkippableRow(rec []string) bool {
	if len(rec) == 0 {
		return true
	}
	joined := strings.TrimSpace(strings.Join(rec, ""))
	if joined == "" {
		return true
	}
	if strings.Contains(strings.ToLower(joined), "values are displayed") {
		return true
	}
	return false
}

// parseDate normalises casing (Fidelity closed lots use UPPER-case months) and
// parses using the given layout. A blank or "-" value is treated as an error so
// callers can decide whether the field is optional.
func parseDate(s, layout string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	return time.Parse(layout, normaliseMonthCase(s))
}

// normaliseMonthCase converts the alphabetic month token to Title case so that
// "MAY/15/2023" parses with a "Jan/02/2006" layout.
func normaliseMonthCase(s string) string {
	var b strings.Builder
	prevAlpha := false
	for _, r := range s {
		isAlpha := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		switch {
		case isAlpha && !prevAlpha:
			// first letter of a token -> upper
			if r >= 'a' && r <= 'z' {
				r -= 'a' - 'A'
			}
		case isAlpha && prevAlpha:
			// subsequent letters -> lower
			if r >= 'A' && r <= 'Z' {
				r += 'a' - 'A'
			}
		}
		b.WriteRune(r)
		prevAlpha = isAlpha
	}
	return b.String()
}

// parseFloat parses a USD/quantity cell, treating blank and "-" as 0.
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" {
		return 0, nil
	}
	s = strings.ReplaceAll(s, ",", "")
	return strconv.ParseFloat(s, 64)
}
