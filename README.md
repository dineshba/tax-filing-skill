AI Skill for filling details for ITR 2

## Schedule FA generator (`fa` CLI)

A Go CLI that reads Fidelity **Open Lots** and **Closed Lots** CSV exports and
produces a Markdown **Schedule FA (A3)** report for a target **calendar year**
(Jan–Dec, as Schedule FA requires — not the Apr–Mar financial year).

Exchange rates are **always** the SBI USD **TT buying** rates pulled from
[`sahilgupta/sbi-fx-ratekeeper`](https://github.com/sahilgupta/sbi-fx-ratekeeper).
Daily prices and dividends are **auto-fetched from Yahoo Finance** for the ticker
(default `MSFT`), so the peak, closing and dividend columns are populated out of
the box. The report is **INR-only** (with a per-lot dividend column).

### Build & test

```sh
go build -o fa-cli .
go test ./...
```

### Usage

```sh
# Everything fetched automatically (SBI rates + MSFT prices/dividends):
go run . -open sample-open-lots.csv -closed sample-closed-lots.csv -year 2025 \
  -out schedule-fa-2025.md

# A different holding:
go run . -open open.csv -closed closed.csv -year 2025 -ticker AAPL -out fa.md

# Fully offline: supply local copies and disable network fetch.
go run . -open sample-open-lots.csv -closed sample-closed-lots.csv -year 2025 \
  -sbi sbi_usd.csv -prices msft.csv \
  -dividends dividends.example.json -fetch=false -out schedule-fa-2025.md
```

### Flags

| Flag | Type | Default | Required | Description |
|---|---|---|---|---|
| `-open` | string | – | **yes** | Path to the Fidelity **Open Lots** CSV (current holdings snapshot). |
| `-closed` | string | – | **yes** | Path to the Fidelity **Closed Lots** CSV (sales/redemptions). |
| `-year` | int | `0` | **yes** | Target **calendar year** (Jan–Dec). Sets the reporting window `[1 Jan, 31 Dec]` that bounds which lots appear, the year-end closing holdings, in-year sales/proceeds, the peak-value scan, and in-year dividends. |
| `-ticker` | string | `MSFT` | no | Ticker symbol used to **auto-fetch** daily prices and dividends from Yahoo Finance. |
| `-prices` | string | – | no | Local daily price CSV (`Date,Price,Open,High,Low`) to use **instead of** the Yahoo fetch. Needed for the Peak and Closing columns; without any price data those columns are 0 and a warning is emitted. |
| `-dividends` | string | – | no | Local JSON (`dividends` + optional `rateOverrides`, see `dividends.example.json`) to use **instead of** the Yahoo dividend fetch. Feeds the "Total Gross Amount Paid/Credited" column. |
| `-sbi` | string | – | no | Local SBI USD rates CSV (e.g. `sbi_usd.csv`) instead of fetching from `sbi-fx-ratekeeper`. Enables fully offline runs. |
| `-fetch` | bool | `true` | no | When `true`, auto-fetch prices/dividends from Yahoo for any of `-prices`/`-dividends` not supplied. Set `-fetch=false` to disable all Yahoo network access (use with local `-prices`/`-dividends`). |
| `-out` | string | – | no | Output Markdown path. Defaults to **stdout** when empty. |
| `-entity` | string | `Microsoft Corporation` | no | Foreign entity name shown in the "Name of Entity" column. |

**Reproducibility note:** with the defaults, SBI rates and prices/dividends are
fetched live, and Yahoo's rolling `range=10y` window can return slightly
different series between runs (this is why peak/closing values can drift). For
stable, auditable numbers, pin all inputs and disable fetching:

```sh
go run . -open open.csv -closed closed.csv -year 2025 \
  -sbi sbi_usd.csv -prices msft.csv -dividends dividends.json \
  -fetch=false -out schedule-fa-2025.md
```

### Input files

- **Open Lots CSV** (`-open`) — Fidelity export of current holdings, one row per
  open lot (`Date acquired, Quantity, Cost basis, ...`).
- **Closed Lots CSV** (`-closed`) — Fidelity export of disposals, one row per
  sale (`Date acquired, Quantity, Date sold, Proceeds, Cost basis, ...`).
- **`sbi_usd.csv`** (`-sbi`) — SBI USD reference rates from `sbi-fx-ratekeeper`
  (`DATE, PDF FILE, TT BUY, TT SELL, ...`); the **TT BUY** column is the TTBR used
  for all USD→INR conversion. A copy is checked into the repo.
- **`msft.csv`** (`-prices`) — daily USD close prices for the ticker
  (`Date,Price,Open,High,Low`); required for the Peak and Closing columns.
- **`dividends.example.json`** (`-dividends`) — dividend schedule:
  `dividends[].exDate` + `dividends[].usdPerShare`, plus an optional
  `rateOverrides` map (`"YYYY-MM-DD": rate`) to pin a manual SBI rate for
  acquisition dates that predate the ratekeeper dataset.

### What it computes

The **A3 (INR)** table uses the official Schedule FA columns — Country Name and
Code, Name/Address/Zip/Nature of Entity, Date of Acquiring the Interest, Initial
Value, Peak Value, Closing Balance, Total Gross Amount Paid/Credited (dividends)
and Total Gross Proceeds — with amounts in Indian-grouped rupees (e.g. ₹13,21,811).

Rows are grouped by acquisition date (one row per lot). For the target year, all
USD→INR conversions use the SBI TT buying rate **on the relevant date** (or the
nearest published rate on/before it):

- **Initial value** = original acquisition USD cost × SBI TT rate on the
  acquisition date.
- **Peak value** = the maximum, over every trading day the lot is held, of
  `shares held × USD close × SBI TT rate` that day (the price-high and FX-high
  can fall on different days, so the true peak is computed day by day).
- **Closing balance** = shares still held on 31 Dec (reconstructed from the
  current open-lots snapshot plus closed lots sold after year-end) × 31 Dec
  close × 31 Dec SBI TT rate.
- **Total gross amount paid/credited** = dividends, attributed per lot as
  shares held on each ex-date × per-share amount × SBI TT rate on the ex-date.
- **Total gross proceeds** = original cost basis of the shares sold in-year ×
  SBI TT rate on the sale date.

The **`-year`** flag defines the reporting window and therefore changes the
report: it selects which lots still have holdings/activity, the 31 Dec year-end
holdings, the in-year sales and proceeds, the peak-scan date range, and the
in-year dividends. Only the **Initial value** column is year-independent (it uses
the acquisition date and its rate).

Dividends and daily prices are not in the CSVs, so they are fetched from Yahoo
Finance by default, or supplied via `-prices` / `-dividends`. The `rateOverrides`
map in the dividends JSON lets you pin a manual SBI rate for acquisition dates
that predate the ratekeeper dataset. Any missing rate/price is reported as a
warning in the output.
