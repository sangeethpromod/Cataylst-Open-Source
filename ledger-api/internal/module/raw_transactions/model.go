package raw_transactions

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	Collection    = "raw_transactions"
	StatusPending = "pending"
	MaxBodyBytes  = 2048
)

type createRequest struct {
	Message string `json:"message"`
}

type RawTransaction struct {
	ID         bson.ObjectID `bson:"_id"`
	User       string        `bson:"user"`
	Message    string        `bson:"message"`
	ReceivedAt time.Time     `bson:"received_at"`
	Status     string        `bson:"status"`
}
