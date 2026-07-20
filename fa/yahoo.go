package fa

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// YahooChartURL is the Yahoo Finance chart endpoint (JSON, no API key). It
// returns daily OHLC prices and dividend events for a ticker.
const YahooChartURL = "https://query1.finance.yahoo.com/v8/finance/chart/%s?range=%s&interval=1d&events=div"

// yahooChart mirrors the subset of the Yahoo chart response we consume.
type yahooChart struct {
	Chart struct {
		Result []struct {
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []*float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
			Events struct {
				Dividends map[string]struct {
					Amount float64 `json:"amount"`
					Date   int64   `json:"date"`
				} `json:"dividends"`
			} `json:"events"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

// FetchYahoo downloads the daily close-price series and dividend events for a
// ticker from Yahoo Finance. range_ is a Yahoo range string such as "10y".
func FetchYahoo(client *http.Client, ticker, range_ string) (*PriceSeries, []DividendEvent, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if range_ == "" {
		range_ = "10y"
	}
	url := fmt.Sprintf(YahooChartURL, ticker, range_)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (tax-filing-skill)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch Yahoo prices for %s: %w", ticker, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("fetch Yahoo prices for %s: status %s", ticker, resp.Status)
	}

	var chart yahooChart
	if err := json.NewDecoder(resp.Body).Decode(&chart); err != nil {
		return nil, nil, fmt.Errorf("decode Yahoo prices for %s: %w", ticker, err)
	}
	if len(chart.Chart.Result) == 0 {
		return nil, nil, fmt.Errorf("Yahoo returned no series for %s", ticker)
	}
	res := chart.Chart.Result[0]
	if len(res.Indicators.Quote) == 0 {
		return nil, nil, fmt.Errorf("Yahoo returned no quotes for %s", ticker)
	}
	closes := res.Indicators.Quote[0].Close

	byDay := map[time.Time]float64{}
	for i, ts := range res.Timestamp {
		if i >= len(closes) || closes[i] == nil {
			continue
		}
		day := dayKey(time.Unix(ts, 0).UTC())
		byDay[day] = *closes[i]
	}
	if len(byDay) == 0 {
		return nil, nil, fmt.Errorf("Yahoo returned no usable closes for %s", ticker)
	}
	points := make([]pricePoint, 0, len(byDay))
	for d, p := range byDay {
		points = append(points, pricePoint{day: d, price: p})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].day.Before(points[j].day) })
	series := &PriceSeries{points: points}

	var dividends []DividendEvent
	for _, dv := range res.Events.Dividends {
		if dv.Amount <= 0 {
			continue
		}
		dividends = append(dividends, DividendEvent{
			ExDate:      dayKey(time.Unix(dv.Date, 0).UTC()),
			USDPerShare: dv.Amount,
		})
	}
	sort.Slice(dividends, func(i, j int) bool { return dividends[i].ExDate.Before(dividends[j].ExDate) })

	return series, dividends, nil
}
