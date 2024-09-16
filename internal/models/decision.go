package models

import "github.com/google/uuid"

type Decision struct {
	UserId   uuid.UUID
	BidId    uuid.UUID
	Decision DecisionType
}
