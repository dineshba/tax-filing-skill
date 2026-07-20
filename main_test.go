package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildCLI compiles the CLI once into a temp dir and returns the binary path.
func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fa-cli")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

// sbiFixture is an offline SBI USD reference-rates CSV covering the sample data
// range (2022-2026), so the e2e tests never hit the network.
const sbiFixture = `DATE,PDF FILE,TT BUY,TT SELL,BILL BUY,BILL SELL
2022-01-03 09:00,url,74.00,75.00,73.90,75.10
2023-05-15 09:00,url,82.00,83.00,81.90,83.10
2024-06-14 09:00,url,83.50,84.50,83.40,84.60
2025-01-02 09:00,url,86.00,87.00,85.90,87.10
2025-06-16 09:00,url,85.00,86.00,84.90,86.10
2025-12-31 09:00,url,89.68,90.68,89.58,90.78`

// priceFixture is an offline daily USD close-price series for 2025.
const priceFixture = `Date,Price,Open,High,Low
2025-01-15,450.0000,449,451,448
2025-06-16,500.0000,499,501,498
2025-11-17,520.0000,519,521,518
2025-12-31,470.0000,469,471,468`

const dividendsFixture = `{"dividends":[{"exDate":"2025-06-16","usdPerShare":0.83}],"rateOverrides":{"2022-12-30":74.0}}`

func writeFixtures(t *testing.T) (sbi, prices, dividends string) {
	t.Helper()
	dir := t.TempDir()
	sbi = filepath.Join(dir, "sbi.csv")
	prices = filepath.Join(dir, "prices.csv")
	dividends = filepath.Join(dir, "dividends.json")
	for path, content := range map[string]string{
		sbi:       sbiFixture,
		prices:    priceFixture,
		dividends: dividendsFixture,
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return sbi, prices, dividends
}

func TestE2E_INRReport(t *testing.T) {
	bin := buildCLI(t)
	sbi, prices, dividends := writeFixtures(t)
	outFile := filepath.Join(t.TempDir(), "schedule-fa-2025.md")

	cmd := exec.Command(bin,
		"-open", "sample-open-lots.csv",
		"-closed", "sample-closed-lots.csv",
		"-year", "2025",
		"-sbi", sbi,
		"-prices", prices,
		"-dividends", dividends,
		"-out", outFile,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run failed: %v\n%s", err, out)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	md := string(data)
	for _, want := range []string{
		"# Schedule FA — A3 Foreign Equity Holdings (Calendar Year 2025)",
		"A3 — Details of Foreign Equity and Debt Interest (INR)",
		"2-UNITED STATES OF AMERICA",
		"Total Gross Amount Paid/Credited (₹)",
		"15/5/2023",
		"₹",
		"**Total**",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("report missing %q", want)
		}
	}
	if strings.Contains(md, "USD working") {
		t.Error("USD working table should be absent in the INR report")
	}
}

func TestE2E_NoPricesWarns(t *testing.T) {
	bin := buildCLI(t)
	sbi, _, _ := writeFixtures(t)
	outFile := filepath.Join(t.TempDir(), "schedule-fa-2025-noprices.md")

	cmd := exec.Command(bin,
		"-open", "sample-open-lots.csv",
		"-closed", "sample-closed-lots.csv",
		"-year", "2025",
		"-sbi", sbi,
		"-fetch=false",
		"-out", outFile,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("run failed: %v\n%s", err, out)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	md := string(data)
	if !strings.Contains(md, "Warnings") || !strings.Contains(md, "price series") {
		t.Error("expected a warning about the missing daily price series")
	}
	// Initial values and proceeds are still produced from the SBI rates.
	if !strings.Contains(md, "A3 — Details of Foreign Equity and Debt Interest (INR)") {
		t.Error("expected INR A3 table even without a price series")
	}
}

func TestE2E_MissingArgs(t *testing.T) {
	bin := buildCLI(t)
	cmd := exec.Command(bin, "-open", "sample-open-lots.csv")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit for missing args")
	}
	if !strings.Contains(string(out), "required") {
		t.Errorf("expected usage error, got: %s", out)
	}
}
