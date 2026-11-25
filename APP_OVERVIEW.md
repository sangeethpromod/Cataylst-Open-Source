# Catalyst (Finanx): App Overview

A personal finance app for one user. Bank SMS messages are forwarded to an API, an LLM turns them into structured transactions, and a web app shows spending, net worth, cards, loans, investments and an AI advisor. All money is stored in **paise** (integers) and all day logic runs in **IST**.

## 1. Architecture

```
iOS Shortcut ──SMS──▶ ledger-api (Go, AWS Lambda) ──▶ MongoDB (raw_transactions)
                                                         │ change stream
                                                         ▼
Browser ◀── finance-frontend (Next.js) ◀── ledger-backend (Go, local/Docker) ──▶ LLMs (Gemini)
                                                         │                  ──▶ Yahoo Finance, mfapi.in
                                                         ▼
                                                      MongoDB
```

| Part | Folder | Role |
|---|---|---|
| **Ingest API** | `ledger-api/` | Tiny Go service on AWS Lambda (`provided.al2023`, arm64, Function URL). `POST /raw-transactions` with an `X-API-Key` header stores the raw SMS text. Deploy steps and the GitHub Actions workflow are in `ledger-api/DEPLOY.md`. |
| **Backend** | `ledger-backend/` | Go 1.x service (`cmd/backend`). Parses messages, owns all business logic, serves the REST API under `/finance/api`. Has a Dockerfile. |
| **Frontend** | `finance-frontend/` | Next.js 15, React 19, Tailwind 4, TanStack Query, Phosphor icons, Sonner toasts. Installable PWA. |
| **Import tool** | `ledger-backend/cmd/importstmt` | Command-line statement import. |
| **Sample SMS** | `messages.txt` | Example Federal, HDFC, Union and Kotak debit and credit messages, used to test parsing. |
| **Ideas** | `FEATURE_IDEAS.md` | Roadmap and ideas not yet built. |

### Security model
- The browser never sees the backend secret. Next.js route handlers (`src/app/api/[...path]`, `login`, `logout`) add `X-API-Key` and proxy to the backend.
- Login is username + PIN (the `USERS` list: name, pin, api_secret); the api_secret picks that person's backend database. A cookie signed with `SESSION_SECRET` holds the session.
- Every backend route is wrapped in the `APIKey` middleware.

### Environment variables
- **Backend:** `PEOPLE` (comma-separated names, default `ME`), then per person (`ME_`, ...): `_SECRET`, `_MONGODB_URI`, `_MONGODB_DB`, `_HOLDER`, optional `_PARSE_HINT`; then `PORT`, `GEMINI_API_KEY`, `GEMINI_MODEL`, optional `YAHOO_CHART_URL`.
- **Frontend:** `API_BASE_URL` (for example `http://localhost:8081/finance/api`), `USERS`, `SESSION_SECRET`.
- **Lambda:** per person: `_SECRET`, `_MONGODB_URI`, `_MONGODB_DB`.
- **People:** each has their own cluster. A person is on when their `<NAME>_SECRET` is set; `<NAME>` comes from `PEOPLE`. The same secret is the ingest `X-API-Key` and the backend API key. `_HOLDER` is the name on their bank accounts; `_PARSE_HINT` is optional extra SMS rules. Each raw message is stored with `user` = the lowercase name. The `api_secret` in the frontend's `USERS` must equal that person's `<NAME>_SECRET`.

### Run locally
```
cd ledger-backend && go run ./cmd/backend      # needs .env
cd finance-frontend && npm install && npm run dev
```

## 2. How a transaction is born

1. A bank SMS arrives. The iOS Shortcut posts it to `ledger-api`, which stores it as a raw message.
2. The backend watches Mongo with a **change stream**. It also catches up on anything missed on every (re)connect. Failed messages are retried at startup. Parsing is sequential, so only one LLM call runs at a time.
3. The LLM (prompt version `v4`) returns JSON: whether it is a transaction, its type, direction, amount, time, bank, account last 4, ref, raw vendor, UPI ID and a suggested category.
4. The result is validated. The amount is capped at ₹10 crore as a bad-extraction guard. The type must be one of `expense`, `income`, `transfer`, `card_bill_payment`, `refund`, `atm`, `emi`.
5. The transaction is linked to an **account** or **card** by bank and last 4 digits, and to a **vendor** (aliases, UPI ID). The category comes from the vendor, or from the LLM's suggestion.
6. New vendors, uncategorised items and doubtful parses are flagged `needs_review`.
7. A message can be re-parsed from the UI (`POST /raw-transactions/{id}/parse`).

### What counts as "spending"
`internal/spend` is the single definition shared by budgets, insights and the dashboard:
- Only `expense`, `atm` and `emi` debits count. Transfers, refunds and card-bill payments never do.
- The part of a payment that a friend owes back is subtracted.
- **Split** payments fan out into one part per category.

## 3. Features by area

### Dashboard (`/`)
Four numbered sections:
1. **Financial position:** net worth, liquid cash and obligations.
2. **Net-worth trajectory:** a chart built from daily snapshots, plus the **Catalyst AI** advice card ("Get advice").
3. **Cash flow:** income, spending and savings, with a **Sankey** diagram. A period picker is supported.
4. **Upcoming commitments:** EMIs, card dues, mandates and SIPs.

Also: a daily-spending view and a spending heatmap.

### Copilot (`/copilot`, plus the dashboard's "Now" card and the header bell)
What to do now, from the deterministic engine in `internal/module/copilot` (no ML; the AI only explains it through its tools).
- **Modes and safe to spend:** one universal mode sets how long cash must last with no salary: Ultra poor 12 months, Saving 6, Normal 3, Luxury 1. Safe to spend = (cash − everything due over those months, fixed costs included and SIPs left out − buffer) ÷ those days. When that goal is out of reach, the day gets the mode's minimum (30%, 50%, 80% or 100% of a usual day) and the page says how far short you are. Today's under- or overspend carries into tomorrow.
- **Can I buy this?:** buy, wait or skip, with the runway cost, the reasons, and a now / 2 weeks / 1 month / after salary comparison.
- **Nudges:** rules run every 15 minutes and after each parsed message, 8 am to 10 pm IST, at most 2 a day ranked by money at stake (the Monday brief is extra). Dismiss, snooze or mute; kinds that keep being dismissed switch off on their own.
- **Also:** cooling-off list with a one-week check-in, week ahead, habits, card advice, subscription costs in runway days.
- **Delivery:** in-app inbox, and PWA push for urgent kinds and the Monday brief.
- **Database load:** one shared in-memory State (60 s, cleared when a transaction lands). Daily history is read from snapshots by primary key; the recurring scan runs at most every 6 hours; nudges expire after 180 days.

### Transactions (`/transactions`)
- Paged table with search, plus filters for account, category, type, direction, date range, amount range, status (review, uncategorised, attention) and "owed only".
- **Saved views:** saved searches that sync across devices.
- Inline edit of category and vendor.
- **Edit transaction** (pencil on a row): correct the amount, day and time, type, account or card, payee, reference and note, for SMS and manual transactions alike (`PATCH /transactions/{id}` with `edit`). Only a changed day, account or payee is sent, so an SMS keeps the details it came with. It is refused where it would break something that depends on it: the amount of a split payment, the type of a payment linked to a card, or an amount below what is owed back. Re-parsing never overwrites an edit.
- **Split dialog:** divide one payment across categories.
- **Owed dialog:** record what a friend owes back and repayments. This feeds **Owed to Me** (`/owed`).
- Link a payment to the credit card it settles. The type becomes `card_bill_payment`, so it is not counted as spending.
- Re-parse a raw SMS.

### Accounts (`/accounts`, `/accounts/[id]`)
- Bank accounts with balances. The balance comes from the SMS "Bal" value or from computed movement.
- Detail page: 30-day balance line, top payees, spending by category, linked SIPs and account details.
- Create and edit accounts.

### Credit cards (`/cards`)
- Cards with limit, billing cycle and due date.
- Statement building, unpaid-statement tracking and a payments list per card.
- Card tiles with utilisation.

### EMIs and loans (`/emis`)
- Loans, fixed-instalment purchases and card balances.
- Record payments and undo the last one. Set the outstanding balance by hand.
- **Amortisation schedule** and **rate history**. Interest-rate changes apply from a given date.
- **Prepayment impact:** interest saved and tenure cut.
- **Payoff planner** (avalanche or snowball).

### Mandates (`/mandates`)
- Everything that must be paid in the next 12 months: insurance, annual fees, tax, school fees and so on.
- Frequencies: monthly, quarterly, yearly or one-off. The next due date is computed.
- "Set aside ₹X / month" for non-monthly items.
- **Suggestions:** the app detects payments that look recurring and offers to add them as mandates.

### Budgets (`/budgets`)
- A monthly limit per category, with progress.
- Uses the shared spending definition, so owed and split payments are counted fairly.

### Categories and vendors
- **Categories:** create, rename and search. Includes the special "Credit card payments" category.
- **Vendors:** aliases, a merge dialog that combines duplicates (for example "SWIGGY" and "Swiggy Instamart"), delete, and search. Changing a vendor re-syncs its transactions (`OnChange` → `Resync`).

### Insights (`/insights`)
Patterns from the last year of spending:
- **Recurring charges** and subscriptions (active or lapsed), with a monthly total.
- **Anomalies:** unusual amounts or unknown vendors.
- **When you spend:** weekday and hour patterns.

### Investments
- **Mutual funds (`/funds`):** holdings with live NAV from mfapi.in, fund-code lookup, refresh, SIP recording, exit and re-enter. Allocation donuts by fund type and platform. A dedicated "Genius SIP" flow. Funds can be grouped.
- **Stocks (`/stocks`):** holdings (symbol, quantity, buy price, date) valued with Yahoo Finance quotes. A rate-limit-shielding cache collection is used. Includes a **stock heatmap** and a TradingView panel.
- **Portfolio (`/portfolio`):**
  - Asset allocation against a target mix, with a rebalance dialog.
  - **XIRR** for funds and the whole portfolio.
  - **SIP health.**
  - A **dividend and interest income** log.
- **Price alerts:** fire once when a stock price or a fund NAV crosses a level. A background job checks every 5 minutes.

### Net-worth snapshots
- One row per IST day: every account balance, stocks, funds, other holdings, net worth, amounts owed, and that day's spending and income.
- A background job refreshes today's row every 15 minutes. `POST /snapshots/backfill` rebuilds the past from transactions.
- Charts read these rows, not live replays.

### AI (`/ai` and the dashboard)
- **Catalyst AI chat:** powered by Gemini, with Markdown replies and saved conversations (list, open, delete).
- **Tool calling.** The model calls small read-only lookups instead of reading a vector store:
  - `spend_by_category(month)`
  - `transactions(vendor, from, to, limit)`
  - `compare_months(a, b)`
  - `remember(fact)`
  - `safe_to_spend`, `purchase_check(amount, date)`, `delay_scenarios(amount)`, `upcoming_money(days)`: the Copilot engine's answers, which the model explains but never recalculates
- **Advice** on the dashboard from the same data.

### Profile (`/profile`, "About me")
- A single document: salary, SIP and commitment totals, cash buffer, hand-entered holdings and goals.
- The AI reads it and the `remember` tool adds to it. "Forget memory" clears what the AI saved.

### Maintenance
- `POST /maintenance/purge`: deletes everything before a cut-off day (transactions, raw messages, snapshots). Account balances are kept exactly, because the deleted movement is first folded into each opening balance.
- `legacy` package: one-off data fixes run at startup.

## 4. Frontend map

| Route | Page |
|---|---|
| `/` | Dashboard |
| `/transactions` | Transactions, filters, saved views |
| `/accounts`, `/accounts/[id]` | Accounts and detail |
| `/cards` | Credit cards |
| `/emis` | Loans and EMIs |
| `/mandates` | Yearly obligations |
| `/budgets` | Category budgets |
| `/categories`, `/vendors` | Categories, vendors and merge |
| `/insights` | Recurring, anomalies, patterns |
| `/funds`, `/stocks`, `/portfolio` | Investments |
| `/owed` | Money friends owe back |
| `/ai` | AI chat |
| `/profile` | About me |
| `/login` | Username + PIN sign-in |

**Component layers** (atomic design): `atoms` (buttons, inputs, logos), `molecules` (StatCard, Modal, Tabs, ListRow, PageHeader, Skeleton), `organisms` (tables, forms, dialogs, charts, sidebar, mobile nav) and `templates` (AppShell, PageLayout, CenteredLayout). There are no browser dialogs or date inputs: `Calendar` and `DatePicker` (molecules; keyboard and mobile friendly, optional time, `min` and `max`) replace the native date fields, and `useDialogs()` (organisms) gives `confirm` and `prompt` in the app's own popup, with a red "danger" confirm for deletes. The layout is responsive: a sidebar on desktop and a bottom tab bar with a "More" sheet on mobile.

## 5. REST API (all under `/finance/api`, all need `X-API-Key`)

| Area | Endpoints |
|---|---|
| Dashboard | `GET /dashboard`, `/dashboard/daily`, `/dashboard/networth`, `/dashboard/advice`; `POST /dashboard/chat`; `GET /dashboard/chats`, `/dashboard/chats/{id}`; `DELETE /dashboard/chats/{id}` |
| Transactions | `GET /summary`, `GET /transactions`, `POST /transactions/search`, `PATCH /transactions/{id}`, `POST /raw-transactions/{id}/parse` |
| Accounts | `GET/POST /accounts`, `GET/PATCH /accounts/{id}` |
| Cards | `GET/POST /cards`, `PATCH /cards/{id}`, `GET /cards/{id}/payments` |
| Categories | `GET/POST /categories`, `POST /categories/search`, `PATCH /categories/{id}` |
| Vendors | `GET/POST /vendors`, `POST /vendors/search`, `PATCH/DELETE /vendors/{id}`, `POST /vendors/{id}/merge`, `POST /vendors/{id}/aliases` |
| EMIs | `GET/POST /emis`, `PUT/DELETE /emis/{id}`, `POST /emis/{id}/payments`, `DELETE /emis/{id}/payments/last`, `PUT /emis/{id}/balance`, `GET /emis/payoff`, `/emis/{id}/impact`, `/emis/{id}/schedule`, `POST /emis/{id}/rates`, `DELETE /emis/{id}/rates/{from}` |
| Mandates | `GET/POST /mandates`, `GET /mandates/suggestions`, `PUT/DELETE /mandates/{id}` |
| Budgets | `GET /budgets`, `PUT/DELETE /budgets/{categoryId}` |
| Insights | `GET /insights`, `GET /insights/patterns` |
| Funds | `GET/POST /funds`, `GET /funds/lookup/{code}`, `POST /funds/refresh`, `PUT/DELETE /funds/{id}`, `POST /funds/{id}/sip`, `/exit`, `/reenter`, and `PUT/DELETE/POST /funds/genius/sip` |
| Stocks | `GET/POST /stocks`, `PUT/DELETE /stocks/{id}` |
| Portfolio | `GET /portfolio`, `PUT/DELETE /portfolio/target`, `GET/POST /portfolio/income`, `DELETE /portfolio/income/{id}` |
| Alerts | `GET/POST /alerts`, `DELETE /alerts/{id}` |
| Copilot | `GET /copilot`, `PUT /copilot/mode`, `POST /copilot/check`, `GET /copilot/nudges`, `POST /copilot/nudges/{id}`, `PUT /copilot/mutes/{kind}`, `GET/POST /copilot/wants`, `PATCH/DELETE /copilot/wants/{id}` |
| Views | `GET/PUT /views`, `DELETE /views/{id}` |
| Snapshots | `GET /snapshots`, `POST /snapshots/backfill` |
| Profile | `GET/PUT /profile`, `DELETE /profile/memory` |
| Maintenance | `POST /maintenance/purge` |
| Ingest (`ledger-api`) | `POST /raw-transactions` |

## 6. Background jobs (started in `cmd/backend/main.go`)

| Job | Cadence |
|---|---|
| Parse new raw messages (change stream) | On every insert, plus catch-up on connect |
| Retry failed messages | Once at startup |
| Price alerts check | Every 5 minutes |
| Snapshot refresh | Every 15 minutes |
| Copilot rules | Every 15 minutes and after each parsing pass |

## 7. Not built yet (see `FEATURE_IDEAS.md`)
Event or project tags, self-transfer vs person classification, month-over-month comparison card, cash-flow forecast, goals, what-if simulator, tax view, notifications and weekly digest, statement or email import, multi-user household, app lock and audit log.

> Note: `FEATURE_IDEAS.md` was written earlier, and some of its ideas have since been built (budgets, mandates, owed, split, recurring and anomaly detection, XIRR, allocation, vendor merge, saved views, payoff planner, price alerts, PWA, AI tool calling). Check it before treating any item there as open.

## 8. Tests
Go unit tests sit beside the code (`*_test.go`) for parsing, filters, patches, EMI maths, XIRR, budgets, insights, portfolio, snapshots and AI chat. Some integration tests are named `*_it_test.go`. There is no frontend test suite.
