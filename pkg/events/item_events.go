package events

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

const (
	BidDomain   = "bid"
	BidExchange = "bid.bidding"
)

const (
	BidPlacedEvent = "bid.placed"
)

const (
	EventVersionV1 = "v1"
)

type BidPlacePayload struct {
	ID        uuid.UUID
	ItemID    uuid.UUID
	UserID    uuid.UUID
	Amount    decimal.Decimal
	CreatedAt time.Time
	UpdatedAt time.Time
}
