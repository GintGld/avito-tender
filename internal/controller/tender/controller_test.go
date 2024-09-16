package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"tender/internal/controller/tender/mocks"
	ptr "tender/internal/lib/utils/pointers"
	"tender/internal/models"
	"tender/internal/service"
)

var (
	ID_UUID  = uuid.MustParse("98abb192-f64d-44d6-9fcb-a2b0844c62bd")
	ID_UUID2 = uuid.MustParse("9cee2253-3d20-4f88-8bb4-5118cc7932f8")
	ORG_UUID = uuid.MustParse("002f9d2b-cd76-4921-8e53-21dbde75f993")
)

func Test_tenderController_new(t *testing.T) {
	type fields struct {
		Timeout time.Duration
	}
	type req struct {
		body string
	}
	type tenderRes struct {
		tender models.TenderOut
		err    error
	}
	type resp struct {
		body string
		code int
	}
	tests := []struct {
		name      string
		fields    fields
		req       req
		tenderRes *tenderRes
		resp      resp
	}{
		{
			name:   "main line",
			fields: fields{time.Hour},
			req: req{`{
				"name": "some name",
				"description": "awful description",
				"serviceType": "Construction",
				"status": "Created",
				"organizationId": "002f9d2b-cd76-4921-8e53-21dbde75f993",
				"creatorUsername": "user"
			}`},
			tenderRes: &tenderRes{models.TenderOut{
				TenderBase: models.TenderBase{
					OrgId:       ORG_UUID,
					Name:        "some name",
					Desc:        "awful description",
					ServiceType: models.Construction,
				},
				Id:        ID_UUID,
				Status:    models.TenderCreated,
				Version:   1,
				CreatedAt: time.Unix(1136203445, 0),
			}, nil},
			resp: resp{`{
				"id": "98abb192-f64d-44d6-9fcb-a2b0844c62bd",
				"name": "some name",
				"description": "awful description",
				"status": "Created",
				"organizationId": "002f9d2b-cd76-4921-8e53-21dbde75f993",
				"serviceType": "Construction",
				"version": 1,
				"createdAt": "2006-01-02T15:04:05+03:00"
			}`, 200},
		},
		{
			name:   "user not found",
			fields: fields{time.Hour},
			req: req{`{
				"name": "some name",
				"description": "awful description",
				"serviceType": "Construction",
				"status": "Created",
				"organizationId": "002f9d2b-cd76-4921-8e53-21dbde75f993",
				"creatorUsername": "user"
			}`},
			tenderRes: &tenderRes{models.TenderOut{}, service.ErrUserNotFound},
			resp:      resp{`{"reason":"user not found"}`, 401},
		},
		{
			name:   "tender name empty",
			fields: fields{time.Hour},
			req: req{`{
				"name": "",
				"description": "awful description",
				"serviceType": "Construction",
				"status": "Created",
				"organizationId": "002f9d2b-cd76-4921-8e53-21dbde75f993",
				"creatorUsername": "user"
			}`},
			resp: resp{`{"reason":"tender name must not be empty"}`, 400},
		},
		{
			name:   "username empty",
			fields: fields{time.Hour},
			req: req{`{
				"name": "some name",
				"description": "awful description",
				"serviceType": "Construction",
				"status": "Created",
				"organizationId": "002f9d2b-cd76-4921-8e53-21dbde75f993",
				"creatorUsername": ""
			}`},
			resp: resp{`{"reason":"creator username must not be empty"}`, 401},
		},
		{
			name:   "invalid uuid",
			fields: fields{time.Hour},
			req: req{`{
				"name": "some name",
				"description": "awful description",
				"serviceType": "Construction",
				"status": "Created",
				"organizationId": "invalid org id",
				"creatorUsername": "user"
			}`},
			resp: resp{`{
				"reason": ""
			}`, 400},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tender := mocks.NewTender(t)

			if tt.tenderRes != nil {
				tender.
					On("New", mock.Anything, models.TenderNew{
						TenderBase: models.TenderBase{
							OrgId:       ORG_UUID,
							Name:        "some name",
							Desc:        "awful description",
							ServiceType: models.Construction,
						},
						CreatorUsername: "user",
					}).
					Return(tt.tenderRes.tender, tt.tenderRes.err)
			}

			tr := &tenderController{
				Timeout: tt.fields.Timeout,
				tender:  tender,
			}

			app := fiber.New()
			app.Post("/new", tr.new)

			req := httptest.NewRequest("POST", "/new", bytes.NewBuffer([]byte(tt.req.body)))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, int(tr.Timeout.Seconds()))
			require.NoError(t, err)

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.JSONEq(t, tt.resp.body, string(respBody))
			assert.Equal(t, tt.resp.code, resp.StatusCode)
		})
	}
}

func Test_tenderController_edit(t *testing.T) {
	type fields struct {
		Timeout time.Duration
	}
	type req struct {
		body     string
		username string
		tenderId uuid.UUID
	}
	type editRes struct {
		tender models.TenderOut
		err    error
	}
	type resp struct {
		body string
		code int
	}
	tests := []struct {
		name    string
		fields  fields
		req     req
		editRes *editRes
		resp    resp
	}{
		{
			name:   "main line",
			fields: fields{time.Hour},
			req: req{`{
				"name": "new name",
				"description": "new awful description",
				"serviceType": "Delivery"
			}`, "user", ID_UUID},
			editRes: &editRes{models.TenderOut{
				TenderBase: models.TenderBase{
					OrgId:       ORG_UUID,
					Name:        "new name",
					Desc:        "new awful description",
					ServiceType: models.Delivery,
				},
				Id:        ID_UUID,
				Status:    models.TenderCreated,
				Version:   2,
				CreatedAt: time.Unix(1136203445, 0),
			}, nil},
			resp: resp{`{
				"id": "98abb192-f64d-44d6-9fcb-a2b0844c62bd",
				"name": "new name",
				"description": "new awful description",
				"status": "Created",
				"organizationId": "002f9d2b-cd76-4921-8e53-21dbde75f993",
				"serviceType": "Delivery",
				"version": 2,
				"createdAt": "2006-01-02T15:04:05+03:00"
			}`, 200},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tender := mocks.NewTender(t)

			if tt.editRes != nil {
				tender.
					On("Edit", mock.Anything, tt.req.username, tt.req.tenderId, models.TenderPatch{
						Name:        ptr.Ptr("new name"),
						Desc:        ptr.Ptr("new awful description"),
						ServiceType: ptr.Ptr(models.Delivery),
					}).
					Return(tt.editRes.tender, tt.editRes.err)
			}

			tr := &tenderController{
				Timeout: tt.fields.Timeout,
				tender:  tender,
			}

			app := fiber.New()
			app.Patch("/:tenderId/edit", tr.edit)

			req := httptest.NewRequest(
				"PATCH",
				fmt.Sprintf("/%s/edit?username=%s", tt.req.tenderId.String(), tt.req.username),
				bytes.NewBuffer([]byte(tt.req.body)),
			)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req, int(tr.Timeout.Seconds()))
			require.NoError(t, err)

			respBody, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.JSONEq(t, tt.resp.body, string(respBody))
			assert.Equal(t, tt.resp.code, resp.StatusCode)
		})
	}
}
