package raw_transactions

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Service stamps every message with user, taken from the API key's owner and never from the request.
type Service struct {
	col  *mongo.Collection
	user string
}

func NewService(db *mongo.Database, user string) *Service {
	return &Service{col: db.Collection(Collection), user: user}
}

func (s *Service) Create(ctx context.Context, message string) (*RawTransaction, error) {
	rt := &RawTransaction{ID: bson.NewObjectID(), User: s.user, Message: message, ReceivedAt: time.Now().UTC(), Status: StatusPending}
	if _, err := s.col.InsertOne(ctx, rt); err != nil {
		return nil, err
	}
	return rt, nil
}
