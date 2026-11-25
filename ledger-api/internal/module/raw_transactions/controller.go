package raw_transactions

import (
	"log"
	"net/http"
	"strings"

	"ledger-api/internal/httpx"
)

type Controller struct {
	svc *Service
}

func NewController(svc *Service) *Controller { return &Controller{svc: svc} }

func (c *Controller) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	// Message is stored verbatim; trim is only used for the emptiness check.
	if strings.TrimSpace(req.Message) == "" || len(req.Message) > MaxBodyBytes {
		httpx.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "message must be non-empty and at most 2048 bytes"})
		return
	}
	rt, err := c.svc.Create(r.Context(), req.Message)
	if err != nil {
		log.Printf("raw_transactions insert failed: %v", err) // never log the message itself
		httpx.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not store message"})
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]string{"id": rt.ID.Hex()})
}
