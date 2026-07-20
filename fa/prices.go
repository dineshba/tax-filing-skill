package fa

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

// pricePoint is a daily USD close price.
type pricePoint struct {
	day   time.Time
	price float64
}

// PriceSeries is a date-ordered series of daily USD close prices for a ticker,
// used to compute the true intra-year peak holding value and the year-end
// closing value. It accepts the "Date,Price,Open,High,Low,..." layout exported
// by investing.com / Stooq / Yahoo.
type PriceSeries struct {
	points []pricePoint // sorted ascending by day
}

// priceDateLayouts are accepted formats for the Date column.
var priceDateLayouts = []string{"2006-01-02", "01/02/2006", "02-Jan-2006", "02 Jan 2006", "Jan 02, 2006"}

// LoadPriceSeries parses a daily price CSV from r. It reads the "Date" and
// "Price" (daily close) columns by header, falling back to the first two columns.
func LoadPriceSeries(r io.Reader) (*PriceSeries, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read price series: %w", err)
	}
	dateCol, priceCol := 0, 1
	byDay := map[time.Time]float64{}
	for i, rec := range records {
		if len(rec) < 2 {
			continue
		}
		if i == 0 && strings.EqualFold(strings.TrimSpace(rec[0]), "Date") {
			if c := indexOf(rec, "Price"); c >= 0 {
				priceCol = c
			} else if c := indexOf(rec, "Close"); c >= 0 {
				priceCol = c
			}
			continue
		}
		if priceCol >= len(rec) || dateCol >= len(rec) {
			continue
		}
		day, err := parsePriceDate(strings.TrimSpace(rec[dateCol]))
		if err != nil {
			continue
		}
		price, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(rec[priceCol]), ",", ""), 64)
		if err != nil || price <= 0 {
			continue
		}
		byDay[day] = price
	}
	if len(byDay) == 0 {
		return nil, fmt.Errorf("no usable price rows found")
	}
	points := make([]pricePoint, 0, len(byDay))
	for d, p := range byDay {
		points = append(points, pricePoint{day: d, price: p})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].day.Before(points[j].day) })
	return &PriceSeries{points: points}, nil
}

// PriceOn returns the close price for d: an exact match, else the nearest price
// on/before d.
func (p *PriceSeries) PriceOn(d time.Time) (float64, error) {
	day := dayKey(d)
	idx := sort.Search(len(p.points), func(i int) bool { return p.points[i].day.After(day) })
	if idx == 0 {
		return 0, fmt.Errorf("no price available on or before %s", d.Format("2006-01-02"))
	}
	return p.points[idx-1].price, nil
}

// DaysIn returns the trading days (with a price) in the inclusive range [from, to].
func (p *PriceSeries) DaysIn(from, to time.Time) []time.Time {
	from, to = dayKey(from), dayKey(to)
	var out []time.Time
	for _, pt := range p.points {
		if !pt.day.Before(from) && !pt.day.After(to) {
			out = append(out, pt.day)
		}
	}
	return out
}

func parsePriceDate(s string) (time.Time, error) {
	raw := strings.TrimSpace(s)
	for _, layout := range priceDateLayouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return dayKey(t), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable price date %q", raw)
}
