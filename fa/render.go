package fa

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

// Entity holds the fixed foreign-entity details reported in every Schedule FA
// A3 row. Defaults match the Microsoft Corporation example in SKILL.md.
type Entity struct {
	CountryName string
	CountryCode string
	Name        string
	Address     string
	ZipCode     string
	Nature      string
}

// DefaultEntity returns the Microsoft Corporation entity used in SKILL.md.
func DefaultEntity() Entity {
	return Entity{
		CountryName: "United States of America",
		CountryCode: "2",
		Name:        "Microsoft Corporation",
		Address:     "One Microsoft Way, Redmond, Washington",
		ZipCode:     "98052",
		Nature:      "Listed Company",
	}
}

// countryLabel formats the "Country Name and Code" column, e.g.
// "2-UNITED STATES OF AMERICA".
func (e Entity) countryLabel() string {
	return fmt.Sprintf("%s-%s", e.CountryCode, strings.ToUpper(e.CountryName))
}

// inr formats a rupee amount rounded to the nearest whole rupee using the Indian
// grouping convention with a ₹ prefix, e.g. 1321811 -> "₹13,21,811".
func inr(v float64) string {
	n := int64(math.Round(v))
	neg := n < 0
	if neg {
		n = -n
	}
	grouped := groupIndian(strconv.FormatInt(n, 10))
	if neg {
		return "-₹" + grouped
	}
	return "₹" + grouped
}

// groupIndian applies the Indian digit grouping (last three digits, then
// groups of two) to a non-negative integer string.
func groupIndian(s string) string {
	if len(s) <= 3 {
		return s
	}
	last3 := s[len(s)-3:]
	rest := s[:len(s)-3]
	var parts []string
	for len(rest) > 2 {
		parts = append([]string{rest[len(rest)-2:]}, parts...)
		rest = rest[:len(rest)-2]
	}
	if len(rest) > 0 {
		parts = append([]string{rest}, parts...)
	}
	return strings.Join(parts, ",") + "," + last3
}

func money(v float64) string  { return fmt.Sprintf("%.2f", v) }
func shares(v float64) string { return fmt.Sprintf("%.4f", v) }

// fmtAcqDate formats an acquisition date as D/M/YYYY, e.g. "15/5/2023".
func fmtAcqDate(t time.Time) string {
	return fmt.Sprintf("%d/%d/%d", t.Day(), int(t.Month()), t.Year())
}

func fmtDate(t time.Time) string { return t.Format("2-Jan-2006") }

func fmtGrant(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return fmtDate(*t)
}

// RenderMarkdown writes the Schedule FA report for s as Markdown to w.
func RenderMarkdown(w io.Writer, s ScheduleFA, e Entity) error {
	var b strings.Builder

	fmt.Fprintf(&b, "# Schedule FA — A3 Foreign Equity Holdings (Calendar Year %d)\n\n", s.Year)
	fmt.Fprintf(&b, "_Reporting period: 1 January %d to 31 December %d (calendar year)._\n\n", s.Year, s.Year)

	if s.HasINR {
		renderA3INR(&b, s, e)
	} else {
		b.WriteString("_No SBI rate series supplied, so the INR Schedule FA A3 table ")
		b.WriteString("cannot be produced. Rates are pulled from sbi-fx-ratekeeper by ")
		b.WriteString("default, or pass `-sbi <csv>` for an offline copy. ")
		b.WriteString("The USD working below is shown for reference only._\n\n")
		renderUSDDetail(&b, s)
	}

	if len(s.Rows) == 0 {
		b.WriteString("_No foreign equity holdings for this calendar year._\n\n")
	}

	if len(s.Warnings) > 0 {
		b.WriteString("## Warnings\n\n")
		seen := map[string]bool{}
		for _, wn := range s.Warnings {
			if seen[wn] {
				continue
			}
			seen[wn] = true
			fmt.Fprintf(&b, "- %s\n", wn)
		}
		b.WriteString("\n")
	}

	b.WriteString("---\n")
	b.WriteString("_Initial value = fair market value on the acquisition date (shares × USD close × SBI TT buying rate); this equals cost basis for RSUs and uses FMV for discounted ESPP lots. ")
	b.WriteString("Peak value = max over each trading day of shares held × USD close × SBI TT rate that day. ")
	b.WriteString("Closing balance = shares held on 31 Dec × 31 Dec close × 31 Dec SBI TT rate. ")
	b.WriteString("Gross amount paid/credited = dividends on shares held on each ex-date × SBI TT rate. ")
	b.WriteString("Gross proceeds = original cost basis of shares sold in-year × SBI TT rate on the sale date. ")
	b.WriteString("Rates are the SBI USD TT buying rates from sbi-fx-ratekeeper._\n")

	_, err := io.WriteString(w, b.String())
	return err
}

// renderA3INR writes the official 12-column Schedule FA A3 table in INR.
func renderA3INR(b *strings.Builder, s ScheduleFA, e Entity) {
	b.WriteString("## A3 — Details of Foreign Equity and Debt Interest (INR)\n\n")
	b.WriteString("| S No. | Country Name and Code | Name of Entity | Address of Entity | Zip Code | Nature of Entity | Date of Acquiring the Interest | Initial Value of the Investment (₹) | Peak Value of Investment During the Period (₹) | Closing Balance (₹) | Total Gross Amount Paid/Credited (₹) | Total Gross Proceeds from Sale or Redemption (₹) |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|---|\n")
	for i, r := range s.Rows {
		fmt.Fprintf(b, "| %d | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			i+1,
			e.countryLabel(),
			e.Name,
			e.Address,
			e.ZipCode,
			e.Nature,
			fmtAcqDate(r.AcquisitionDate),
			inr(r.InitialValueINR),
			inr(r.PeakValueINR),
			inr(r.ClosingValueINR),
			inr(r.GrossDividendINR),
			inr(r.GrossProceedsINR),
		)
	}
	fmt.Fprintf(b, "| | **Total** | | | | | | **%s** | **%s** | **%s** | **%s** | **%s** |\n\n",
		inr(s.TotalInitialValueINR),
		inr(s.TotalPeakValueINR),
		inr(s.TotalClosingValueINR),
		inr(s.TotalDividendINR),
		inr(s.TotalProceedsINR),
	)
}

// renderUSDDetail writes the supplementary per-lot USD working (derived purely
// from the CSVs) for verification.
func renderUSDDetail(b *strings.Builder, s ScheduleFA) {
	b.WriteString("## A3 detail (USD working, derived from lots)\n\n")
	b.WriteString("| # | Date of acquisition | Grant date | Initial shares | Initial cost (USD) | Shares sold in year | Proceeds in year (USD) | Dividend (USD) | Closing shares (Dec-31) |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|\n")
	for i, r := range s.Rows {
		fmt.Fprintf(b, "| %d | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			i+1,
			fmtDate(r.AcquisitionDate),
			fmtGrant(r.GrantDate),
			shares(r.InitialShares),
			money(r.InitialValueUSD),
			shares(r.SoldInYearShares),
			money(r.GrossProceedsUSD),
			money(r.GrossDividendUSD),
			shares(r.ClosingShares),
		)
	}
	fmt.Fprintf(b, "| | **Total** | | | **%s** | | **%s** | **%s** | |\n\n",
		money(s.TotalInitialValueUSD),
		money(s.TotalProceedsUSD),
		money(s.TotalDividendUSD),
	)
}
