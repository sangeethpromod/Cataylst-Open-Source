package raw_transactions

import "net/http"

func RegisterRoutes(mux *http.ServeMux, c *Controller, auth func(http.Handler) http.Handler) {
	mux.Handle("POST /raw-transactions", auth(http.HandlerFunc(c.Create)))
}
