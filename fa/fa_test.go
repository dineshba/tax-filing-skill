package fa

import (
	"strings"
	"testing"
	"time"
)

const smallOpen = `Date acquired,Quantity,Cost basis,Cost basis/share,Value,Gain/loss,Sale availability date,Transfer availability date,Grant date,Share source,Holding period
May-15-2023,4.0000,1235.88,308.97,1491.88,256.00,-,-,-,DO,Long
Jun-15-2024,10.0000,4000.00,400.00,4500.00,500.00,-,-,Apr-01-2024,SP,Long
,
The values are displayed in USD`

const smallClosed = `Date acquired,Quantity,<span style="color: rgb(0,0,51);">Date sold or transferred</span>,Proceeds,Cost basis,Gain/loss,Term
MAY/15/2023,10.0000,JAN/28/2025,4458.29,3089.70,1368.59,LONG
MAY/15/2023,4.0000,DEC/11/2024,1799.94,1235.88,564.06,LONG
,
The values are displayed in USD`

// smallSBI is an SBI USD reference-rates fixture (DATE, PDF FILE, TT BUY, ...).
const smallSBI = `DATE,PDF FILE,TT BUY,TT SELL,BILL BUY,BILL SELL
2020-01-04 09:00,url,0.00,0.00,71.29,72.34
2023-05-15 09:00,url,82.00,83.00,81.90,83.10
2024-06-14 09:00,url,83.00,84.00,82.90,84.10
2025-01-02 09:00,url,86.00,87.00,85.90,87.10
2025-06-15 09:00,url,85.00,86.00,84.90,86.10
2025-12-31 09:00,url,89.00,90.00,88.90,90.10`

// smallPrices is a daily close-price fixture (Date, Price, Open, High, Low).
const smallPrices = `Date,Price,Open,High,Low
2025-01-15,450.0000,449,451,448
2025-06-16,500.0000,499,501,498
2025-12-31,470.0000,469,471,468`

func mustParseOpen(t *testing.T, s string) []OpenLot {
	t.Helper()
	lots, err := ParseOpenLots(strings.NewReader(s))
	if err != nil {
		t.Fatalf("ParseOpenLots: %v", err)
	}
	return lots
}

func mustParseClosed(t *testing.T, s string) []ClosedLot {
	t.Helper()
	lots, err := ParseClosedLots(strings.NewReader(s))
	if err != nil {
		t.Fatalf("ParseClosedLots: %v", err)
	}
	return lots
}

func mustSBI(t *testing.T, s string) *SBIRateSeries {
	t.Helper()
	r, err := LoadSBIRates(strings.NewReader(s))
	if err != nil {
		t.Fatalf("LoadSBIRates: %v", err)
	}
	return r
}

func mustPrices(t *testing.T, s string) *PriceSeries {
	t.Helper()
	p, err := LoadPriceSeries(strings.NewReader(s))
	if err != nil {
		t.Fatalf("LoadPriceSeries: %v", err)
	}
	return p
}

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func approx(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.01
}

func findRow(t *testing.T, s ScheduleFA, y int, m time.Month, d int) FALotRow {
	t.Helper()
	want := date(y, m, d)
	for _, r := range s.Rows {
		if r.AcquisitionDate.Equal(want) {
			return r
		}
	}
	t.Fatalf("row for %v not found", want)
	return FALotRow{}
}

// --- parsing ---------------------------------------------------------------

func TestParseOpenLots(t *testing.T) {
	lots := mustParseOpen(t, smallOpen)
	if len(lots) != 2 {
		t.Fatalf("expected 2 open lots, got %d", len(lots))
	}
	l := lots[0]
	if !l.DateAcquired.Equal(date(2023, time.May, 15)) {
		t.Errorf("date acquired = %v", l.DateAcquired)
	}
	if l.Quantity != 4.0 || l.CostBasis != 1235.88 || l.CostBasisPerShare != 308.97 {
		t.Errorf("numbers wrong: %+v", l)
	}
	if l.GrantDate != nil {
		t.Errorf("expected nil grant date, got %v", l.GrantDate)
	}
	if lots[1].GrantDate == nil || !lots[1].GrantDate.Equal(date(2024, time.April, 1)) {
		t.Errorf("grant date parse failed: %+v", lots[1].GrantDate)
	}
	if lots[1].ShareSource != "SP" {
		t.Errorf("share source = %q", lots[1].ShareSource)
	}
}

func TestParseClosedLots_HTMLHeaderAndUpperMonths(t *testing.T) {
	lots := mustParseClosed(t, smallClosed)
	if len(lots) != 2 {
		t.Fatalf("expected 2 closed lots, got %d", len(lots))
	}
	if !lots[0].DateAcquired.Equal(date(2023, time.May, 15)) {
		t.Errorf("date acquired = %v", lots[0].DateAcquired)
	}
	if !lots[0].DateSold.Equal(date(2025, time.January, 28)) {
		t.Errorf("date sold = %v", lots[0].DateSold)
	}
	if lots[0].Proceeds != 4458.29 || lots[0].Term != "LONG" {
		t.Errorf("closed lot wrong: %+v", lots[0])
	}
}

func TestParseOpenLots_BOMAndCapitalizedHeader(t *testing.T) {
	// UTF-8 BOM prefix and a differently-cased "Date Acquired" header, as some
	// Fidelity exports produce.
	in := "\ufeffDate Acquired,Quantity,Cost basis,Cost basis/share,Value,Gain/loss,Sale availability date,Transfer availability date,Grant date,Share source,Holding period\n" +
		"May-15-2023,4.0000,1235.88,308.97,1491.88,256.00,-,-,-,DO,Long\n"
	lots := mustParseOpen(t, in)
	if len(lots) != 1 {
		t.Fatalf("expected 1 lot, got %d", len(lots))
	}
	if !lots[0].DateAcquired.Equal(date(2023, time.May, 15)) {
		t.Errorf("date acquired = %v", lots[0].DateAcquired)
	}
}

func TestParseClosedLots_BOM(t *testing.T) {
	in := "\ufeffDate acquired,Quantity,Date sold or transferred,Proceeds,Cost basis,Gain/loss,Term\n" +
		"MAY/15/2023,10.0000,JAN/28/2025,4458.29,3089.70,1368.59,LONG\n"
	lots := mustParseClosed(t, in)
	if len(lots) != 1 {
		t.Fatalf("expected 1 closed lot, got %d", len(lots))
	}
	if !lots[0].DateSold.Equal(date(2025, time.January, 28)) {
		t.Errorf("date sold = %v", lots[0].DateSold)
	}
}

func TestNormaliseMonthCase(t *testing.T) {
	cases := map[string]string{
		"MAY/15/2023": "May/15/2023",
		"Jun-15-2026": "Jun-15-2026",
		"dec/01/2025": "Dec/01/2025",
	}
	for in, want := range cases {
		if got := normaliseMonthCase(in); got != want {
			t.Errorf("normaliseMonthCase(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseFloatDashAndBlank(t *testing.T) {
	for _, in := range []string{"-", "", "  "} {
		v, err := parseFloat(in)
		if err != nil || v != 0 {
			t.Errorf("parseFloat(%q) = %v, %v", in, v, err)
		}
	}
	v, err := parseFloat("1,234.50")
	if err != nil || v != 1234.50 {
		t.Errorf("parseFloat comma = %v, %v", v, err)
	}
}

// --- SBI rates -------------------------------------------------------------

func TestLoadSBIRates_SkipsZeroAndNearestPrior(t *testing.T) {
	s := mustSBI(t, smallSBI)
	// Exact date.
	if r, _ := s.RateOn(date(2025, time.January, 2)); !approx(r, 86.0) {
		t.Errorf("RateOn 2025-01-02 = %v, want 86", r)
	}
	// Nearest prior (2025-03-10 -> 2025-01-02).
	if r, _ := s.RateOn(date(2025, time.March, 10)); !approx(r, 86.0) {
		t.Errorf("RateOn 2025-03-10 = %v, want 86 (nearest prior)", r)
	}
	// The 0.00 TT BUY row for 2020-01-04 must be skipped -> no rate before it.
	if _, err := s.RateOn(date(2020, time.January, 4)); err == nil {
		t.Error("expected error before first usable rate")
	}
}

func TestSBIOverrideAndConvert(t *testing.T) {
	s := mustSBI(t, smallSBI)
	s.SetOverride(date(2019, time.August, 19), 71.765)
	if r, _ := s.RateOn(date(2019, time.August, 19)); !approx(r, 71.765) {
		t.Errorf("override rate = %v", r)
	}
	inr, err := s.USDToINR(100, date(2025, time.January, 2))
	if err != nil || !approx(inr, 8600.0) {
		t.Errorf("USDToINR = %v, %v", inr, err)
	}
}

// --- prices ----------------------------------------------------------------

func TestLoadPriceSeries(t *testing.T) {
	p := mustPrices(t, smallPrices)
	if pr, _ := p.PriceOn(date(2025, time.June, 16)); !approx(pr, 500.0) {
		t.Errorf("PriceOn 2025-06-16 = %v", pr)
	}
	// Nearest prior.
	if pr, _ := p.PriceOn(date(2025, time.July, 1)); !approx(pr, 500.0) {
		t.Errorf("PriceOn 2025-07-01 = %v (nearest prior)", pr)
	}
	days := p.DaysIn(date(2025, time.January, 1), date(2025, time.December, 31))
	if len(days) != 3 {
		t.Errorf("expected 3 trading days in 2025, got %d", len(days))
	}
}

// --- schedule (USD) --------------------------------------------------------

func TestComputeScheduleFA_2025_USD(t *testing.T) {
	s, err := ComputeScheduleFA(mustParseOpen(t, smallOpen), mustParseClosed(t, smallClosed), 2025, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.HasINR {
		t.Error("HasINR should be false with nil valuer")
	}
	may := findRow(t, s, 2023, time.May, 15)
	if !approx(may.InitialShares, 18.0) {
		t.Errorf("initial shares = %v, want 18", may.InitialShares)
	}
	if !approx(may.InitialValueUSD, 1235.88+3089.70+1235.88) {
		t.Errorf("initial USD = %v", may.InitialValueUSD)
	}
	if !approx(may.SoldInYearShares, 10.0) || !approx(may.GrossProceedsUSD, 4458.29) {
		t.Errorf("in-year sales wrong: %+v", may)
	}
	if !approx(may.ClosingShares, 4.0) {
		t.Errorf("closing shares = %v, want 4", may.ClosingShares)
	}
	jun := findRow(t, s, 2024, time.June, 15)
	if !approx(jun.ClosingShares, 10.0) || jun.GrossProceedsUSD != 0 {
		t.Errorf("jun lot wrong: %+v", jun)
	}
}

func TestComputeScheduleFA_2024_Reconstruction(t *testing.T) {
	s, err := ComputeScheduleFA(mustParseOpen(t, smallOpen), mustParseClosed(t, smallClosed), 2024, nil)
	if err != nil {
		t.Fatal(err)
	}
	may := findRow(t, s, 2023, time.May, 15)
	if !approx(may.ClosingShares, 14.0) { // 4 held + 10 sold in 2025 (after year-end)
		t.Errorf("2024 closing shares = %v, want 14", may.ClosingShares)
	}
	if !approx(may.SoldInYearShares, 4.0) || !approx(may.GrossProceedsUSD, 1799.94) {
		t.Errorf("2024 in-year sale wrong: %+v", may)
	}
}

func TestComputeScheduleFA_ExcludesFutureAcquisitions(t *testing.T) {
	s, err := ComputeScheduleFA(mustParseOpen(t, smallOpen), mustParseClosed(t, smallClosed), 2023, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range s.Rows {
		if r.AcquisitionDate.Year() == 2024 {
			t.Errorf("2024 acquisition should not be in 2023 schedule")
		}
	}
}

// --- schedule (INR via SBI + prices) ---------------------------------------

func newValuer(t *testing.T, dividends []DividendEvent) *Valuer {
	return &Valuer{Rates: mustSBI(t, smallSBI), Prices: mustPrices(t, smallPrices), Dividends: dividends}
}

func TestComputeScheduleFA_INR(t *testing.T) {
	s, err := ComputeScheduleFA(mustParseOpen(t, smallOpen), mustParseClosed(t, smallClosed), 2025, newValuer(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	if !s.HasINR {
		t.Fatal("HasINR should be true")
	}
	may := findRow(t, s, 2023, time.May, 15)

	// Initial = cost basis 5561.46 x rate on 2023-05-15 (82).
	if !approx(may.InitialValueINR, 5561.46*82.0) {
		t.Errorf("initial INR = %v, want %v", may.InitialValueINR, 5561.46*82.0)
	}
	// Proceeds column = cost basis 3089.70 x rate on/before 2025-01-28 (86).
	if !approx(may.GrossProceedsINR, 3089.70*86.0) {
		t.Errorf("proceeds INR = %v", may.GrossProceedsINR)
	}
	// Closing = 4 x price(12-31)=470 x rate(12-31)=89.
	if !approx(may.ClosingValueINR, 4.0*470.0*89.0) {
		t.Errorf("closing INR = %v, want %v", may.ClosingValueINR, 4.0*470.0*89.0)
	}
	// Peak: on 2025-01-15 the lot still held 14 shares (10 sold 28-Jan). price 450,
	// rate on/before 15-Jan = 86 -> 14*450*86 = 541800, higher than later days.
	if !approx(may.PeakValueINR, 14.0*450.0*86.0) {
		t.Errorf("peak INR = %v, want %v", may.PeakValueINR, 14.0*450.0*86.0)
	}
	if !may.PeakDate.Equal(date(2025, time.January, 15)) {
		t.Errorf("peak date = %v, want 2025-01-15", may.PeakDate)
	}

	// Jun lot holds 10 shares all year; peak on 2025-06-16: 10*500*85 = 425000.
	jun := findRow(t, s, 2024, time.June, 15)
	if !approx(jun.PeakValueINR, 10.0*500.0*85.0) {
		t.Errorf("jun peak INR = %v, want %v", jun.PeakValueINR, 10.0*500.0*85.0)
	}
}

func TestComputeScheduleFA_Dividends(t *testing.T) {
	div := []DividendEvent{
		{ExDate: date(2025, time.June, 10), USDPerShare: 0.75},
		{ExDate: date(2023, time.March, 1), USDPerShare: 1.0}, // before both lots existed as of 2025
	}
	s, err := ComputeScheduleFA(mustParseOpen(t, smallOpen), mustParseClosed(t, smallClosed), 2025, newValuer(t, div))
	if err != nil {
		t.Fatal(err)
	}
	may := findRow(t, s, 2023, time.May, 15)
	if !approx(may.GrossDividendUSD, 4.0*0.75) { // 4 shares held on 2025-06-10
		t.Errorf("may dividend USD = %v, want 3.0", may.GrossDividendUSD)
	}
	jun := findRow(t, s, 2024, time.June, 15)
	if !approx(jun.GrossDividendUSD, 10.0*0.75) {
		t.Errorf("jun dividend USD = %v, want 7.5", jun.GrossDividendUSD)
	}
	if !approx(s.TotalDividendUSD, 10.5) {
		t.Errorf("total dividend USD = %v, want 10.5", s.TotalDividendUSD)
	}
}

func TestComputeScheduleFA_NoPricesWarns(t *testing.T) {
	v := &Valuer{Rates: mustSBI(t, smallSBI)} // no price series
	s, err := ComputeScheduleFA(mustParseOpen(t, smallOpen), mustParseClosed(t, smallClosed), 2025, v)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range s.Warnings {
		if strings.Contains(w, "price series") {
			found = true
		}
	}
	if !found {
		t.Error("expected a warning about the missing price series")
	}
	// Initial value / proceeds still computed from rates.
	may := findRow(t, s, 2023, time.May, 15)
	if may.InitialValueINR == 0 {
		t.Error("initial INR should still be computed without prices")
	}
	if may.PeakValueINR != 0 || may.ClosingValueINR != 0 {
		t.Error("peak/closing should be 0 without a price series")
	}
}

func TestDividendConfigUnmarshal(t *testing.T) {
	dc, err := LoadDividendConfig(strings.NewReader(
		`{"dividends":[{"exDate":"2025-02-19","usdPerShare":0.83}],"rateOverrides":{"2019-08-19":71.765}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(dc.Dividends) != 1 || dc.Dividends[0].USDPerShare != 0.83 {
		t.Errorf("dividends parsed wrong: %+v", dc.Dividends)
	}
	if dc.RateOverrides["2019-08-19"] != 71.765 {
		t.Errorf("rate overrides parsed wrong: %+v", dc.RateOverrides)
	}
}

// --- rendering -------------------------------------------------------------

func TestINRFormatting(t *testing.T) {
	cases := map[float64]string{
		0:        "₹0",
		999:      "₹999",
		1000:     "₹1,000",
		100000:   "₹1,00,000",
		1321811:  "₹13,21,811",
		12345678: "₹1,23,45,678",
	}
	for in, want := range cases {
		if got := inr(in); got != want {
			t.Errorf("inr(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestRenderMarkdown_USDOnly(t *testing.T) {
	s, _ := ComputeScheduleFA(mustParseOpen(t, smallOpen), mustParseClosed(t, smallClosed), 2025, nil)
	var b strings.Builder
	if err := RenderMarkdown(&b, s, DefaultEntity()); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "USD working") {
		t.Error("USD working table expected in USD-only mode")
	}
	if strings.Contains(out, "Total Gross Amount Paid/Credited") {
		t.Error("INR A3 table should be absent without a valuer")
	}
}

func TestRenderMarkdown_INR(t *testing.T) {
	s, _ := ComputeScheduleFA(mustParseOpen(t, smallOpen), mustParseClosed(t, smallClosed), 2025, newValuer(t, nil))
	var b strings.Builder
	if err := RenderMarkdown(&b, s, DefaultEntity()); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{
		"A3 — Details of Foreign Equity and Debt Interest (INR)",
		"2-UNITED STATES OF AMERICA",
		"Total Gross Amount Paid/Credited (₹)",
		"15/5/2023",
		"₹",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("INR markdown missing %q", want)
		}
	}
	if strings.Contains(out, "USD working") {
		t.Error("USD working table should be absent when a valuer is supplied")
	}
}
