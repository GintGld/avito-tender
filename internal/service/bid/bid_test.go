package bid

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
	"tender/internal/service/bid/mocks"
	"tender/internal/storage"
)

var (
	BID_UUID    = uuid.MustParse("98abb192-f64d-44d6-9fcb-a2b0844c62bd")
	BID_UUID2   = uuid.MustParse("9cee2253-3d20-4f88-8bb4-5118cc7932f8")
	ORG_UUID    = uuid.MustParse("002f9d2b-cd76-4921-8e53-21dbde75f993")
	AUTH_UUID   = uuid.MustParse("ce61bdc8-d435-454a-92c7-5e51c9a21907")
	TENDER_UUID = uuid.MustParse("0284744f-ee56-485d-b124-173315723ba6")
	REVIEW_UUID = uuid.MustParse("75129d25-acbe-4e64-9e57-342781135841")
)

func TestNewBid(t *testing.T) {
	type args struct {
		ctx    context.Context
		bidNew models.BidNew
	}
	type want struct {
		bid models.BidOut
		err error
	}
	type validateUserRes struct {
		err error
	}
	type validateOrgRes struct {
		err error
	}
	type insertBidRes struct {
		bid models.Bid
		err error
	}
	tests := []struct {
		name            string
		args            args
		want            want
		validateUserRes *validateUserRes
		validateOrgRes  *validateOrgRes
		insertBidRes    *insertBidRes
	}{
		{
			name: "main line org",
			args: args{context.Background(), models.BidNew{
				BidBase: models.BidBase{
					AuthorType: models.Organization,
					AuthorId:   AUTH_UUID,
				},
			}},
			validateOrgRes: &validateOrgRes{nil},
			insertBidRes: &insertBidRes{models.Bid{
				Id:        BID_UUID,
				Version:   1,
				CreatedAt: time.Unix(10000, 0),
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.Organization,
				},
			}, nil},
			want: want{models.BidOut{
				Id:        BID_UUID,
				Version:   1,
				CreatedAt: time.Unix(10000, 0),
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.Organization,
				},
			}, nil},
		},
		{
			name: "main line user",
			args: args{context.Background(), models.BidNew{
				BidBase: models.BidBase{
					AuthorType: models.User,
					AuthorId:   AUTH_UUID,
				},
			}},
			validateUserRes: &validateUserRes{nil},
			insertBidRes: &insertBidRes{models.Bid{
				Id:        BID_UUID,
				Version:   1,
				CreatedAt: time.Unix(10000, 0),
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.User,
				},
			}, nil},
			want: want{models.BidOut{
				Id:        BID_UUID,
				Version:   1,
				CreatedAt: time.Unix(10000, 0),
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.User,
				},
			}, nil},
		},
		{
			name: "main line org",
			args: args{context.Background(), models.BidNew{
				BidBase: models.BidBase{
					AuthorType: models.Organization,
					AuthorId:   AUTH_UUID,
				},
			}},
			validateOrgRes: &validateOrgRes{nil},
			insertBidRes: &insertBidRes{models.Bid{
				Id:        BID_UUID,
				Version:   1,
				CreatedAt: time.Unix(10000, 0),
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.Organization,
				},
			}, nil},
			want: want{models.BidOut{
				Id:        BID_UUID,
				Version:   1,
				CreatedAt: time.Unix(10000, 0),
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.Organization,
				},
			}, nil},
		},
		{
			name: "user invalid",
			args: args{bidNew: models.BidNew{
				BidBase: models.BidBase{
					AuthorType: models.User,
					AuthorId:   AUTH_UUID,
				}},
			},
			validateUserRes: &validateUserRes{service.ErrUserNotFound},
			want:            want{models.BidOut{}, service.ErrUserNotFound},
		},
		{
			name: "org invalid",
			args: args{bidNew: models.BidNew{
				BidBase: models.BidBase{
					AuthorType: models.Organization,
					AuthorId:   AUTH_UUID,
				}},
			},
			validateOrgRes: &validateOrgRes{service.ErrOrganizationNotFound},
			want:           want{models.BidOut{}, service.ErrOrganizationNotFound},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := mocks.NewUserService(t)
			bStorage := mocks.NewBidStorage(t)

			bStorage.
				On("Begin", tt.args.ctx).
				Return(tt.args.ctx, nil)
			if tt.validateUserRes != nil {
				user.
					On("ValidateUserId", tt.args.ctx, tt.args.bidNew.AuthorId).
					Return(tt.validateUserRes.err)
			}
			if tt.validateOrgRes != nil {
				user.
					On("ValidateOrgId", tt.args.ctx, tt.args.bidNew.AuthorId).
					Return(tt.validateOrgRes.err)
			}
			if tt.insertBidRes != nil {
				bStorage.
					On("InsertBid", mock.Anything, mock.Anything).
					Return(tt.insertBidRes.bid, tt.insertBidRes.err)

				if tt.insertBidRes.err == nil {
					bStorage.
						On("Commit", tt.args.ctx).
						Return(nil)
				}
			}
			bStorage.
				On("Rollback", tt.args.ctx).
				Return(nil)

			bid := Bid{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				userSrv:    user,
				bidStorage: bStorage,
			}

			res, err := bid.New(tt.args.ctx, tt.args.bidNew)
			if tt.want.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.bid, res)
			} else {
				assert.EqualError(t, err, tt.want.err.Error())
			}
		})
	}
}

func TestDecision(t *testing.T) {
	type args struct {
		ctx      context.Context
		username string
		bidId    uuid.UUID
		decision models.DecisionType
	}
	type validateRes struct {
		err error
	}
	type bidRes struct {
		bid models.Bid
		err error
	}
	type tenderRes struct {
		tender models.Tender
		err    error
	}
	type permissionRes struct {
		err error
	}
	type userIdRes struct {
		id  uuid.UUID
		err error
	}
	type insertDecRes struct {
		err error
	}
	type decisionsRes struct {
		decisions []models.Decision
		err       error
	}
	type orgSizeRes struct {
		size int64
		err  error
	}
	type updBidRes struct {
		err error
	}
	type want struct {
		bid models.BidOut
		err error
	}
	tests := []struct {
		name          string
		args          args
		validateRes   *validateRes
		bidRes        *bidRes
		tenderRes     *tenderRes
		permissionRes *permissionRes
		userIdRes     *userIdRes
		insertDecRes  *insertDecRes
		decisionsRes  *decisionsRes
		orgSizeRes    *orgSizeRes
		updBidRes     *updBidRes
		want          want
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := mocks.NewUserService(t)
			bStorage := mocks.NewBidStorage(t)
			tender := mocks.NewTenderService(t)

			bStorage.
				On("Begin", tt.args.ctx).
				Return(tt.args.ctx)
			if tt.validateRes != nil {
				user.
					On("Validate", tt.args.ctx, tt.args.username).
					Return(tt.validateRes.err)
			}
			if tt.bidRes != nil {
				bStorage.
					On("Bid", tt.args.ctx, tt.args.bidId).
					Return(tt.bidRes.bid, tt.bidRes.err)
			}
			if tt.tenderRes != nil {
				tender.
					On("Tender", tt.args.ctx, tt.bidRes.bid.TenderId).
					Return(tt.tenderRes.tender, tt.tenderRes.err)
			}
			if tt.permissionRes != nil {
				user.
					On("Permission", tt.args.ctx, tt.tenderRes.tender.OrgId).
					Return(tt.permissionRes.err)
			}
			if tt.userIdRes != nil {
				user.
					On("UserId", tt.args.ctx, tt.args.username).
					Return(tt.userIdRes.id, tt.userIdRes.err)
			}
			if tt.insertDecRes != nil {
				bStorage.
					On("InsertDecision", tt.args.ctx, nil). // TODO
					Return(tt.insertDecRes.err)
			}
			if tt.decisionsRes != nil {
				bStorage.
					On("Decisions", tt.args.ctx, tt.bidRes.bid.Id).
					Return(tt.decisionsRes.decisions, tt.decisionsRes.err)
			}
			if tt.orgSizeRes != nil {
				user.
					On("OrgSize", tt.args.ctx, tt.tenderRes.tender.OrgId).
					Return(tt.orgSizeRes.size, tt.orgSizeRes.err)
			}
			if tt.updBidRes != nil {
				bStorage.
					On("UpdateBid", tt.args.ctx, nil). // TODO
					Return(tt.updBidRes.err)

				if tt.updBidRes.err == nil {
					bStorage.
						On("Commit", tt.args.ctx).
						Return(nil)
				}
			}
			bStorage.
				On("Rollback", tt.args.ctx).
				Return(nil)

			bid := Bid{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				userSrv:    user,
				bidStorage: bStorage,
				tenderSrv:  tender,
			}

			res, err := bid.SubmitDecision(tt.args.ctx, tt.args.username, tt.args.bidId, tt.args.decision)
			assert.Equal(t, tt.want.bid, res)
			if tt.want.err == nil {
				assert.NoError(t, err)
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
		status   models.BidStatus
	}
	type want struct {
		bid models.BidOut
		err error
	}
	type validateRes struct {
		err error
	}
	type bidRes struct {
		bid models.Bid
		err error
	}
	type userIdRes struct {
		id  uuid.UUID
		err error
	}
	type permissionRes struct {
		err error
	}
	type setStatusRes struct {
		bid models.Bid
		err error
	}
	tests := []struct {
		name          string
		args          args
		validateRes   *validateRes
		bidsRes       *bidRes
		userIdRes     *userIdRes
		permissionRes *permissionRes
		setStatusRes  *setStatusRes
		want          want
	}{
		{
			name:        "main line user",
			args:        args{username: "user", id: BID_UUID, status: models.BidCreated},
			validateRes: &validateRes{nil},
			bidsRes: &bidRes{models.Bid{
				Id:        BID_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				Status:    models.BidCanceled,
				BidBase: models.BidBase{
					AuthorType: models.User,
					AuthorId:   AUTH_UUID,
				},
			}, nil},
			userIdRes: &userIdRes{AUTH_UUID, nil},
			setStatusRes: &setStatusRes{models.Bid{
				Id:        BID_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				Status:    models.BidCreated,
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.User,
				},
			}, nil},
			want: want{models.BidOut{
				Id:        BID_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				Status:    models.BidCreated,
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.User,
				},
			}, nil},
		},
		{
			name:        "main line org",
			args:        args{username: "user", id: BID_UUID, status: models.BidCreated},
			validateRes: &validateRes{nil},
			bidsRes: &bidRes{models.Bid{
				Id:        BID_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				Status:    models.BidCanceled,
				BidBase: models.BidBase{
					AuthorType: models.Organization,
					AuthorId:   AUTH_UUID,
				},
			}, nil},
			permissionRes: &permissionRes{nil},
			setStatusRes: &setStatusRes{models.Bid{
				Id:        BID_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				Status:    models.BidCreated,
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.Organization,
				},
			}, nil},
			want: want{models.BidOut{
				Id:        BID_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				Status:    models.BidCreated,
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.Organization,
				},
			}, nil},
		},
		{
			name:        "bid not found",
			args:        args{username: "name", id: BID_UUID, status: models.BidCreated},
			validateRes: &validateRes{nil},
			bidsRes:     &bidRes{models.Bid{}, storage.ErrBidNotFound},
			want:        want{models.BidOut{}, service.ErrBidNotFound},
		},
		{
			name:        "no permissions user",
			args:        args{username: "name", id: BID_UUID, status: models.BidCreated},
			validateRes: &validateRes{nil},
			bidsRes: &bidRes{models.Bid{
				BidBase: models.BidBase{
					AuthorType: models.User,
					AuthorId:   AUTH_UUID,
				},
			}, nil},
			userIdRes: &userIdRes{ORG_UUID, nil},
			want:      want{models.BidOut{}, service.ErrNotEnoughPrivileges},
		},
		{
			name:        "no permissions org",
			args:        args{username: "name", id: BID_UUID, status: models.BidCreated},
			validateRes: &validateRes{nil},
			bidsRes: &bidRes{models.Bid{
				BidBase: models.BidBase{
					AuthorType: models.Organization,
					AuthorId:   AUTH_UUID,
				},
			}, nil},
			permissionRes: &permissionRes{service.ErrNotEnoughPrivileges},
			want:          want{models.BidOut{}, service.ErrNotEnoughPrivileges},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := mocks.NewUserService(t)
			bStorage := mocks.NewBidStorage(t)

			bStorage.
				On("Begin", tt.args.ctx).
				Return(tt.args.ctx, nil)
			if tt.validateRes != nil {
				user.
					On("Validate", tt.args.ctx, tt.args.username).
					Return(tt.validateRes.err)
			}
			if tt.bidsRes != nil {
				bStorage.
					On("Bid", tt.args.ctx, tt.args.id).
					Return(tt.bidsRes.bid, tt.bidsRes.err)
			}
			if tt.userIdRes != nil {
				user.
					On("UserId", tt.args.ctx, tt.args.username).
					Return(tt.userIdRes.id, tt.userIdRes.err)
			}
			if tt.permissionRes != nil {
				user.
					On("Permission", tt.args.ctx, tt.args.username, tt.bidsRes.bid.AuthorId).
					Return(tt.permissionRes.err)
			}
			if tt.setStatusRes != nil {
				bStorage.
					On("BidSetStatus", tt.args.ctx, tt.args.id, tt.args.status).
					Return(tt.setStatusRes.bid, tt.setStatusRes.err)

				if tt.setStatusRes.err == nil {
					bStorage.
						On("Commit", tt.args.ctx).
						Return(nil)
				}
			}
			bStorage.
				On("Rollback", tt.args.ctx).
				Return(nil)

			bid := Bid{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				userSrv:    user,
				bidStorage: bStorage,
			}

			res, err := bid.SetStatus(tt.args.ctx, tt.args.username, tt.args.id, tt.args.status)
			if tt.want.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.bid, res)
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
		patch    models.BidPatch
	}
	type want struct {
		bid models.BidOut
		err error
	}
	type validateRes struct {
		err error
	}
	type bidRes struct {
		bid models.Bid
		err error
	}
	type userIdRes struct {
		id  uuid.UUID
		err error
	}
	type permissionRes struct {
		err error
	}
	type updateRes struct {
		err error
	}
	type saveBidSrc struct {
		err error
	}
	tests := []struct {
		name          string
		args          args
		validateRes   *validateRes
		bidRes        *bidRes
		userIdRes     *userIdRes
		permissionRes *permissionRes
		saveBidSrc    *saveBidSrc
		updateRes     *updateRes
		want          want
	}{
		{
			name: "main line org",
			args: args{username: "user", id: BID_UUID, patch: models.BidPatch{
				Name: ptr.Ptr("new name"),
				Desc: ptr.Ptr("new desc"),
			}},
			validateRes: &validateRes{nil},
			bidRes: &bidRes{models.Bid{
				Id:        BID_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.Organization,
					Desc:       "old desc",
					Name:       "old name",
				},
			}, nil},
			permissionRes: &permissionRes{nil},
			updateRes:     &updateRes{nil},
			saveBidSrc:    &saveBidSrc{nil},
			want: want{models.BidOut{
				Id:        BID_UUID,
				Version:   3,
				CreatedAt: time.Unix(10, 0),
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.Organization,
					Desc:       "new desc",
					Name:       "new name",
				},
			}, nil},
		},
		{
			name: "main line user",
			args: args{username: "user", id: BID_UUID, patch: models.BidPatch{
				Name: ptr.Ptr("new name"),
				Desc: ptr.Ptr("new desc"),
			}},
			validateRes: &validateRes{nil},
			bidRes: &bidRes{models.Bid{
				Id:        BID_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.User,
					Desc:       "old desc",
					Name:       "old name",
				},
			}, nil},
			userIdRes:  &userIdRes{AUTH_UUID, nil},
			updateRes:  &updateRes{nil},
			saveBidSrc: &saveBidSrc{nil},
			want: want{models.BidOut{
				Id:        BID_UUID,
				Version:   3,
				CreatedAt: time.Unix(10, 0),
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.User,
					Desc:       "new desc",
					Name:       "new name",
				},
			}, nil},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := mocks.NewUserService(t)
			bStorage := mocks.NewBidStorage(t)
			rollbackSrv := mocks.NewRollbackService(t)

			bStorage.
				On("Begin", tt.args.ctx).
				Return(tt.args.ctx, nil)
			if tt.validateRes != nil {
				user.
					On("Validate", tt.args.ctx, tt.args.username).
					Return(tt.validateRes.err)
			}
			if tt.bidRes != nil {
				bStorage.
					On("Bid", tt.args.ctx, tt.args.id).
					Return(tt.bidRes.bid, tt.bidRes.err)
			}
			if tt.userIdRes != nil {
				user.
					On("UserId", tt.args.ctx, tt.args.username).
					Return(tt.userIdRes.id, tt.userIdRes.err)
			}
			if tt.permissionRes != nil {
				user.
					On("Permission", tt.args.ctx, tt.args.username, tt.bidRes.bid.AuthorId).
					Return(tt.permissionRes.err)
			}
			if tt.updateRes != nil {
				newBid := tt.bidRes.bid
				newBid.Patch(tt.args.patch)
				newBid.Version += 1

				bStorage.
					On("UpdateBid", tt.args.ctx, newBid).
					Return(tt.updateRes.err)
			}
			if tt.saveBidSrc != nil {
				rollbackSrv.
					On("SaveBid", tt.args.ctx, tt.bidRes.bid).
					Return(tt.saveBidSrc.err)

				if tt.saveBidSrc.err == nil {
					bStorage.
						On("Commit", tt.args.ctx).
						Return(nil)
				}
			}
			bStorage.
				On("Rollback", tt.args.ctx).
				Return(nil)

			bid := Bid{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				userSrv:     user,
				bidStorage:  bStorage,
				rollbackSrv: rollbackSrv,
			}

			res, err := bid.Edit(tt.args.ctx, tt.args.username, tt.args.id, tt.args.patch)
			if tt.want.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.bid, res)
			} else {
				assert.EqualError(t, err, tt.want.err.Error())
			}
		})
	}
}

func TestReviews(t *testing.T) {
	type args struct {
		ctx               context.Context
		requester, author string
		tenderId          uuid.UUID
		limit, offset     int32
	}
	type want struct {
		reviews []models.ReviewOut
		err     error
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
	type reviewsRes struct {
		reviews []models.Review
		err     error
	}
	tests := []struct {
		name          string
		args          args
		valReqRes     *validateRes
		valAuthRes    *validateRes
		tenderRes     *tenderRes
		permissionRes *permissionRes
		reviewsRes    *reviewsRes
		want          want
	}{
		{
			name: "main line",
			args: args{
				requester: "user1",
				author:    "user2",
				tenderId:  TENDER_UUID,
				limit:     3,
				offset:    0,
			},
			valReqRes:  &validateRes{nil},
			valAuthRes: &validateRes{nil},
			tenderRes: &tenderRes{models.Tender{
				Id:        TENDER_UUID,
				Version:   2,
				CreatedAt: time.Unix(10, 0),
				TenderBase: models.TenderBase{
					OrgId: ORG_UUID,
					Desc:  "desc",
					Name:  "name",
				},
			}, nil},
			permissionRes: &permissionRes{nil},
			reviewsRes: &reviewsRes{[]models.Review{
				{
					BidId: BID_UUID,
					ReviewBase: models.ReviewBase{
						Id:        REVIEW_UUID,
						Desc:      "desc",
						CreatedAt: time.Unix(32, 0),
					},
				},
			}, nil},
			want: want{
				[]models.ReviewOut{
					{
						ReviewBase: models.ReviewBase{
							Id:        REVIEW_UUID,
							Desc:      "desc",
							CreatedAt: time.Unix(32, 0),
						},
					},
				},
				nil,
			},
		},
		{
			name:      "requester not found",
			args:      args{requester: "user1", author: "user2", tenderId: TENDER_UUID, limit: 5},
			valReqRes: &validateRes{service.ErrUserNotFound},
			want:      want{nil, service.ErrUserNotFound},
		},
		{
			name:       "author not found",
			args:       args{requester: "user1", author: "user2", tenderId: TENDER_UUID, limit: 5},
			valReqRes:  &validateRes{nil},
			valAuthRes: &validateRes{service.ErrUserNotFound},
			want:       want{nil, service.ErrAuthorNotFound},
		},
		{
			name:       "tender not found",
			args:       args{requester: "user1", author: "user2", tenderId: TENDER_UUID, limit: 5},
			valReqRes:  &validateRes{nil},
			valAuthRes: &validateRes{nil},
			tenderRes:  &tenderRes{models.Tender{}, service.ErrTenderNotFound},
			want:       want{nil, service.ErrTenderNotFound},
		},
		{
			name:          "no permissions for requester",
			args:          args{requester: "user1", author: "user2", tenderId: TENDER_UUID, limit: 5},
			valReqRes:     &validateRes{nil},
			valAuthRes:    &validateRes{nil},
			tenderRes:     &tenderRes{models.Tender{}, nil},
			permissionRes: &permissionRes{service.ErrNotEnoughPrivileges},
			want:          want{nil, service.ErrNotEnoughPrivileges},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := mocks.NewUserService(t)
			bStorage := mocks.NewBidStorage(t)
			tender := mocks.NewTenderService(t)

			bStorage.
				On("Begin", tt.args.ctx).
				Return(tt.args.ctx, nil)
			if tt.valReqRes != nil {
				user.
					On("Validate", tt.args.ctx, tt.args.requester).
					Return(tt.valReqRes.err)
			}
			if tt.valAuthRes != nil {
				user.
					On("Validate", tt.args.ctx, tt.args.author).
					Return(tt.valAuthRes.err)
			}
			if tt.tenderRes != nil {
				tender.
					On("Tender", tt.args.ctx, tt.args.tenderId).
					Return(tt.tenderRes.tender, tt.tenderRes.err)
			}
			if tt.permissionRes != nil {
				user.
					On("Permission", tt.args.ctx, tt.args.requester, tt.tenderRes.tender.OrgId).
					Return(tt.permissionRes.err)
			}
			if tt.reviewsRes != nil {
				bStorage.
					On("Reviews", tt.args.ctx, tt.args.tenderId, tt.args.author, tt.args.limit, tt.args.offset).
					Return(tt.reviewsRes.reviews, tt.reviewsRes.err)

				if tt.reviewsRes.err == nil {
					bStorage.
						On("Commit", tt.args.ctx).
						Return(nil)
				}
			}
			bStorage.
				On("Rollback", tt.args.ctx).
				Return(nil)

			bid := Bid{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				userSrv:    user,
				bidStorage: bStorage,
				tenderSrv:  tender,
			}

			res, err := bid.Reviews(tt.args.ctx, tt.args.requester, tt.args.author, tt.args.tenderId, tt.args.limit, tt.args.offset)
			if tt.want.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.reviews, res)
			} else {
				assert.EqualError(t, err, tt.want.err.Error())
			}
		})
	}
}

func TestFeedback(t *testing.T) {
	type args struct {
		ctx                context.Context
		username, feedback string
		bidId              uuid.UUID
	}
	type want struct {
		bid models.BidOut
		err error
	}
	type validateRes struct {
		err error
	}
	type bidRes struct {
		bid models.Bid
		err error
	}
	type tenderRes struct {
		tender models.Tender
		err    error
	}
	type permissionRes struct {
		err error
	}
	type insertReviewRes struct {
		id  uuid.UUID
		err error
	}
	tests := []struct {
		name            string
		args            args
		validateRes     *validateRes
		bidRes          *bidRes
		tenderRes       *tenderRes
		permissionRes   *permissionRes
		insertReviewRes *insertReviewRes
		want            want
	}{
		{
			name:        "main line org",
			args:        args{username: "user", bidId: BID_UUID, feedback: "feedback"},
			validateRes: &validateRes{nil},
			bidRes: &bidRes{models.Bid{
				Id:        BID_UUID,
				Version:   3,
				CreatedAt: time.Unix(34, 0),
				BidBase: models.BidBase{
					AuthorType: models.Organization,
					AuthorId:   AUTH_UUID,
					TenderId:   TENDER_UUID,
				},
			}, nil},
			tenderRes: &tenderRes{models.Tender{
				Version:   7,
				CreatedAt: time.Unix(10, 0),
				Id:        BID_UUID,
				TenderBase: models.TenderBase{
					OrgId: ORG_UUID,
				},
			}, nil},
			permissionRes:   &permissionRes{nil},
			insertReviewRes: &insertReviewRes{REVIEW_UUID, nil},
			want: want{models.BidOut{
				Id:        BID_UUID,
				Version:   3,
				CreatedAt: time.Unix(34, 0),
				BidBase: models.BidBase{
					AuthorId:   AUTH_UUID,
					AuthorType: models.Organization,
					TenderId:   TENDER_UUID,
				},
			}, nil},
		},
		{
			name:        "bid not found",
			args:        args{username: "user", bidId: BID_UUID, feedback: "feedback"},
			validateRes: &validateRes{nil},
			bidRes:      &bidRes{models.Bid{}, storage.ErrBidNotFound},
			want:        want{models.BidOut{}, service.ErrBidNotFound},
		},
		{
			name:        "tender not found",
			args:        args{username: "user", bidId: BID_UUID, feedback: "feedback"},
			validateRes: &validateRes{nil},
			bidRes:      &bidRes{models.Bid{}, nil},
			tenderRes:   &tenderRes{models.Tender{}, service.ErrTenderNotFound},
			want:        want{models.BidOut{}, service.ErrTenderNotFound},
		},
		{
			name:          "user without permissions",
			args:          args{username: "user", bidId: BID_UUID, feedback: "feedback"},
			validateRes:   &validateRes{nil},
			bidRes:        &bidRes{models.Bid{}, nil},
			tenderRes:     &tenderRes{models.Tender{}, nil},
			permissionRes: &permissionRes{service.ErrNotEnoughPrivileges},
			want:          want{models.BidOut{}, service.ErrNotEnoughPrivileges},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := mocks.NewUserService(t)
			bStorage := mocks.NewBidStorage(t)
			tender := mocks.NewTenderService(t)

			bStorage.
				On("Begin", tt.args.ctx).
				Return(tt.args.ctx, nil)
			if tt.validateRes != nil {
				user.
					On("Validate", tt.args.ctx, tt.args.username).
					Return(tt.validateRes.err)
			}
			if tt.bidRes != nil {
				bStorage.
					On("Bid", tt.args.ctx, tt.args.bidId).
					Return(tt.bidRes.bid, tt.bidRes.err)
			}
			if tt.tenderRes != nil {
				tender.
					On("Tender", tt.args.ctx, tt.bidRes.bid.TenderId).
					Return(tt.tenderRes.tender, tt.tenderRes.err)
			}
			if tt.permissionRes != nil {
				user.
					On("Permission", tt.args.ctx, tt.args.username, tt.tenderRes.tender.OrgId).
					Return(tt.permissionRes.err)
			}
			if tt.insertReviewRes != nil {
				review := models.Review{
					BidId: tt.bidRes.bid.Id,
					ReviewBase: models.ReviewBase{
						Desc: tt.args.feedback,
					},
					AuthorName: tt.args.username,
				}

				bStorage.
					On("InsertReview", tt.args.ctx, review).
					Return(tt.insertReviewRes.id, tt.insertReviewRes.err)

				if tt.insertReviewRes.err == nil {
					bStorage.
						On("Commit", tt.args.ctx).
						Return(nil)
				}
			}
			bStorage.
				On("Rollback", tt.args.ctx).
				Return(nil)

			bid := Bid{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				userSrv:    user,
				bidStorage: bStorage,
				tenderSrv:  tender,
			}

			res, err := bid.Feedback(tt.args.ctx, tt.args.username, tt.args.bidId, tt.args.feedback)
			if tt.want.err == nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.bid, res)
			} else {
				assert.EqualError(t, err, tt.want.err.Error())
			}
		})
	}
}
