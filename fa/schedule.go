package fa

import (
	"errors"
	"sort"
	"time"
)

var errNoPrices = errors.New("no daily price series supplied; peak and closing values omitted (pass -prices <ticker>.csv)")

// FALotRow is one Schedule FA (A3) row: the foreign-equity holding for a single
// acquisition date over the target calendar year.
type FALotRow struct {
	AcquisitionDate time.Time
	GrantDate       *time.Time

	// USD figures, derived from the CSVs (dividends require config).
	InitialShares    float64 // shares originally acquired on this date
	InitialValueUSD  float64 // original USD cost basis
	SoldInYearShares float64 // shares of this lot sold during the calendar year
	GrossProceedsUSD float64 // USD proceeds from those in-year sales
	ClosingShares    float64 // shares still held at Dec-31 of the year
	GrossDividendUSD float64 // dividends credited on this lot during the year

	// INR figures, populated when a Valuer is supplied.
	InitialValueINR  float64
	PeakValueINR     float64
	PeakDate         time.Time // date of the intra-year peak holding value
	ClosingValueINR  float64
	GrossDividendINR float64
	GrossProceedsINR float64
}

// ScheduleFA is the computed Schedule FA A3 schedule for a calendar year.
type ScheduleFA struct {
	Year     int
	HasINR   bool
	Rows     []FALotRow
	Warnings []string

	TotalInitialValueINR float64
	TotalPeakValueINR    float64
	TotalClosingValueINR float64
	TotalDividendINR     float64
	TotalProceedsINR     float64

	TotalInitialValueUSD float64
	TotalDividendUSD     float64
	TotalProceedsUSD     float64
}

// sale is a single disposal of shares from a lot.
type sale struct {
	date     time.Time
	qty      float64
	proceeds float64
	cost     float64 // USD cost basis of the disposed shares
}

// lotAgg accumulates all activity for one acquisition date.
type lotAgg struct {
	acqDate       time.Time
	grant         *time.Time
	initialShares float64
	initialUSD    float64
	openHeld      float64 // shares still held at the snapshot (report) date
	sales         []sale
}

// heldAt returns the shares of this lot still held at the end of day d, assuming
// d is on/after the acquisition date. Shares sold after d were still held at d.
func (a *lotAgg) heldAt(d time.Time) float64 {
	held := a.openHeld
	for _, s := range a.sales {
		if s.date.After(d) {
			held += s.qty
		}
	}
	return held
}

// dayKey collapses a time to its calendar date for grouping by acquisition date.
func dayKey(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// ComputeScheduleFA builds the Schedule FA A3 schedule for the given calendar
// year. v may be nil, in which case only USD figures are populated.
//
// INR conversion uses the SBI TT buying rate on each relevant date (Explanation
// to Rule 26). The intra-year peak is the maximum, over every trading day the
// lot is held, of sharesHeld(day) x USD close(day) x SBI TT(day) — because the
// price-high and FX-high can fall on different days.
//
// Reconstruction of year-end holdings: the open-lots CSV is a snapshot of
// current holdings, so shares held at Dec-31 of year Y equal the shares still
// held now plus any closed-lot shares that were sold AFTER Dec-31 of year Y.
func ComputeScheduleFA(open []OpenLot, closed []ClosedLot, year int, v *Valuer) (ScheduleFA, error) {
	yearEnd := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)
	yearStart := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)

	groups := map[time.Time]*lotAgg{}
	get := func(d time.Time) *lotAgg {
		k := dayKey(d)
		a, ok := groups[k]
		if !ok {
			a = &lotAgg{acqDate: k}
			groups[k] = a
		}
		return a
	}

	sched := ScheduleFA{Year: year, HasINR: v != nil}
	addWarn := func(err error) {
		if err != nil {
			sched.Warnings = append(sched.Warnings, err.Error())
		}
	}

	for _, l := range open {
		a := get(l.DateAcquired)
		a.initialShares += l.Quantity
		a.initialUSD += l.CostBasis
		a.openHeld += l.Quantity
		if a.grant == nil && l.GrantDate != nil {
			a.grant = l.GrantDate
		}
	}
	for _, l := range closed {
		a := get(l.DateAcquired)
		a.initialShares += l.Quantity
		a.initialUSD += l.CostBasis
		a.sales = append(a.sales, sale{date: l.DateSold, qty: l.Quantity, proceeds: l.Proceeds, cost: l.CostBasis})
	}

	var dividends []DividendEvent
	if v != nil {
		dividends = v.DividendsInYear(year)
	}

	keys := make([]time.Time, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Before(keys[j]) })

	for _, k := range keys {
		a := groups[k]
		if a.acqDate.After(yearEnd) {
			continue
		}

		closingShares := a.heldAt(yearEnd)
		var soldInYear, proceedsUSD, proceedsINR float64
		for _, s := range a.sales {
			if !s.date.Before(yearStart) && !s.date.After(yearEnd) {
				soldInYear += s.qty
				proceedsUSD += s.proceeds
				if v != nil {
					// Schedule FA "Total gross proceeds" column is valued from the
					// original cost basis of the disposed shares at the SBI rate on
					// the sale date (matches the user's reference figures).
					if r, err := v.RateOn(s.date); err != nil {
						addWarn(err)
					} else {
						proceedsINR += s.cost * r
					}
				}
			}
		}

		var divUSD, divINR float64
		for _, d := range dividends {
			if a.acqDate.After(d.ExDate) {
				continue
			}
			held := a.heldAt(d.ExDate)
			if held <= 0 {
				continue
			}
			divUSD += held * d.USDPerShare
			if r, err := v.RateOn(d.ExDate); err != nil {
				addWarn(err)
			} else {
				divINR += held * d.USDPerShare * r
			}
		}

		if closingShares <= 0 && soldInYear == 0 && divUSD == 0 {
			continue
		}

		row := FALotRow{
			AcquisitionDate:  a.acqDate,
			GrantDate:        a.grant,
			InitialShares:    a.initialShares,
			InitialValueUSD:  a.initialUSD,
			SoldInYearShares: soldInYear,
			GrossProceedsUSD: proceedsUSD,
			ClosingShares:    closingShares,
			GrossDividendUSD: divUSD,
		}

		if v != nil {
			if r, err := v.RateOn(a.acqDate); err != nil {
				addWarn(err)
			} else {
				row.InitialValueINR = a.initialUSD * r
			}
			row.GrossProceedsINR = proceedsINR
			row.GrossDividendINR = divINR

			// Intra-year peak from the daily price series.
			peakStart := a.acqDate
			if peakStart.Before(yearStart) {
				peakStart = yearStart
			}
			for _, day := range v.TradingDays(peakStart, yearEnd) {
				held := a.heldAt(day)
				if held <= 0 {
					continue
				}
				price, perr := v.PriceOn(day)
				rate, rerr := v.RateOn(day)
				if perr != nil || rerr != nil {
					continue
				}
				val := held * price * rate
				if val > row.PeakValueINR {
					row.PeakValueINR = val
					row.PeakDate = day
				}
			}
			if v.Prices == nil {
				addWarn(errNoPrices)
			}

			// Closing value = shares held on Dec-31 x close(Dec-31) x rate(Dec-31).
			if closingShares > 0 && v.Prices != nil {
				price, perr := v.PriceOn(yearEnd)
				rate, rerr := v.RateOn(yearEnd)
				if perr != nil {
					addWarn(perr)
				} else if rerr != nil {
					addWarn(rerr)
				} else {
					row.ClosingValueINR = closingShares * price * rate
				}
			}
		}

		sched.Rows = append(sched.Rows, row)
		sched.TotalInitialValueINR += row.InitialValueINR
		sched.TotalPeakValueINR += row.PeakValueINR
		sched.TotalClosingValueINR += row.ClosingValueINR
		sched.TotalDividendINR += row.GrossDividendINR
		sched.TotalProceedsINR += row.GrossProceedsINR
		sched.TotalInitialValueUSD += row.InitialValueUSD
		sched.TotalDividendUSD += row.GrossDividendUSD
		sched.TotalProceedsUSD += row.GrossProceedsUSD
	}

	return sched, nil
}
