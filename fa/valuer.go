package fa

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Valuer supplies everything needed to value Schedule FA rows in INR: SBI TT
// buying rates (USD->INR), a daily USD price series (for the intra-year peak and
// year-end closing value), and dividend events.
//
// Prices may be nil, in which case peak and closing values cannot be computed
// and are reported as warnings.
type Valuer struct {
	Rates     *SBIRateSeries
	Prices    *PriceSeries
	Dividends []DividendEvent
}

// DividendConfig is the JSON shape for the optional dividends/overrides file.
type DividendConfig struct {
	Dividends []DividendEvent `json:"dividends"`
	// RateOverrides maps "YYYY-MM-DD" to a manual SBI TTBR, for acquisition
	// dates that predate the ratekeeper dataset.
	RateOverrides map[string]float64 `json:"rateOverrides"`
}

// LoadDividendConfig decodes a DividendConfig from JSON.
func LoadDividendConfig(r io.Reader) (*DividendConfig, error) {
	var dc DividendConfig
	if err := json.NewDecoder(r).Decode(&dc); err != nil {
		return nil, fmt.Errorf("decode dividends config: %w", err)
	}
	return &dc, nil
}

// RateOn returns the SBI TT buying rate on/nearest-before d.
func (v *Valuer) RateOn(d time.Time) (float64, error) {
	if v.Rates == nil {
		return 0, fmt.Errorf("no SBI rates loaded")
	}
	return v.Rates.RateOn(d)
}

// PriceOn returns the USD close price on/nearest-before d.
func (v *Valuer) PriceOn(d time.Time) (float64, error) {
	if v.Prices == nil {
		return 0, fmt.Errorf("no price series loaded")
	}
	return v.Prices.PriceOn(d)
}

// TradingDays returns the price-series trading days within [from, to].
func (v *Valuer) TradingDays(from, to time.Time) []time.Time {
	if v.Prices == nil {
		return nil
	}
	return v.Prices.DaysIn(from, to)
}

// DividendsInYear returns the dividend events with an ex-date in year.
func (v *Valuer) DividendsInYear(year int) []DividendEvent {
	var out []DividendEvent
	for _, d := range v.Dividends {
		if d.ExDate.Year() == year {
			out = append(out, d)
		}
	}
	return out
}
