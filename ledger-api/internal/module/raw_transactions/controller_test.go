package raw_transactions

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ledger-api/internal/middleware"
)

// Covers auth and validation only; the success path needs a live Mongo.
func TestCreateRejects(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, NewController(nil), middleware.APIKey("k"))
	cases := []struct {
		name, key, body string
		want            int
	}{
		{"no key", "", `{"message":"x"}`, 401},
		{"wrong key", "bad", `{"message":"x"}`, 401},
		{"empty", "k", `{"message":"  "}`, 400},
		{"bad json", "k", `nope`, 400},
		{"too long", "k", `{"message":"` + strings.Repeat("a", MaxBodyBytes+1) + `"}`, 400},
	}
	for _, c := range cases {
		req := httptest.NewRequest("POST", "/raw-transactions", strings.NewReader(c.body))
		req.Header.Set("X-API-Key", c.key)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != c.want {
			t.Errorf("%s: got %d want %d", c.name, rec.Code, c.want)
		}
	}
}
