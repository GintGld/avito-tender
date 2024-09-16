package models

import (
	"encoding/json"
	"time"

	valid "tender/internal/lib/validate"

	"github.com/google/uuid"
)

type BidBase struct {
	TenderId   uuid.UUID  `json:"tenderId"`
	Name       string     `json:"name"`
	Desc       string     `json:"description"`
	AuthorType AuthorType `json:"authorType"`
	AuthorId   uuid.UUID  `json:"authorId"`
}

type BidNew struct {
	BidBase
}

func (b *BidNew) validate() error {
	if err := valid.Validate(b.Name, "name", 100); err != nil {
		return NewParseError(err.Error())
	}

	if len(b.Desc) > 500 {
		return NewParseError("description must not be longer than 500 characters")
	}

	return nil
}

func (b *BidNew) UnmarshalJSON(data []byte) error {
	type _bidNew BidNew

	var tmp _bidNew
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	b.BidBase = tmp.BidBase

	if err := b.validate(); err != nil {
		return err
	}

	return nil
}

func (b *BidNew) ToBid() Bid {
	return Bid{
		BidBase: b.BidBase,
		Version: 1,
		Status:  BidCreated,
	}
}

type BidPatch struct {
	Name *string `json:"name"`
	Desc *string `json:"description"`
}

func (b *BidPatch) validate() error {
	if b.Name != nil && len(*b.Name) > 100 {
		return NewParseError("organization id must not be empty")
	}

	if b.Desc != nil && len(*b.Desc) > 500 {
		return NewParseError("description must not be longer than 100 characters")
	}

	return nil
}

func (b *BidPatch) UnmarshalJSON(data []byte) error {
	type _bidPatch BidPatch

	var tmp _bidPatch
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	b.Name = tmp.Name
	b.Desc = tmp.Desc

	if err := b.validate(); err != nil {
		return err
	}

	return nil
}

type BidOut struct {
	BidBase
	Id        uuid.UUID `json:"id"`
	Version   int32     `json:"version"`
	Status    BidStatus `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

type Bid struct {
	BidBase
	Id        uuid.UUID
	Version   int32
	Status    BidStatus
	CreatedAt time.Time
}

func (b *Bid) ToOut() BidOut {
	return BidOut{
		Id:        b.Id,
		BidBase:   b.BidBase,
		Version:   b.Version,
		Status:    b.Status,
		CreatedAt: b.CreatedAt,
	}
}

// Patch applies patch to bid.
func (b *Bid) Patch(patch BidPatch) {
	if patch.Name != nil {
		b.Name = *patch.Name
	}
	if patch.Desc != nil {
		b.Desc = *patch.Desc
	}
}
