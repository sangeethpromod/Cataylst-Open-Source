# Good first issues

Maintainers: open each of these as a GitHub issue labelled `good first issue` / `help wanted`.

1. **Add SMS test cases for your bank.** Add 3–5 fake messages (debit, credit, UPI, card) in the parser's tests and fix the prompt if one parses wrong. One issue per bank.
2. **Support more banks in the statement importers** (`ledger-backend/cmd/import*`): add a parser for another bank's PDF/text statement.
3. **Make the Gemini model/provider configurable** so any OpenAI-compatible endpoint (including local models via Ollama) can parse messages.
4. **Android SMS forwarding guide.** Document an app + settings that forwards SMS to `POST /raw-transactions`.
5. **iOS Shortcut guide.** Document the Shortcut automation with screenshots.
6. **Seed script with demo data** so new users can see a populated dashboard without real messages.
7. **Translate the UI** (Hindi, Malayalam, Tamil...): extract strings into a locale file.
8. **Dark mode** for the web app.
9. **Export transactions to CSV.**
10. **Add tests** for any module under `ledger-backend/internal/module` with low coverage.
11. **Postgres support** (larger): put the storage behind an interface. Discuss in an issue first.
