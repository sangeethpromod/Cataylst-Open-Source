<p align="center"><img src="ledger-backend/docs/logo.svg" alt="Catalyst" width="280"></p>

<h1 align="center">Catalyst</h1>

<p align="center">A self-hosted personal finance tracker for India. It turns your bank SMS into categorised transactions, and tracks accounts, cards, EMIs, mutual funds, stocks, budgets and net worth. Your data stays in your own database.</p>

## Features

- **SMS to transactions.** Forward bank SMS from your phone; an LLM extracts amount, vendor, bank, UPI ref and category. Duplicate messages are dropped.
- **Money overview.** Accounts, credit cards and statements, EMIs and loans (with amortisation), mutual funds and stocks, mandates, debts, budgets and daily net-worth snapshots.
- **Copilot.** Rules run every 15 minutes: safe-to-spend, bills due, card advice, price alerts, with optional push notifications.
- **Chat** over your own data.
- **One install, many people.** Each person gets their own database and API key.

## Quick start (Docker)

You need Docker and a [Gemini API key](https://aistudio.google.com/apikey).

```sh
cp .env.example .env     # set ME_SECRET (openssl rand -base64 32), ME_HOLDER (your name as on the bank account) and GEMINI_API_KEY
docker compose up --build
```

Open <http://localhost:3000> and sign in as `me` / `1234` (change the PIN in `docker-compose.yml`).

Send a test message (fake examples are in [`examples/sample-messages.txt`](examples/sample-messages.txt)):

```sh
curl -X POST http://localhost:8080/raw-transactions \
  -H "X-API-Key: $ME_SECRET" -H "Content-Type: application/json" \
  -d '{"message":"Federal Bank: Debited Rs 125.71 from a/c X1234 on 26Oct26 via UPI to shop@okbank. Ref 100000000001"}'
```

It appears under Transactions within a few seconds. To feed real SMS, point a phone shortcut (iOS Shortcuts, or an SMS-forwarding app on Android) at that URL with the `X-API-Key` header.

## Layout

| Folder | What |
|---|---|
| [`ledger-api`](ledger-api) | Go. Receives SMS and stores them raw. Runs as a plain server or on AWS Lambda ([deploy notes](ledger-api/DEPLOY.md)). |
| [`ledger-backend`](ledger-backend) | Go. Parses messages with the LLM and serves the app's API ([details](ledger-backend/README.md)). |
| [`finance-frontend`](finance-frontend) | Next.js web app ([details](finance-frontend/README.md)). |

[`APP_OVERVIEW.md`](APP_OVERVIEW.md) describes every screen and how the parts fit together.

## Running without Docker

Needs Go, Node 22 and MongoDB as a replica set (the backend uses a change stream; Atlas works). Set up `.env` and `finance-frontend/.env.local` from the `.env.example` files, then:

```sh
cd ledger-api && go run ./cmd/api
cd ledger-backend && go run ./cmd/backend
cd finance-frontend && npm install && npm run dev
```

## Contributing

Contributions are welcome, and bank SMS formats are the easiest place to start. See [CONTRIBUTING.md](CONTRIBUTING.md) and the [good first issues](docs/GOOD_FIRST_ISSUES.md).

## Privacy and security

Messages and transactions go to your database only. The only outside calls are to the LLM (message text, to parse it), market-data APIs, and optional S3 and push services. Report vulnerabilities as described in [SECURITY.md](SECURITY.md).

## Licence

[MIT](LICENSE)
