# ITR-2 Filing Skill — Salaried + Indian Stocks/MF + US ESPP/RSU

Use when filing **ITR-2** for a profile matching:
- Salaried employee (India-based, employer with Form 16)
- Indian equity stocks and/or mutual fund investments (Groww, Zerodha, CAMS, KFintech)
- US shares acquired via **ESPP and/or RSU** through Fidelity NetBenefits
- Foreign dividends from US stocks with withholding tax (Form 1042-S)
- Foreign asset reporting (Schedule FA)

**You (AI):** verify rules, parse documents, compute capital gains, convert currencies, fill schedules, catch errors, verify JSON.
**User:** downloads files, logs into portal, enters values, pays tax, submits, e-verifies. Never ask for portal credentials.

---

## STEP 0 — Verify Current-Year Rules FIRST

Tax rules change every year. Before computing anything, fetch/search the current AY rules and confirm:

- **STCG 111A and LTCG 112A rates** and the LTCG exemption threshold
- Cut-off date if rates changed mid-year (e.g., 23-Jul-2024 changed rates)
- **Holding period thresholds**: 12 months for listed Indian equity/MF; 24 months for foreign/unlisted shares
- **Basic exemption limit** and **87A rebate** income limit per regime
- **Standard deduction** from salary (Rs.75,000 from FY 2025-26)
- **80CCD(2)** employer NPS cap (14% of Basic+DA as of FY 2025-26)
- **LTCG exemption** threshold (Rs.1.25L under Section 112A as of FY 2025-26)
- **Filing deadline** (non-audit July 31; audit October 31)
- **Form 67 deadline** — must be filed on or before ITR due date to claim FTC

Sources: incometax.gov.in, cbdt.gov.in, cleartax.in, taxguru.in

---

## STEP 0.5 — Kickoff Questions to Ask User (before/while collecting documents)

Ask these up front — each avoids a known back-and-forth or dead-end from prior filings:

1. **AIS PDF is often unreadable by AI (CID/Identity-H fonts).** Ask the user to open the AIS PDF,
   select the relevant table(s) (e.g., Part B1 TDS, Part B2 SFT dividends/interest), copy the text,
   and paste it into a `.txt` file in the working folder — or paste it directly into the chat.
   Do NOT spend time building/debugging a custom PDF text extractor; ask for pasted text instead.
2. **Ask for LAST YEAR's Indian stocks and mutual fund capital gains reports** (Groww/Zerodha CG
   report, CAMS/KFintech MF CG report) in addition to this year's — needed to check for brought-forward
   losses (Schedule CFL) and to verify holding-period/lot dates that straddle the FY boundary
   (e.g., a lot bought last FY, sold this FY).
3. **US stocks: ask for the Fidelity (or equivalent broker) transaction/sale summary PDF** covering
   the full FY (Apr-Mar) — this is the primary source for US share sale proceeds and dates.
4. **US stocks: ask for both the Closed Lots CSV and Open Lots CSV** — Closed Lots gives historical
   cost basis per lot for sales made this year; Open Lots gives Dec-31 and Mar-31 holdings needed for
   Schedule FA and Schedule AL.
5. **Ask whether the user is using (or willing to use) the Income Tax Department's Offline Utility**
   for ITR-2, rather than the portal's direct online screens or a raw JSON upload. The Offline Utility
   is generally the easier and more reliable path — see STEP 5.5 for why a hand-built JSON cannot be
   uploaded directly to the portal.
6. **Ask the user to download the pre-filled JSON from the IT portal first** (portal → e-File → Income
   Tax Return → Download Pre-Filled Data). Fill/correct/verify it together in this AI session, then
   have the user import/upload that corrected JSON into the Offline Utility (not directly to the portal)
   to generate the final submittable return.

---

## STEP 1 — Collect Documents

Ask the user to put all files in one folder and share the path. Read each file directly — do not ask the user to pre-process.

| # | Document | Where to get it | Purpose |
|---|---|---|---|
| 1 | Form 16 Part A + B | Employer HR portal | Salary, perquisites (RSU/ESPP), TDS on salary |
| 2 | AIS / TIS (PDF + JSON) | portal → Services → AIS | Master income record, TDS, dividends |
| 3 | Last year's filed ITR (JSON) | portal → e-File → View Filed Returns | Carry-forwards, regime, form used |
| 4 | Groww / Zerodha CG Report (Excel) | Broker app → Reports → P&L | Indian stock STCG/LTCG |
| 5 | CAMS / KFintech MF CG Report (Excel) | cams.com or kfintech.com | MF STCG/LTCG |
| 6 | Fidelity Stock Sale Summary | Fidelity NetBenefits → History | US share sales in the FY |
| 7 | Fidelity View Open Lots CSV | Fidelity → Positions → Open Lots | Schedule FA closing values, Schedule AL |
| 8 | Fidelity View Closed Lots CSV | Fidelity → History → Closed Lots | Historical cost basis per lot |
| 9 | Dividend statements | Fidelity → Documents → Tax Forms | Schedule OS amounts, Schedule FA |
| 10 | Form 1042-S | Fidelity → Tax Documents | US withholding tax amounts for FTC |
| 11 | Self-assessment tax challan | After paying via e-Pay Tax | Schedule IT entry |

**Also ask (one-line answers):**
- Old or new tax regime?
- Any advance tax paid during the year?
- NPS employer contribution amount?
- Any house property income?
- Fidelity account number? (for Schedule FA A2)
- Was ESPP discount taxed as perquisite in Form 16? Which FY?

**Parsing notes:**
- JSONs: read directly.
- .xlsx (Groww, CAMS): use Excel COM or openpyxl. Find header row by scanning for Date/Stock name. Watch for blank PnL cells — compute manually as SellValue minus BuyValue.
- PDF files from portal (preview ITR, last year ITR) are often image-based or DRM-protected — fall back to JSON for verification.

---

## STEP 2 — Classify Capital Gains

### Holding Period and Tax Rates (post 23-Jul-2024)

| Asset | LTCG Threshold | Section | Tax Rate | ITR Location |
|---|---|---|---|---|
| Indian equity shares (listed, STT paid) | 12 months | 112A | 12.5% | CG A4 + Schedule 112A |
| Equity-oriented MF (STT paid) | 12 months | 112A | 12.5% | CG A4 + Schedule 112A |
| Equity ETFs (Nifty BeES, IT ETF listed, STT paid) | 12 months | 112A | 12.5% | CG A4 + Schedule 112A |
| Commodity ETFs (Silver BeES, Gold BeES listed NSE, no STT) | 12 months | 112 | 12.5% | CG B8 a(ii) |
| US shares STCG (held less than 24 months) | 24 months | 48 | Slab rate | CG A5 |
| US shares LTCG (held 24 months or more) | 24 months | 112 | 12.5% | CG B8 a(i) |
| Debt MF (bought after Apr-2023) | none | Slab | Slab rate | CG A5 |
| Bonus shares (via corporate action) | 12 or 24 months | per above | per above | Cost = Rs.0 per Section 55(2)(aa) |

WARNING: STCG 111A at 20% is ONLY for Indian equity/equity MF with STT paid. US shares go to A5 at slab rate, NOT A2.
WARNING: Silver BeES is NOT a 24-month asset. It is listed on NSE so 12-month threshold applies. Goes to B8 a(ii), not 112A.
WARNING: Buyback post Oct-2024 is treated as deemed dividend in Schedule OS row 1aiii, not capital gain.

Pre 23-Jul-2024 sales: STCG 111A = 15%; LTCG 112A = 10% with Rs.1L exemption.

### Where Each Income Goes in Schedule CG

| Income Type | Section | Reasoning |
|---|---|---|
| Indian equity STCG (STT paid) | A2 at 20% | Listed equity, STT paid |
| Indian equity LTCG (STT paid, 12+ months) | A4 + Schedule 112A at 12.5% | Listed equity, STT paid |
| MSFT STCG (less than 24 months) | A5 at slab rate | Foreign share, no STT |
| MSFT LTCG (24 months or more) | B8 a(i) at 12.5% | Not listed on Indian exchange |
| Silver BeES LTCG (12 months or more) | B8 a(ii) at 12.5% | Listed on NSE, commodity ETF, no STT |

### B8 a(i) Section 50CA for MSFT
For NASDAQ-listed MSFT: FMV = market price = actual transaction price. No uplift needed. Enter actual proceeds in FullValueConsdSec50CA.

---

## STEP 3 — Currency Conversion

### Rule 115 for Capital Gains (sale consideration and cost)

INR amount = USD amount x SBI TTBR of last working day of month PRECEDING the transaction date
TTBR = RBI Reference Rate minus Rs.0.25

If month-end is weekend or holiday, use last working day before it.
Source: https://www.rbi.org.in/Scripts/ReferenceRateArchive.aspx

### Rule 128 for FTC (withholding tax conversion)

Withholding tax INR = USD withheld x TTBR of last working day of month PRECEDING withholding date

### Example Rates AY 2026-27 (verify fresh each year from RBI site)

| Month-end | TTBR (Rs.) | Used for |
|---|---|---|
| Apr-30-2023 | 81.58 | RSU vest May-2023 cost |
| May-31-2023 | 82.32 | ESPP Jun-2023 cost |
| Jul-31-2023 | 81.99 | RSU vest Aug-2023 cost |
| Apr-30-2025 | 84.31 | Sales May-2025 proceeds |
| Aug-29-2025 | 87.92 | Sales Sep-2025 proceeds (Aug-31 was Sunday) |
| Nov-30-2025 | 89.68 | Schedule FA Dec-31 closing |
| Mar-31-2026 | 93.22 | Schedule AL, FA A3 closing |

---

## STEP 4 — ESPP and RSU Cost Basis

### RSU
- Cost = FMV on vest date (Fidelity usually shows this correctly)
- Employer taxes FMV as perquisite on vest date (Form 16 Section 17(2))
- INR cost = FMV x TTBR of month-end preceding vest date

### ESPP Critical Check
- Fidelity shows discounted purchase price as cost basis
- If employer taxed the ESPP discount as perquisite in Form 16 that year, correct INR cost = FMV on purchase date, not discounted price
- Check Form 16 for FY of ESPP purchase, look for Section 17(2) "Perquisite - ESPP/ESOP"
- If perquisite tax was paid on discount: using discounted cost = double taxation. Must use FMV instead.

---

## STEP 5 — Build Pre-Entry Workpapers

Complete all of these before opening the portal.

### A. Schedule 112A CSV (AE format for post 31-Jan-2018 lots)

| Column | AE Rule |
|---|---|
| 1a | AE |
| 2 ISIN | INNOTREQUIRD |
| 3 Name | CONSOLIDATED |
| 4 Quantity | blank |
| 5 Sale price per share | blank |
| 6 Full consideration | Rounded to nearest rupee |
| 7 Cost without indexation | same as col 8 |
| 8 Actual cost | Up to 4 decimal places |
| 9 FMV 31Jan2018 | 0 |
| 10 FMV per share 31Jan2018 | blank |
| 11 Total FMV 31Jan2018 | 0 |
| 12 Transfer expenses | 0 |
| 13 Total deductions | Rounded to nearest rupee |
| 14 Balance | Rounded to nearest rupee |

No special characters in any field: no comma, slash, dash, parentheses, ampersand, quote, semicolon, colon.

BE format (acquired before 31-Jan-2018): use actual ISIN; cost = higher of actual cost or FMV on 31-Jan-2018.

### B. Schedule FA A3 CSV (one row per acquisition date)

Header row:
"Country/Region name","Country Name and Code","Name of entity","Address of entity","ZIP Code","Nature of entity","Date of acquiring the interest","Initial value of the investment","Peak value of investment during the Period","Closing balance","Total gross amount paid/credited with respect to the holding during the period","Total gross proceeds from sale or redemption of investment during the period"

Data row example for MSFT:
1,"2-UNITED STATES OF AMERICA","Microsoft Corporation","One Microsoft Way, Redmond, Washington","98052","Listed Company","DD/M/YYYY",[initial],[peak],[closing],[income],[proceeds]

- One row per RSU vest date or ESPP purchase date
- Initial value = USD cost x TTBR of month-end before acquisition
- Peak value = shares in lot x peak price x TTBR on peak date
- Closing value = shares remaining x Dec-31 price x Nov-30 TTBR (0 if fully sold in the year)

### C. Table F Quarterly Breakup

| Quarter | Dates |
|---|---|
| Q1 | Apr 1 to Jun 15 |
| Q2 | Jun 16 to Sep 15 |
| Q3 | Sep 16 to Dec 15 |
| Q4 | Dec 16 to Mar 15 |
| Q5 | Mar 16 to Mar 31 (separate quarter — often missed) |

- Sl.3 STCG applicable rate = post-set-off amounts per quarter. If losses exceed gains in a quarter, enter 0 (no negatives).
- Sl.5 LTCG 12.5% = include ALL 12.5% sources per quarter: 112A stocks + 112A MF + B8 Silver + B8 MSFT
- Validate: Sl.3 total = BFLA item 3v; Sl.5 total = BFLA item 3vii
- Adjust +-1 rupee in Q1 if there is rounding mismatch
- Table F is NOT auto-filled by portal. Must be entered manually every year.

---

## STEP 5.5 — Do NOT Hand-Build JSON for Direct Portal Upload

The e-filing portal's "Upload JSON" feature is **NOT a generic JSON importer** — it validates
`CreationInfo.SWProviderID`/`SWCreatedBy`/`JSONCreatedBy` against a registry of software vendors
(commercial tax software, CA tools, or the Department's own Offline Utility) tied to your login
session. A hand-crafted or AI-generated JSON, however schema-valid, will be rejected with
**"Invalid Software Provider ID (SWProviderID) / does not have access to upload this Return"**
even after every other schema error is fixed. This is an authorization check, not a validation
check, and cannot be worked around by picking a different placeholder code.

**Correct workflow when building/verifying values with AI assistance:**
1. AI computes and verifies all figures (capital gains classification, schedules, totals) in a
   workpaper or draft JSON — this is still useful for cross-checking arithmetic and catching
   classification errors before you touch the real form.
2. Download the Department's **Offline Utility for ITR-2** (incometax.gov.in → Downloads).
3. Manually re-enter (or use the utility's "Import Draft ITR" feature if available for your AY)
   the verified figures into the Offline Utility screens — do NOT try to upload a hand-built
   JSON directly to the portal.
4. Let the Offline Utility generate its own JSON (with a valid registered `SWProviderID`) on
   Validate → Submit. Only JSON produced by an authorized utility/software will upload successfully.
5. If iterating on a downloaded-then-re-edited JSON (e.g., portal round-trip for review), be aware
   this only works if the JSON was originally produced by an authorized utility — editing values
   inside it may or may not survive re-upload depending on portal-side JSON signature/hash checks;
   treat this as unverified/risky, not a substitute for step 3-4.

---

## STEP 5.6 — Manually Editing a Filled JSON: Keep Downstream Schedules in Sync

If you directly edit values in a filled ITR-2 JSON (e.g., fixing a capital-gains
misclassification), the portal's online form normally **auto-cascades** figures from Schedule CG
into Schedule CYLA -> Schedule BFLA -> Table F -> Part B-TI. A hand-edited JSON does **not**
auto-cascade — every downstream schedule that references the changed figure must be updated
by hand, in this order, or the portal will report a mismatch between schedules even though the
final Total Income is correct:

1. **Schedule CG** (`ScheduleCGFor23`) — the source-of-truth figures per asset/section.
2. **Table F** (`AccruOrRecOfCG`, quarterly breakup) — must re-map each sale to its correct
   quarter bucket for the NEW classification (a reclassified sale moves quarters/rows too, not
   just its section).
3. **Schedule CYLA** (`ScheduleCYLA`) — `IncOfCurYrUnderThatHead`/`IncOfCurYrAfterSetOff` per head
   must match the new Schedule CG totals for that head exactly.
4. **Schedule BFLA** (`ScheduleBFLA`) — `IncOfCurYrUndHeadFromCYLA`/`IncOfCurYrAfterSetOffBFLosses`
   must match Schedule CYLA's post-set-off figures for that head. BFLA item **3v must equal Table F
   Sl.3 total** (STCG applicable rate) and BFLA item **3vii must equal Table F Sl.5 total** (LTCG
   12.5%) — these are two independently-checked places in the schema, not one shared field.
5. **Part B-TI** (`PartB-TI`) — totals derived from BFLA.

When told "X and Y are not yet fixed" after you believe you fixed them, re-check **both** CYLA and
BFLA for that head (not just Table F) — a common mistake is updating Table F and BFLA but leaving
Schedule CYLA (the intermediate schedule) with stale pre-fix values, which the portal also validates
against.

---

## STEP 6 — Portal Entry Order

Fill schedules in this exact order. Order matters — later schedules depend on earlier ones.

1. Schedule S (Salary) — FIRST. Portal needs Basic+DA to compute 80CCD(2) cap. Without this, 80CCD(2) shows Rs.0.
2. Schedule CG — A2 Indian STCG equity, A4 112A LTCG, A5 STCG slab/foreign, B8 a(i) MSFT, B8 a(ii) Silver BeES
3. Schedule 112A — upload CSV
4. Schedule OS — dividends full FY Apr-Mar, interest, buyback deemed dividend row 1aiii
5. Table 10 — quarterly OS income breakup (avoids 234C warning)
6. Schedule FA — A2 custodial account + A3 equity holdings (CALENDAR YEAR Jan-Dec, NOT Apr-Mar)
7. Schedule FSI — foreign income by head and country
8. Schedule TR1 — FTC claim
9. Schedule VI-A — deductions 80CCD(2), 80C, 80CCD(1B), 80D
10. Schedule AL — assets and liabilities (mandatory if total income > Rs.50L); includes MSFT at Mar-31 value
11. Table F — quarterly CG breakup Sl.3 and Sl.5 (mandatory, not auto-filled)
12. Schedule IT — self-assessment tax challan

---

## STEP 7 — Schedule-by-Schedule Field Guide

### Schedule S — Salary (fill FIRST)

| Field | Source |
|---|---|
| Salary as per 17(1) | Form 16 Part B gross salary |
| Perquisites 17(2) | RSU vest FMV + ESPP discount taxed by employer |
| Profits in lieu 17(3) | Usually 0 |
| Standard deduction | Rs.75,000 (FY 2025-26 onwards) |
| Entertainment allowance | 0 for private sector |

### Schedule CG — B8 a(i) MSFT Fields

| Field | How to fill |
|---|---|
| a. Full value of consideration (actual proceeds) | USD proceeds x sale TTBR |
| b. FMV of unquoted shares 50CA | Rs.0 (NASDAQ listed = FMV = market price = actual proceeds) |
| c. Adopted value (higher of a or b) | = actual proceeds |
| bi. Cost of acquisition | USD cost x acquisition TTBR |
| biii. Transfer expenses | Rs.0 |
| e. LTCG | c minus bi |

### Schedule OS — Other Sources

| Row | What to enter |
|---|---|
| 1ai | All dividends: Indian + foreign (full FY Apr-Mar) |
| 1aiii | Buyback deemed dividend from Indian companies (post Oct-2024) |
| 3 | Savings bank interest |
| 4 | FD interest |

### Schedule FA — CRITICAL: Calendar Year Jan 1 to Dec 31, NOT financial year Apr-Mar

FA dividends = Jan-Dec only (different from OS which is full Apr-Mar FY). Two different correct numbers.

**A2 — Fidelity Custodial Account**

| Field | Value |
|---|---|
| Country | United States of America Code 2 |
| Name of institution | Fidelity Brokerage Services LLC |
| Address | 900 Salem Street, Smithfield, Rhode Island 02917, USA |
| Account number | From Fidelity statement |
| Status | Active / Beneficial owner |
| Peak value | Max(shares x price x TTBR) on any single date in Jan-Dec |
| Closing balance | Shares on Dec-31 x Dec-31 price x Nov-30 TTBR |
| Gross income | Dividends received Jan-Dec only |
| Gross proceeds | Sale proceeds Jan-Dec only |

**A3 — Foreign Equity Holdings (one row per acquisition date)**

| Field | Value |
|---|---|
| Country | United States of America Code 2 |
| Name | Microsoft Corporation |
| Address | One Microsoft Way, Redmond, Washington, 98052 |
| Nature | Listed Company |
| Date of acquisition | RSU vest or ESPP purchase date in DD/M/YYYY |
| Initial value | Lot shares x USD cost x TTBR of month-end before acquisition |
| Peak value | Lot shares x peak price in calendar year x TTBR on peak date |
| Closing value | Lot shares remaining x Dec-31 price x Nov-30 TTBR (0 if fully sold) |
| Gross income | 0 per lot (enter total only in A2) |
| Gross proceeds | Sale proceeds for that lot in Jan-Dec only |

### Schedule FSI — Foreign Source Income

| Column | Value |
|---|---|
| Head of income | Other Sources |
| Country code | 2 (USA) |
| Taxpayer ID | Fidelity account number |
| Income from foreign sources | Full FY dividend in INR |
| Tax paid outside India | US withholding in INR (Rule 128) |
| Tax payable in India | Dividend x slab rate |
| Relief claimed | Min(foreign tax, Indian tax) |

Add a second row for Capital Gains:
- Foreign CG = MSFT STCG + LTCG in INR
- Tax paid outside India = Rs.0 (US does not tax capital gains for non-residents)
- Relief = Rs.0

### Schedule TR1 — Tax Relief (DTAA)

| Field | Value |
|---|---|
| Country | USA |
| Article of DTAA | 25 (Elimination of Double Taxation) |
| Income type | Dividends Article 10 |
| DTAA rate | 25% (for individual shareholders; 15% only if company holds 10% or more voting stock) |
| Relief section | Section 90 |
| Amount of relief | Min(US withholding INR, Indian tax on dividend) |

### Form 67 — FTC (file BEFORE ITR submission)

Portal path: e-File → Income Tax Forms → File Income Tax Forms → Form 67

| Field | Value |
|---|---|
| Income from outside India | Full FY dividend in INR |
| Tax paid outside India Amount | US withholding in INR |
| Tax paid outside India Rate | 25% |
| Tax payable in India (normal provisions) | Dividend x slab rate |
| Tax payable u/s 115JB/JC | Rs.0 (only for companies with MAT; individuals always 0) |
| Credit u/s 90/90A Article No. | 25 |
| Credit u/s 90/90A Amount | Min(foreign tax, Indian tax) |
| Credit u/s 91 | Rs.0 (Section 91 is for non-DTAA countries only; USA has DTAA) |
| Total FTC | = 90/90A credit amount |

Attach Form 1042-S as supporting document. Submit Form 67 before submitting ITR.

Form 1042-S is NOT uploaded to ITR portal. Only the values from it are used. Keep the physical form for 6+ years as scrutiny evidence.

### Schedule VI-A — Deductions

| Section | Description | Max | Note |
|---|---|---|---|
| 80C | LIC, PF, PPF, ELSS, tuition fees | Rs.1,50,000 | |
| 80CCD(1B) | Voluntary NPS | Rs.50,000 | Over and above 80C |
| 80CCD(2) | Employer NPS | 14% of Basic+DA | Fill Schedule S first or shows Rs.0 |
| 80D | Health insurance premium | Rs.25,000 self + Rs.25,000 parents | |
| 80TTA | Savings bank interest | Rs.10,000 | |

### Schedule AL — Assets and Liabilities (income > Rs.50L)
- Immovable property: at cost or stamp value
- Shares and securities: Indian Demat portfolio (Mar-31) + MSFT holdings (shares x USD price x Mar-31 TTBR)
- Bank deposits, cash, insurance, vehicles, jewellery
- Liabilities: outstanding loans
- Closing date = Mar-31 (financial year end), NOT Dec-31

### Schedule SI — Special Income (auto-computed)
- Portal shows gross LTCG in income column but taxes net after Rs.1.25L exemption
- Verify: tax = (Gross 112A LTCG minus Rs.1,25,000) x 12.5%
- Do NOT enter Rs.1,25,000 manually anywhere. Portal applies exemption automatically.

### Section 87A — Rebate
- Old regime: eligible only if total income is Rs.5,00,000 or below
- New regime: eligible only if total income is Rs.12,00,000 or below
- 87A rebate does NOT apply to 112A LTCG or 111A STCG even if income is below threshold
- For high-income filers: enter Rs.0

### BFLA — Brought Forward and Set-Off (auto-computed by portal)
- BFLA 3v = Net STCG after set-off (must equal Table F Sl.3 total)
- BFLA 3vii = Net LTCG 12.5% after set-off (must equal Table F Sl.5 total)
- Portal automatically sets off STCL against STCG across sources

---

## STEP 8 — Verify JSON Before Submit

Download ITR JSON from portal and verify:

- AccruOrRecOfCG node (Table F) is NOT all zeros — if zeros, Table F was not saved
- BFLA 3v = Table F Sl.3 total
- BFLA 3vii = Table F Sl.5 total
- Schedule FA closing = Dec-31 values, not Mar-31
- Schedule OS dividends = full FY Apr-Mar
- Schedule FA A2 gross income = Jan-Dec only
- FTC = min(foreign tax INR, Indian tax on foreign income)
- Schedule AL includes MSFT foreign holdings at Mar-31 value
- TDS1 matches Form 16
- TDS2 checked against AIS/26AS
- 80CCD(2) eligible amount is non-zero

---

## STEP 9 — Submit, E-Verify, Done

1. File Form 67 (FTC) before ITR
2. Pay self-assessment tax via e-Pay Tax, Challan 280, Minor Head 300
3. Enter challan in Schedule IT (BSR code, serial number, date, amount)
4. Confirm Part B-TTI balance = Rs.0
5. Submit ITR
6. E-Verify with Aadhaar OTP — unverified return = not filed
7. Record acknowledgement number

---

## Pre-Filing Document Checklist

### Personal Info
- PAN, Aadhaar, bank account + IFSC, mobile linked to Aadhaar

### From Employer
- Form 16 Part A + B
- Confirm: ESPP discount taxed as perquisite? Note FY and amount.
- Confirm: RSU FMV on vest taxed as perquisite? Note FY and amount.
- NPS employer contribution amount

### From Indian Brokers
- Groww/Zerodha CG Report Excel for Apr 1 to Mar 31
- CAMS/KFintech MF CG Report Excel
- Verify ETF classifications: equity ETFs vs commodity ETFs
- Manually compute PnL for blank cells in Groww Excel (SellValue minus BuyValue)
- Separate: STCG gains, STCG losses, LTCG gains, LTCG losses

### From Fidelity
- Stock Sale Summary (FY Apr-Mar)
- View Open Lots CSV
- View Closed Lots CSV
- Dividend statements (all dates and USD amounts for full FY)
- Form 1042-S (withholding tax)
- Fidelity account number

### Exchange Rates (compute before portal)
- TTBR for each acquisition date (month-end preceding)
- TTBR for each sale date (month-end preceding)
- TTBR for each dividend/withholding date (Rule 128)
- TTBR for Dec-31 (FA closing) and Mar-31 (AL and FA A3)

### AIS / Tax Portal
- Download AIS and cross-check TDS, dividends, interest
- Download last year ITR JSON
- Note any advance tax payments (BSR code, serial, date, amount)

### Pre-computed Workpapers
- Schedule 112A CSV in AE format
- Schedule FA A3 CSV with one row per acquisition date
- Table F quarterly breakup Q1 to Q5 for Sl.3 and Sl.5
- FTC total in INR using Rule 128
- Self-assessment tax estimate

---

## Known Gotchas — Learned from AY 2026-27

| # | Gotcha | What went wrong | Fix |
|---|---|---|---|
| 1 | Silver BeES classified as 24-month | Put in same bucket as foreign shares | Silver BeES is listed on NSE, 12-month threshold, goes to B8 a(ii) not 112A |
| 2 | Table F all zeros | Left blank, portal validation failure | Must enter manually every year; map each sale to Q1-Q5 |
| 3 | 80CCD(2) shows Rs.0 eligible | Schedule VI-A filled before Schedule S | Fill Schedule S first, portal then auto-computes 14% cap |
| 4 | FA dividends = OS dividends | Used OS full FY total in Schedule FA A2 | FA = Jan-Dec only; OS = full Apr-Mar FY. Two different correct numbers. |
| 5 | FA closing used Mar-31 | Proposed Mar-31 for Schedule FA | FA requires Dec-31 calendar year end |
| 6 | ESPP double taxation | Used Fidelity discounted price as cost | If perquisite tax paid on discount, use FMV not discounted price |
| 7 | Table F total mismatch | Groww Excel had missing and blank rows | Always verify source data; blank PnL cells need manual computation |
| 8 | Q5 missed entirely | Computed only Q1 to Q4 | Mar 16-31 is a separate Q5 quarter |
| 9 | Table F zeros in JSON after portal entry | Portal did not save after entry | After Table F update: re-validate, re-download JSON, check AccruOrRecOfCG node |
| 10 | MSFT STCG put in A2 Section 111A | Confused with Indian equity | MSFT = foreign share, no STT, goes to A5 at slab rate |
| 11 | Rs.1.25L entered manually in Schedule SI | Tried to enter exemption manually | Portal applies Rs.1.25L automatically. Verify via tax = (gross minus 1.25L) x 12.5% |
| 12 | 87A rebate considered for CG income | Thought rebate applies to all income | 87A does NOT apply to 112A LTCG or 111A STCG |
| 13 | Form 1042-S upload attempted | Tried to upload US form to ITR portal | Not uploaded anywhere; only values used in OS and TR |
| 14 | 115JB/JC in Form 67 unclear | Not sure what to enter | Always Rs.0 for individuals; MAT/AMT is only for companies |
| 15 | PDF files unreadable | Tried to parse ITR preview and last year ITR PDF | Image-based or DRM-protected PDFs cannot be parsed; use JSON instead |
| 16 | STCL set-off manual calculation | Tried to compute set-off manually | Portal automatically applies via BFLA; confirm BFLA shows correct net STCG |
| 17 | Hand-built JSON uploaded directly to portal | Assumed schema-valid JSON = uploadable | Portal rejects with "Invalid SWProviderID" — only an authorized utility/software JSON can be uploaded (see STEP 5.5) |
| 18 | Schema errors fixed one round at a time, portal kept complaining | Treated each portal error message as complete/final | Portal validators run in stages; expect 2-4+ rounds of new errors after each fix (missing NRI stub fields, TaxAccumulatedBalRecPF sub-fields, email regex, SW code pattern, etc.) — budget for iterative rounds, not a single pass |
| 19 | NRI placeholder fields omitted for a resident filer | Assumed NRI-only fields aren't required for residents | Schema requires `NRITransacSec48Dtl`, `NRISecur115AD`, `NRISaleOfEquityShareUs112A`, `NRISaleofForeignAsset` as **present with zero values** even for resident filers — omission fails required-key validation regardless of residency status |
| 20 | Nested required sub-fields missed one level down | Added a required object but not required for the object itself | e.g., `ScheduleOS.IncOthThanOwnRaceHorse.TaxAccumulatedBalRecPF` itself needed nested `TotalIncomeBenefit`/`TotalTaxBenefit` (both default 0) — always check the schema recursively for `required` arrays inside newly-added objects, not just the top-level key |
| 21 | Debt/index MF holding NASDAQ 100 or other foreign index treated as normal equity LTCG | Assumed "Equity" category label in broker report = normal 112A/holding-period rules | Section 50AA "specified mutual fund" test is based on % invested in **domestic** equity, not equity overall. A fund investing in foreign stocks (e.g., NASDAQ 100 index fund) has ~0% domestic equity and qualifies as a "specified mutual fund" — units acquired on/after 1-Apr-2023 are ALWAYS deemed STCG at slab rate regardless of actual holding period, overriding normal LTCG rules. Broker reports (e.g., Groww) may still label these "Long Term" in their column header — verify lot-by-lot purchase dates against 1-Apr-2023, don't trust the broker's category label |
| 22 | PDF (AIS) text extraction attempted via custom CID/ToUnicode parsing | Assumed all portal PDFs use simple Tj/TJ ASCII text | AIS PDFs often use Identity-H encoded Type0 fonts requiring ToUnicode CMap decoding — writing a robust decoder in PowerShell is high-effort and error-prone (multiple attempts failed/produced garbled text). Prefer asking the user to paste the relevant AIS table text manually rather than sinking time into PDF parsing |
| 23 | Custom XLSX reader script silently truncated output at 60 rows | Assumed script printed the full sheet | A `[Math]::Min($rows.Count, 60)` cap in `_read_xlsx2.ps1`-style scripts silently truncates long sheets (e.g., LTCG mutual fund lot lists) with no error — always check row-count caps in ad-hoc reader scripts before trusting "no more rows" as ground truth |

---

## Key References

| Topic | Reference |
|---|---|
| TTBR rates | https://www.rbi.org.in/Scripts/ReferenceRateArchive.aspx |
| Income Tax portal | https://www.incometax.gov.in |
| India-USA DTAA | Article 10 = dividend rate cap (25% for individuals); Article 25 = elimination of double taxation |
| Capital gains sections | Section 111A, 112, 112A of Income Tax Act 1961 |
| ESPP taxation | Section 17(2) perquisite on discount; Section 49(2AA) cost basis |
| Currency conversion | Rule 115 for capital gains; Rule 128 for FTC |
| Employer NPS | Section 80CCD(2) — 14% of Basic+DA per Finance Act 2023 |
| Foreign asset law | Black Money (Undisclosed Foreign Income and Assets) Act 2015 |
| ITR-2 instructions | https://www.incometax.gov.in/iec/foportal/help/individual/return-applicable |
