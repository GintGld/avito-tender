package models

import (
	"encoding/json"
	ptr "tender/internal/lib/utils/pointers"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTenderNew(t *testing.T) {
	s := `{
		"name": "some name",
		"description": "awful description",
		"serviceType": "Construction",
		"status": "Created",
		"organizationId": "002f9d2b-cd76-4921-8e53-21dbde75f993",
		"creatorUsername": "user"
	}`
	expect := TenderNew{
		TenderBase: TenderBase{
			Name:        "some name",
			Desc:        "awful description",
			ServiceType: Construction,
			OrgId:       uuid.MustParse("002f9d2b-cd76-4921-8e53-21dbde75f993"),
		},
		CreatorUsername: "user",
	}

	var tender TenderNew

	err := json.Unmarshal([]byte(s), &tender)
	assert.NoError(t, err)
	assert.Equal(t, expect, tender)
}

func TestTenderPatch(t *testing.T) {
	s := `{
		"name" : "new name",
		"description" : "new description"
	}`
	expect := TenderPatch{
		Name: ptr.Ptr("new name"),
		Desc: ptr.Ptr("new description"),
	}

	var patch TenderPatch

	err := json.Unmarshal([]byte(s), &patch)
	assert.NoError(t, err)
	assert.Equal(t, expect, patch)
}
