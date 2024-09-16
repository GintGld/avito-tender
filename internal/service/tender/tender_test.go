package tender

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	ptr "tender/internal/lib/utils/pointers"
	"tender/internal/models"
	"tender/internal/service"
	"tender/internal/service/tender/mocks"
	"tender/internal/storage"
)

var (
	ID_UUID  = uuid.MustParse("98abb192-f64d-44d6-9fcb-a2b0844c62bd")
	ID_UUID2 = uuid.MustParse("9cee2253-3d20-4f88-8bb4-5118cc7932f8")
	ORG_UUID = uuid.MustParse("002f9d2b-cd76-4921-8e53-21dbde75f993")
)

func TestNewTender(t *testing.T) {
	type args struct {
		ctx       context.Context
		tenderNew models.TenderNew
	}
	type want struct {
		tender models.TenderOut
		err    error
	}
	type validateRes struct {
		err error
	}
	type insertTenderRes struct {
		tender models.Tender
		err    error
	}
	tests := []struct {
		name            string
		args            args
		want            want
		validateRes     *validateRes
		insertTenderRes *insertTenderRes
	}{
		{
			name: "main line",
			args: args{context.Background(), models.TenderNew{
				CreatorUsername: "user",
				TenderBase: models.TenderBase{
					OrgId: ORG_UUID,
				},
			}},
			want: want{models.TenderOut{
				Version:   1,
				CreatedAt: time.Unix(10000, 0),
				Id:        ID_UUID,
				TenderBase: models.TenderBase{
					OrgId: ORG_UUID,
				},
			}, nil},
			validateRes: &validateRes{nil},
			insertTenderRes: &insertTenderRes{models.Tender{
				Version:   1,
				CreatedAt: time.Unix(10000, 0),
				Id:        ID_UUID,
				TenderBase: models.TenderBase{
					OrgId: ORG_UUID,
				},
			}, nil},
		},
		{
			name:        "user not found",
			args:        args{},
			want:        want{models.TenderOut{}, service.ErrUserNotFound},
			validateRes: &validateRes{service.ErrUserNotFound},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := mocks.NewUserService(t)
			tStorage := mocks.NewTenderStorage(t)

			tStorage.
				On("Begin", tt.args.ctx).
				Return(tt.args.ctx, nil)
			if tt.validateRes != nil {
				user.
					On("Validate", mock.Anything, tt.args.tenderNew.CreatorUsername).
					Return(tt.validateRes.err)
				if tt.insertTenderRes != nil {
					tStorage.
						On("InsertTender", mock.Anything, mock.Anything).
						Return(tt.insertTenderRes.tender, tt.insertTenderRes.err)

					if tt.insertTenderRes.err == nil {
						tStorage.
							On("Commit", tt.args.ctx).
							Return(nil)
					}
				}
			}
			tStorage.
				On("Rollback", tt.args.ctx).
				Return(nil)

			tender := Tender{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				userSrv:       user,
				tenderStorage: tStorage,
			}

			res, err := tender.New(tt.args.ctx, tt.args.tenderNew)
			if tt.want.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.tender, res)
			} else {
				assert.EqualError(t, err, tt.want.err.Error())
			}
		})
	}
}

func TestAll(t *testing.T) {
	type args struct {
		ctx           context.Context
		limit, offset int32
		serviceType   []models.ServiceType
	}
	type want struct {
		tender []models.TenderOut
		err    error
	}
	type tendersRes struct {
		tenders []models.Tender
		err     error
	}
	tests := []struct {
		name       string
		args       args
		want       want
		tendersRes tendersRes
	}{
		{
			name: "main line",
			args: args{limit: 3},
			want: want{[]models.TenderOut{
				{Id: ID_UUID, Version: 3, CreatedAt: time.Unix(0, 0), TenderBase: models.TenderBase{OrgId: ORG_UUID}},
			}, nil},
			tendersRes: tendersRes{[]models.Tender{
				{Id: ID_UUID, Version: 3, CreatedAt: time.Unix(0, 0), TenderBase: models.TenderBase{OrgId: ORG_UUID}},
			}, nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tStorage := mocks.NewTenderStorage(t)

			tStorage.
				On("Begin", tt.args.ctx).
				Return(tt.args.ctx, nil)
			tStorage.
				On("Tenders", tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.serviceType).
				Return(tt.tendersRes.tenders, tt.tendersRes.err)
			if tt.tendersRes.err == nil {
				tStorage.
					On("Commit", tt.args.ctx).
					Return(nil)
			}
			tStorage.
				On("Rollback", tt.args.ctx).
				Return(nil)

			tender := Tender{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				userSrv:       nil,
				tenderStorage: tStorage,
			}

			res, err := tender.All(tt.args.ctx, tt.args.limit, tt.args.offset, tt.args.serviceType)
			if tt.want.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.tender, res)
			} else {
				assert.EqualError(t, err, tt.want.err.Error())
			}
		})
	}
}

func TestSetStatus(t *testing.T) {
	type args struct {
		ctx      context.Context
		username string
		id       uuid.UUID
		status   models.TenderStatus
	}
	type want struct {
		tender models.TenderOut
		err    error
	}
	type validateRes struct {
		err error
	}
	type tenderRes struct {
		tender models.Tender
		err    error
	}
	type permissionRes struct {
		err error
	}
	type setStatusRes struct {
		tender models.Tender
		err    error
	}
	tests := []struct {
		name          string
		args          args
		validateRes   *validateRes
		tendersRes    *tenderRes
		permissionRes *permissionRes
		setStatusRes  *setStatusRes
		want          want
	}{
		{
			name:        "main line",
			args:        args{username: "user", id: ID_UUID, status: models.TenderCreated},
			validateRes: &validateRes{nil},
			tendersRes: &tenderRes{models.Tender{
				Id:        ID_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				Status:    models.TenderClosed,
				TenderBase: models.TenderBase{
					OrgId: ORG_UUID,
				}}, nil},
			permissionRes: &permissionRes{nil},
			setStatusRes: &setStatusRes{models.Tender{
				Id:        ID_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				Status:    models.TenderCreated,
				TenderBase: models.TenderBase{
					OrgId: ORG_UUID,
				}}, nil},
			want: want{models.TenderOut{
				Id:        ID_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				Status:    models.TenderCreated,
				TenderBase: models.TenderBase{
					OrgId: ORG_UUID,
				}}, nil},
		},
		{
			name:        "tender not found",
			args:        args{username: "name", id: ID_UUID, status: models.TenderCreated},
			validateRes: &validateRes{nil},
			tendersRes:  &tenderRes{models.Tender{}, storage.ErrTenderNotFound},
			want:        want{models.TenderOut{}, service.ErrTenderNotFound},
		},
		{
			name:          "no permissions",
			args:          args{username: "name", id: ID_UUID, status: models.TenderCreated},
			validateRes:   &validateRes{nil},
			tendersRes:    &tenderRes{models.Tender{}, nil},
			permissionRes: &permissionRes{service.ErrNotEnoughPrivileges},
			want:          want{models.TenderOut{}, service.ErrNotEnoughPrivileges},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := mocks.NewUserService(t)
			tStorage := mocks.NewTenderStorage(t)

			tStorage.
				On("Begin", tt.args.ctx).
				Return(tt.args.ctx, nil)
			if tt.validateRes != nil {
				user.
					On("Validate", tt.args.ctx, tt.args.username).
					Return(tt.validateRes.err)
			}
			if tt.tendersRes != nil {
				tStorage.
					On("Tender", tt.args.ctx, tt.args.id).
					Return(tt.tendersRes.tender, tt.tendersRes.err)
			}
			if tt.permissionRes != nil {
				user.
					On("Permission", tt.args.ctx, tt.args.username, tt.tendersRes.tender.OrgId).
					Return(tt.permissionRes.err)
			}
			if tt.setStatusRes != nil {
				tStorage.
					On("TenderSetStatus", tt.args.ctx, tt.args.id, tt.args.status).
					Return(tt.setStatusRes.tender, tt.setStatusRes.err)
				if tt.tendersRes.err == nil {
					tStorage.
						On("Commit", tt.args.ctx).
						Return(nil)
				}
			}
			tStorage.
				On("Rollback", tt.args.ctx).
				Return(nil)

			tender := Tender{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				userSrv:       user,
				tenderStorage: tStorage,
			}

			res, err := tender.SetStatus(tt.args.ctx, tt.args.username, tt.args.id, tt.args.status)
			if tt.want.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.tender, res)
			} else {
				assert.EqualError(t, err, tt.want.err.Error())
			}
		})
	}
}

func TestEdit(t *testing.T) {
	type args struct {
		ctx      context.Context
		username string
		id       uuid.UUID
		patch    models.TenderPatch
	}
	type want struct {
		tender models.TenderOut
		err    error
	}
	type validateRes struct {
		err error
	}
	type tenderRes struct {
		tender models.Tender
		err    error
	}
	type permissionRes struct {
		err error
	}
	type updateRes struct {
		err error
	}
	type saveTenderRes struct {
		err error
	}
	tests := []struct {
		name          string
		args          args
		validateRes   *validateRes
		tenderRes     *tenderRes
		permissionRes *permissionRes
		updateRes     *updateRes
		saveTenderRes *saveTenderRes
		want          want
	}{
		{
			name: "main line",
			args: args{username: "user", id: ID_UUID, patch: models.TenderPatch{
				Desc:        ptr.Ptr("new desc"),
				ServiceType: ptr.Ptr(models.Delivery),
			}},
			validateRes: &validateRes{nil},
			tenderRes: &tenderRes{models.Tender{
				Id:        ID_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				TenderBase: models.TenderBase{
					OrgId:       ORG_UUID,
					Desc:        "old desc",
					ServiceType: models.Manufacture,
				},
			}, nil},
			permissionRes: &permissionRes{nil},
			updateRes:     &updateRes{nil},
			saveTenderRes: &saveTenderRes{nil},
			want: want{models.TenderOut{
				Id:        ID_UUID,
				Version:   3,
				CreatedAt: time.Unix(10, 0),
				TenderBase: models.TenderBase{
					OrgId:       ORG_UUID,
					Desc:        "new desc",
					ServiceType: models.Delivery,
				},
			}, nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := mocks.NewUserService(t)
			tStorage := mocks.NewTenderStorage(t)
			rollbackSrv := mocks.NewRollbackService(t)

			tStorage.
				On("Begin", tt.args.ctx).
				Return(tt.args.ctx, nil)
			if tt.validateRes != nil {
				user.
					On("Validate", tt.args.ctx, tt.args.username).
					Return(tt.validateRes.err)
			}
			if tt.tenderRes != nil {
				tStorage.
					On("Tender", tt.args.ctx, tt.args.id).
					Return(tt.tenderRes.tender, tt.tenderRes.err)
			}
			if tt.permissionRes != nil {
				user.
					On("Permission", tt.args.ctx, tt.args.username, tt.tenderRes.tender.OrgId).
					Return(tt.permissionRes.err)
			}
			if tt.updateRes != nil {
				tender := tt.tenderRes.tender
				tender.Patch(tt.args.patch)
				tender.Version += 1

				tStorage.
					On("UpdateTender", tt.args.ctx, tender).
					Return(tt.updateRes.err)
			}
			if tt.saveTenderRes != nil {
				rollbackSrv.
					On("SaveTender", tt.args.ctx, tt.tenderRes.tender).
					Return(tt.saveTenderRes.err)
				if tt.saveTenderRes.err == nil {
					tStorage.
						On("Commit", tt.args.ctx).
						Return(nil)
				}
			}
			tStorage.
				On("Rollback", tt.args.ctx).
				Return(nil)

			tender := Tender{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				userSrv:       user,
				tenderStorage: tStorage,
				rollbackSrv:   rollbackSrv,
			}

			res, err := tender.Edit(tt.args.ctx, tt.args.username, tt.args.id, tt.args.patch)
			if tt.want.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.tender, res)
			} else {
				assert.EqualError(t, err, tt.want.err.Error())
			}
		})
	}
}
