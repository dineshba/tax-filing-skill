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

// lotAgg accumulates all activity for one tax lot. Each open-lot line in the
// Fidelity export becomes its own lotAgg (so lots sharing an acquisition date
// are reported on separate rows); a lot that was fully sold before the snapshot
// has no open line and is represented by a synthetic sold-out lotAgg per date.
type lotAgg struct {
	acqDate       time.Time
	grant         *time.Time
	initialShares float64
	initialUSD    float64
	openHeld      float64 // shares still held at the snapshot (report) date
	sales         []sale
	seq           int // input order, for stable per-date row ordering
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

	sched := ScheduleFA{Year: year, HasINR: v != nil}
	addWarn := func(err error) {
		if err != nil {
			sched.Warnings = append(sched.Warnings, err.Error())
		}
	}

	// Each open-lot line is reported as its own row; lots sharing an
	// acquisition date are kept separate. Sales are attached to a held lot of
	// the same acquisition date, or to a synthetic sold-out lot when the whole
	// lot was disposed before the snapshot.
	var lots []*lotAgg
	openByDate := map[time.Time][]*lotAgg{}
	soldOutByDate := map[time.Time]*lotAgg{}
	seq := 0

	for _, l := range open {
		k := dayKey(l.DateAcquired)
		a := &lotAgg{
			acqDate:       k,
			grant:         l.GrantDate,
			initialShares: l.Quantity,
			initialUSD:    l.CostBasis,
			openHeld:      l.Quantity,
			seq:           seq,
		}
		seq++
		lots = append(lots, a)
		openByDate[k] = append(openByDate[k], a)
	}
	for _, l := range closed {
		k := dayKey(l.DateAcquired)
		var a *lotAgg
		switch {
		case len(openByDate[k]) > 0:
			a = openByDate[k][0]
		case soldOutByDate[k] != nil:
			a = soldOutByDate[k]
		default:
			a = &lotAgg{acqDate: k, seq: seq}
			seq++
			soldOutByDate[k] = a
			lots = append(lots, a)
		}
		a.initialShares += l.Quantity
		a.initialUSD += l.CostBasis
		a.sales = append(a.sales, sale{date: l.DateSold, qty: l.Quantity, proceeds: l.Proceeds, cost: l.CostBasis})
	}

	var dividends []DividendEvent
	if v != nil {
		dividends = v.DividendsInYear(year)
	}

	sort.Slice(lots, func(i, j int) bool {
		if !lots[i].acqDate.Equal(lots[j].acqDate) {
			return lots[i].acqDate.Before(lots[j].acqDate)
		}
		// Within an acquisition date, match the reference ordering, which lists
		// same-date open lots in reverse of their order in the Fidelity export.
		return lots[i].seq > lots[j].seq
	})

	for _, a := range lots {
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
				// "Initial Value of the Investment" values the shares still held
				// at fair market value on the acquisition date (shares x USD close
				// x SBI TT rate) and the shares already sold at their original cost
				// basis. For RSUs the cost basis already equals FMV; for discounted
				// ESPP lots FMV is the correct cost of acquisition for the retained
				// shares. Fall back to the CSV cost basis of the held shares when no
				// price is available for the acquisition date.
				var soldCostUSD float64
				for _, s := range a.sales {
					soldCostUSD += s.cost
				}
				heldUSD := a.initialUSD - soldCostUSD // held shares' cost basis
				if v.Prices != nil {
					if p, perr := v.PriceOn(a.acqDate); perr == nil {
						heldUSD = a.openHeld * p
					}
				}
				initUSD := heldUSD + soldCostUSD
				row.InitialValueUSD = initUSD
				row.InitialValueINR = initUSD * r
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
