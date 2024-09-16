package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"tender/internal/models"
	"tender/internal/service"
	"tender/internal/service/rollback/mocks"
	"tender/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

var (
	ID_UUID   = uuid.MustParse("98abb192-f64d-44d6-9fcb-a2b0844c62bd")
	TENDER_ID = uuid.MustParse("9cee2253-3d20-4f88-8bb4-5118cc7932f8")
	ORG_UUID  = uuid.MustParse("002f9d2b-cd76-4921-8e53-21dbde75f993")
)

func TestSaveTender(t *testing.T) {
	type args struct {
		ctx    context.Context
		tender models.Tender
	}
	type saveTenderRes struct {
		err error
	}
	type want struct {
		err error
	}
	tests := []struct {
		name          string
		args          args
		saveTenderRes saveTenderRes
		want          want
	}{
		{
			name: "main line",
			args: args{tender: models.Tender{
				Id:     ID_UUID,
				Status: models.TenderPublished,
				TenderBase: models.TenderBase{
					OrgId:       ORG_UUID,
					Name:        "name",
					Desc:        "desc",
					ServiceType: models.Manufacture,
				},
			}},
			saveTenderRes: saveTenderRes{nil},
			want:          want{nil},
		},
		{
			name:          "some error",
			args:          args{},
			saveTenderRes: saveTenderRes{errors.New("some pgx error")},
			want:          want{errors.New("Rollback.SaveTender: some pgx error")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rollbackStorage := mocks.NewRollbackStorage(t)

			rollbackStorage.
				On("SaveTender", tt.args.ctx, tt.args.tender).
				Return(tt.saveTenderRes.err)

			rollback := Rollback{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				rollbackStorage: rollbackStorage,
			}

			err := rollback.SaveTender(tt.args.ctx, tt.args.tender)
			if tt.want.err != nil {
				assert.EqualError(t, err, tt.want.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSaveBid(t *testing.T) {
	type args struct {
		ctx context.Context
		bid models.Bid
	}
	type saveBidRes struct {
		err error
	}
	type want struct {
		err error
	}
	tests := []struct {
		name       string
		args       args
		saveBidRes saveBidRes
		want       want
	}{
		{
			name: "main line",
			args: args{bid: models.Bid{
				Id:     ID_UUID,
				Status: models.BidPublished,
				BidBase: models.BidBase{
					TenderId:   TENDER_ID,
					Name:       "name",
					Desc:       "desc",
					AuthorType: models.Organization,
					AuthorId:   ORG_UUID,
				},
			}},
			saveBidRes: saveBidRes{nil},
			want:       want{nil},
		},
		{
			name:       "some error",
			args:       args{},
			saveBidRes: saveBidRes{errors.New("some pgx error")},
			want:       want{errors.New("Rollback.SaveBid: some pgx error")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rollbackStorage := mocks.NewRollbackStorage(t)

			rollbackStorage.
				On("SaveBid", tt.args.ctx, tt.args.bid).
				Return(tt.saveBidRes.err)

			rollback := Rollback{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				rollbackStorage: rollbackStorage,
			}

			err := rollback.SaveBid(tt.args.ctx, tt.args.bid)
			if tt.want.err != nil {
				assert.EqualError(t, err, tt.want.err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSwapTender(t *testing.T) {
	type args struct {
		ctx            context.Context
		tenderId       uuid.UUID
		version        int32
		outdatedTender models.Tender
	}
	type saveTenderRes struct {
		err error
	}
	type recoverTenderRes struct {
		tender models.Tender
		err    error
	}
	type want struct {
		tender models.Tender
		err    error
	}
	tests := []struct {
		name             string
		args             args
		saveTenderRes    *saveTenderRes
		recoverTenderRes *recoverTenderRes
		want             want
	}{
		{
			name: "main line",
			args: args{tenderId: TENDER_ID, version: 2, outdatedTender: models.Tender{
				Id:     ID_UUID,
				Status: models.TenderClosed,
				TenderBase: models.TenderBase{
					OrgId:       ORG_UUID,
					Name:        "name",
					Desc:        "desc",
					ServiceType: models.Delivery,
				},
				Version: 4,
			}},
			saveTenderRes: &saveTenderRes{nil},
			recoverTenderRes: &recoverTenderRes{models.Tender{
				Id:     ID_UUID,
				Status: models.TenderCreated,
				TenderBase: models.TenderBase{
					OrgId:       ORG_UUID,
					Name:        "old name",
					Desc:        "old desc",
					ServiceType: models.Construction,
				},
				Version: 2,
			}, nil},
			want: want{models.Tender{
				Id:     ID_UUID,
				Status: models.TenderCreated,
				TenderBase: models.TenderBase{
					OrgId:       ORG_UUID,
					Name:        "old name",
					Desc:        "old desc",
					ServiceType: models.Construction,
				},
				Version: 2,
			}, nil},
		},
		{
			name:             "invalied version",
			args:             args{},
			saveTenderRes:    &saveTenderRes{nil},
			recoverTenderRes: &recoverTenderRes{models.Tender{}, storage.ErrVersionNotFound},
			want:             want{models.Tender{}, service.ErrVersionNotFound},
		},
	}
	for _, tt := range tests {
		rollbackStorage := mocks.NewRollbackStorage(t)

		if tt.saveTenderRes != nil {
			rollbackStorage.
				On("SaveTender", tt.args.ctx, tt.args.outdatedTender).
				Return(tt.saveTenderRes.err)
		}
		if tt.recoverTenderRes != nil {
			rollbackStorage.
				On("RecoverTender", tt.args.ctx, tt.args.tenderId, tt.args.version).
				Return(tt.recoverTenderRes.tender, tt.recoverTenderRes.err)
		}

		rollback := Rollback{
			log: slog.New(slog.NewJSONHandler(
				os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
			rollbackStorage: rollbackStorage,
		}

		newTender, err := rollback.SwapTender(
			tt.args.ctx,
			tt.args.tenderId,
			tt.args.version,
			tt.args.outdatedTender,
		)
		assert.Equal(t, tt.want.tender, newTender)
		if tt.want.err == nil {
			assert.NoError(t, err)
		} else {
			assert.EqualError(t, err, tt.want.err.Error())
		}
	}
}

func TestSwapBid(t *testing.T) {
	type args struct {
		ctx         context.Context
		bidId       uuid.UUID
		version     int32
		outdatedBid models.Bid
	}
	type saveBidRes struct {
		err error
	}
	type recoverBidRes struct {
		bid models.Bid
		err error
	}
	type want struct {
		bid models.Bid
		err error
	}
	tests := []struct {
		name          string
		args          args
		saveBidRes    *saveBidRes
		recoverBidRes *recoverBidRes
		want          want
	}{
		{
			name: "main line",
			args: args{bidId: TENDER_ID, version: 2, outdatedBid: models.Bid{
				Id:     ID_UUID,
				Status: models.BidCreated,
				BidBase: models.BidBase{
					Name: "name",
					Desc: "desc",
				},
				Version: 4,
			}},
			saveBidRes: &saveBidRes{nil},
			recoverBidRes: &recoverBidRes{models.Bid{
				Id:     ID_UUID,
				Status: models.BidCanceled,
				BidBase: models.BidBase{
					Name: "old name",
					Desc: "old desc",
				},
				Version: 2,
			}, nil},
			want: want{models.Bid{
				Id:     ID_UUID,
				Status: models.BidCanceled,
				BidBase: models.BidBase{
					Name: "old name",
					Desc: "old desc",
				},
				Version: 2,
			}, nil},
		},
		{
			name:          "invalied version",
			args:          args{},
			saveBidRes:    &saveBidRes{nil},
			recoverBidRes: &recoverBidRes{models.Bid{}, storage.ErrVersionNotFound},
			want:          want{models.Bid{}, service.ErrVersionNotFound},
		},
	}
	for _, tt := range tests {
		rollbackStorage := mocks.NewRollbackStorage(t)

		if tt.saveBidRes != nil {
			rollbackStorage.
				On("SaveBid", tt.args.ctx, tt.args.outdatedBid).
				Return(tt.saveBidRes.err)
		}
		if tt.recoverBidRes != nil {
			rollbackStorage.
				On("RecoverBid", tt.args.ctx, tt.args.bidId, tt.args.version).
				Return(tt.recoverBidRes.bid, tt.recoverBidRes.err)
		}

		rollback := Rollback{
			log: slog.New(slog.NewJSONHandler(
				os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
			rollbackStorage: rollbackStorage,
		}

		newTender, err := rollback.SwapBid(
			tt.args.ctx,
			tt.args.bidId,
			tt.args.version,
			tt.args.outdatedBid,
		)
		assert.Equal(t, tt.want.bid, newTender)
		if tt.want.err == nil {
			assert.NoError(t, err)
		} else {
			assert.EqualError(t, err, tt.want.err.Error())
		}
	}
}
