package models

import (
	"time"

	"github.com/google/uuid"
)

type ReviewBase struct {
	Id        uuid.UUID
	Desc      string    `json:"description"`
	CreatedAt time.Time `json:"createdAt"`
}

type ReviewOut struct {
	ReviewBase
}

type Review struct {
	ReviewBase
	BidId      uuid.UUID
	AuthorName string
}

func (r *Review) ToOut() ReviewOut {
	return ReviewOut{
		ReviewBase: r.ReviewBase,
	}
}
