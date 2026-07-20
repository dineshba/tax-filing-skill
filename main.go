// Command fa reads Fidelity Open Lots and Closed Lots CSV exports and writes a
// Markdown Schedule FA (A3) report for a target calendar year.
//
// Exchange rates (USD->INR) are always the SBI TT buying rates pulled from the
// sahilgupta/sbi-fx-ratekeeper repository. Daily prices and dividends are
// auto-fetched from Yahoo Finance for the ticker (default MSFT) so the intra-year
// peak, year-end closing and dividend columns are populated by default; pass
// -prices / -dividends to supply them offline instead.
//
// Usage:
//
//	fa -open open.csv -closed closed.csv -year 2025 -out schedule-fa-2025.md
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/dineshba/tax-filing-skill/fa"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	openPath := flag.String("open", "", "path to the Open Lots CSV (required)")
	closedPath := flag.String("closed", "", "path to the Closed Lots CSV (required)")
	year := flag.Int("year", 0, "target calendar year for Schedule FA (required)")
	pricesPath := flag.String("prices", "", "daily price CSV for the ticker (Date,Price,...); default: fetch from Yahoo Finance")
	dividendsPath := flag.String("dividends", "", "JSON file with dividend events and optional rate overrides; default: fetch from Yahoo Finance")
	sbiPath := flag.String("sbi", "", "local SBI USD rates CSV (default: fetch from sbi-fx-ratekeeper)")
	ticker := flag.String("ticker", "MSFT", "ticker symbol used to auto-fetch prices and dividends from Yahoo Finance")
	fetch := flag.Bool("fetch", true, "auto-fetch prices/dividends from Yahoo Finance when -prices/-dividends are not given")
	outPath := flag.String("out", "", "output Markdown path (default: stdout)")
	entityName := flag.String("entity", "Microsoft Corporation", "foreign entity name")
	flag.Parse()

	if *openPath == "" || *closedPath == "" || *year == 0 {
		flag.Usage()
		return fmt.Errorf("-open, -closed and -year are required")
	}

	openLots, err := parseOpen(*openPath)
	if err != nil {
		return err
	}
	closedLots, err := parseClosed(*closedPath)
	if err != nil {
		return err
	}

	valuer, err := buildValuer(*sbiPath, *pricesPath, *dividendsPath, *ticker, *fetch)
	if err != nil {
		return err
	}

	sched, err := fa.ComputeScheduleFA(openLots, closedLots, *year, valuer)
	if err != nil {
		return err
	}

	entity := fa.DefaultEntity()
	entity.Name = *entityName

	out := os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		defer f.Close()
		out = f
	}
	if err := fa.RenderMarkdown(out, sched, entity); err != nil {
		return err
	}
	if *outPath != "" {
		fmt.Fprintf(os.Stderr, "wrote %s (%d A3 rows)\n", *outPath, len(sched.Rows))
	}
	return nil
}

func parseOpen(path string) ([]fa.OpenLot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open lots: %w", err)
	}
	defer f.Close()
	lots, err := fa.ParseOpenLots(f)
	if err != nil {
		return nil, fmt.Errorf("parse open lots: %w", err)
	}
	return lots, nil
}

func parseClosed(path string) ([]fa.ClosedLot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("closed lots: %w", err)
	}
	defer f.Close()
	lots, err := fa.ParseClosedLots(f)
	if err != nil {
		return nil, fmt.Errorf("parse closed lots: %w", err)
	}
	return lots, nil
}

// buildValuer assembles the INR Valuer: SBI rates (fetched or local), the daily
// price series and dividends (from local flags, else auto-fetched from Yahoo
// Finance for the ticker), and optional manual rate overrides.
func buildValuer(sbiPath, pricesPath, dividendsPath, ticker string, fetch bool) (*fa.Valuer, error) {
	var rates *fa.SBIRateSeries
	if sbiPath != "" {
		f, err := os.Open(sbiPath)
		if err != nil {
			return nil, fmt.Errorf("sbi rates: %w", err)
		}
		defer f.Close()
		if rates, err = fa.LoadSBIRates(f); err != nil {
			return nil, err
		}
	} else {
		fmt.Fprintln(os.Stderr, "fetching SBI USD rates from sbi-fx-ratekeeper...")
		r, err := fa.FetchSBIRates(nil)
		if err != nil {
			return nil, fmt.Errorf("%w (pass -sbi <local csv> to use an offline copy)", err)
		}
		rates = r
	}

	valuer := &fa.Valuer{Rates: rates}

	// Auto-fetch prices and/or dividends from Yahoo Finance for the ticker when a
	// local file is not supplied for either.
	if fetch && (pricesPath == "" || dividendsPath == "") {
		fmt.Fprintf(os.Stderr, "fetching %s prices and dividends from Yahoo Finance...\n", ticker)
		ps, divs, err := fa.FetchYahoo(nil, ticker, "10y")
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v (pass -prices/-dividends to supply data offline)\n", err)
		} else {
			if pricesPath == "" {
				valuer.Prices = ps
			}
			if dividendsPath == "" {
				valuer.Dividends = divs
			}
		}
	}

	if pricesPath != "" {
		f, err := os.Open(pricesPath)
		if err != nil {
			return nil, fmt.Errorf("prices: %w", err)
		}
		defer f.Close()
		ps, err := fa.LoadPriceSeries(f)
		if err != nil {
			return nil, err
		}
		valuer.Prices = ps
	}

	if dividendsPath != "" {
		f, err := os.Open(dividendsPath)
		if err != nil {
			return nil, fmt.Errorf("dividends: %w", err)
		}
		defer f.Close()
		dc, err := fa.LoadDividendConfig(f)
		if err != nil {
			return nil, err
		}
		valuer.Dividends = dc.Dividends
		for ds, rate := range dc.RateOverrides {
			d, perr := time.Parse("2006-01-02", ds)
			if perr != nil {
				return nil, fmt.Errorf("rateOverrides key %q: %w", ds, perr)
			}
			rates.SetOverride(d, rate)
		}
	}

	return valuer, nil
}
