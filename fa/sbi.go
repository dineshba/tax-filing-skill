package fa

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SBIRatesURLs are the candidate locations of the SBI USD reference-rates CSV in
// the sahilgupta/sbi-fx-ratekeeper repository. The repo layout has moved before,
// so the first URL that resolves is used. Rates are always pulled from here.
var SBIRatesURLs = []string{
	"https://raw.githubusercontent.com/sahilgupta/sbi-fx-ratekeeper/main/csv_files/SBI_REFERENCE_RATES_USD.csv",
	"https://raw.githubusercontent.com/sahilgupta/sbi-fx-ratekeeper/master/csv_files/SBI_REFERENCE_RATES_USD.csv",
	"https://raw.githubusercontent.com/sahilgupta/sbi-fx-ratekeeper/main/SBI_REFERENCE_RATES_USD.csv",
}

// sbiDateLayouts are accepted formats for the DATE column (with/without time).
var sbiDateLayouts = []string{"2006-01-02 15:04", "2006-01-02 15:04:05", "2006-01-02"}

// ratePoint is a single published TT BUY (TTBR) observation.
type ratePoint struct {
	day  time.Time // date only (UTC midnight)
	ttbr float64
}

// SBIRateSeries is a date-ordered series of SBI USD TT BUY (TTBR) rates used for
// USD -> INR conversion. Per the Explanation to Rule 26 / Schedule FA guidance,
// conversion uses the SBI TT buying rate on the relevant date (or the nearest
// published rate on/before it).
type SBIRateSeries struct {
	points    []ratePoint        // sorted ascending by day, last-write-wins per day
	overrides map[string]float64 // "YYYY-MM-DD" -> rate, take precedence
}

// FetchSBICSV downloads the raw SBI USD reference-rates CSV bytes, trying each
// candidate URL in turn. The raw bytes are returned so callers can cache them.
func FetchSBICSV(client *http.Client) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	var lastErr error
	for _, url := range SBIRatesURLs {
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("%s: status %s", url, resp.Status)
			continue
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("%s: %w", url, err)
			continue
		}
		return data, nil
	}
	return nil, fmt.Errorf("fetch SBI rates: all sources failed: %w", lastErr)
}

// FetchSBIRates downloads and parses the SBI USD reference rates, trying each
// candidate URL in turn.
func FetchSBIRates(client *http.Client) (*SBIRateSeries, error) {
	data, err := FetchSBICSV(client)
	if err != nil {
		return nil, err
	}
	return LoadSBIRates(bytes.NewReader(data))
}

// LoadSBIRates parses the SBI USD reference-rates CSV from r using the TT BUY
// column. Rows with a non-positive TT BUY value are skipped; the last row for a
// given date wins.
func LoadSBIRates(r io.Reader) (*SBIRateSeries, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read SBI rates: %w", err)
	}

	ttBuyCol := 2 // default column index for "TT BUY"
	byDay := map[time.Time]float64{}
	for i, rec := range records {
		if len(rec) <= ttBuyCol {
			continue
		}
		if i == 0 && strings.EqualFold(strings.TrimSpace(rec[0]), "DATE") {
			if c := indexOf(rec, "TT BUY"); c >= 0 {
				ttBuyCol = c
			}
			continue
		}
		day, err := parseSBIDate(strings.TrimSpace(rec[0]))
		if err != nil {
			continue
		}
		ttbr, err := strconv.ParseFloat(strings.TrimSpace(rec[ttBuyCol]), 64)
		if err != nil || ttbr <= 0 {
			continue
		}
		byDay[day] = ttbr // last write for a day wins
	}
	if len(byDay) == 0 {
		return nil, fmt.Errorf("no usable SBI TT BUY rates found")
	}

	points := make([]ratePoint, 0, len(byDay))
	for d, v := range byDay {
		points = append(points, ratePoint{day: d, ttbr: v})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].day.Before(points[j].day) })
	return &SBIRateSeries{points: points, overrides: map[string]float64{}}, nil
}

// SetOverride records a manual TTBR for a specific date (e.g. acquisitions that
// predate the ratekeeper dataset). Overrides take precedence over the series.
func (s *SBIRateSeries) SetOverride(day time.Time, rate float64) {
	if s.overrides == nil {
		s.overrides = map[string]float64{}
	}
	s.overrides[day.Format("2006-01-02")] = rate
}

// RateOn returns the SBI TT buying rate for d: an exact match, else the nearest
// published rate on/before d. Manual overrides take precedence.
func (s *SBIRateSeries) RateOn(d time.Time) (float64, error) {
	key := d.Format("2006-01-02")
	if v, ok := s.overrides[key]; ok {
		return v, nil
	}
	day := dayKey(d)
	// rightmost point with day <= d
	idx := sort.Search(len(s.points), func(i int) bool { return s.points[i].day.After(day) })
	if idx == 0 {
		return 0, fmt.Errorf("no SBI TTBR available on or before %s", key)
	}
	return s.points[idx-1].ttbr, nil
}

// USDToINR converts a USD amount to INR using the SBI TT buying rate on (or the
// nearest published rate before) the given date.
func (s *SBIRateSeries) USDToINR(usd float64, on time.Time) (float64, error) {
	rate, err := s.RateOn(on)
	if err != nil {
		return 0, err
	}
	return usd * rate, nil
}

func parseSBIDate(s string) (time.Time, error) {
	raw := strings.TrimSpace(s)
	for _, layout := range sbiDateLayouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return dayKey(t), nil
		}
	}
	// last resort: first whitespace-delimited token as a date
	if tok := strings.Fields(raw); len(tok) > 0 {
		if t, err := time.Parse("2006-01-02", tok[0]); err == nil {
			return dayKey(t), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable date %q", raw)
}

func indexOf(rec []string, want string) int {
	for i, c := range rec {
		if strings.EqualFold(strings.TrimSpace(c), want) {
			return i
		}
	}
	return -1
}
