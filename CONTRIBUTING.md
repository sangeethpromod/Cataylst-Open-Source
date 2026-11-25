# Contributing

Thanks for helping. Small, focused pull requests are easiest to review.

## Set up

Follow the quick start in the [README](README.md). For development you only need MongoDB (as a replica set), a Gemini key, and the three apps running (`go run ./cmd/api`, `go run ./cmd/backend`, `npm run dev`).

## Before you open a PR

```sh
cd ledger-backend && go vet ./... && go test ./...
cd ledger-api && go vet ./... && go test ./...
cd finance-frontend && npx tsc --noEmit && npm run lint
```

CI runs the same checks. Integration tests that need MongoDB are skipped unless `MONGODB_TEST_URI` is set.

## Guidelines

- Keep changes small and match the surrounding code. Add a test for new logic.
- Money is stored as integer paise (backend) and shown with the `inr` helper (frontend).
- Never commit real bank messages, statements, keys or `.env` files. Use fake data like [`examples/sample-messages.txt`](examples/sample-messages.txt).
- Parsing changes: add the SMS (fake the numbers and names) as a test case.

## Where to start

See [docs/GOOD_FIRST_ISSUES.md](docs/GOOD_FIRST_ISSUES.md). Adding support for another bank's SMS format is the most useful contribution.

## Conduct

Be kind and constructive. Maintainers may remove comments or contributions that are abusive or off-topic.
