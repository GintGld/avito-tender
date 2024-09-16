package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	valid "tender/internal/lib/validate"
)

type TenderBase struct {
	OrgId       uuid.UUID   `json:"organizationId"`
	Name        string      `json:"name"`
	Desc        string      `json:"description"`
	ServiceType ServiceType `json:"serviceType"`
}

type TenderNew struct {
	TenderBase
	CreatorUsername string `json:"creatorUsername"`
}

func (t *TenderNew) validate() error {
	if err := valid.Validate(t.Name, "tender name", 100); err != nil {
		return NewParseError(err.Error())
	}

	if err := valid.Validate(t.CreatorUsername, "creator username", 100); err != nil {
		return NewParseError(err.Error(), true)
	}

	if len(t.Desc) > 500 {
		return NewParseError("description must not be longer than 100 characters")
	}

	return nil
}

func (t *TenderNew) UnmarshalJSON(data []byte) error {
	type _tenderNew TenderNew

	var tmp _tenderNew
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	t.TenderBase = tmp.TenderBase
	t.CreatorUsername = tmp.CreatorUsername

	if err := t.validate(); err != nil {
		return err
	}

	return nil
}

func (t *TenderNew) ToTender() Tender {
	return Tender{
		TenderBase: t.TenderBase,
		Id:         uuid.Nil,
		Status:     TenderCreated,
		Version:    1,
	}
}

type TenderPatch struct {
	Name        *string      `json:"name"`
	Desc        *string      `json:"description"`
	ServiceType *ServiceType `json:"serviceType"`
}

func (t *TenderPatch) validate() error {
	if t.Name != nil && len(*t.Name) > 100 {
		return NewParseError("name must not be longer than 100 characters")
	}

	if t.Desc != nil && len(*t.Desc) > 500 {
		return NewParseError("description must not be longer than 100 characters")
	}

	return nil
}

func (t *TenderPatch) UnmarshalJSON(data []byte) error {
	type _tenderPatch struct {
		Name        *string      `json:"name"`
		Desc        *string      `json:"description"`
		ServiceType *ServiceType `json:"serviceType"`
	}

	var tmp _tenderPatch
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	t.Desc = tmp.Desc
	t.Name = tmp.Name
	t.ServiceType = tmp.ServiceType

	if err := t.validate(); err != nil {
		return err
	}

	return nil
}

type TenderOut struct {
	TenderBase
	Id        uuid.UUID    `json:"id"`
	Status    TenderStatus `json:"status"`
	Version   int32        `json:"version"`
	CreatedAt time.Time    `json:"createdAt"`
}

type Tender struct {
	TenderBase
	Id        uuid.UUID
	Status    TenderStatus
	Version   int32
	CreatedAt time.Time
}

func (t *Tender) ToOut() TenderOut {
	return TenderOut{
		TenderBase: t.TenderBase,
		Id:         t.Id,
		Status:     t.Status,
		Version:    t.Version,
		CreatedAt:  t.CreatedAt,
	}
}

// Patch applies patch to tender.
func (t *Tender) Patch(patch TenderPatch) {
	if patch.Name != nil {
		t.Name = *patch.Name
	}
	if patch.Desc != nil {
		t.Desc = *patch.Desc
	}
	if patch.ServiceType != nil {
		t.ServiceType = *patch.ServiceType
	}
}
