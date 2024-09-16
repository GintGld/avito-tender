package user

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"tender/internal/service"
	"tender/internal/service/user/mocks"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestVerifyUser(t *testing.T) {
	type args struct {
		ctx      context.Context
		username string
	}
	type res struct {
		ok  bool
		err error
	}
	tests := []struct {
		name    string
		args    args
		res     res
		wantErr error
	}{
		{
			name: "exists",
			res: res{
				ok:  true,
				err: nil,
			},
			wantErr: nil,
		},
		{
			name: "not exists",
			res: res{
				ok:  false,
				err: nil,
			},
			wantErr: service.ErrUserNotFound,
		},
		{
			name: "unknow error",
			res: res{
				ok:  false,
				err: errors.New("failed sql"),
			},
			wantErr: errors.New("User.Validate: failed sql"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			employeeStorage := mocks.NewEmployeeStorage(t)

			employeeStorage.
				On("VerifyUser", mock.Anything, tt.args.username).
				Return(tt.res.ok, tt.res.err)

			user := User{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				employeeStorage: employeeStorage,
			}

			err := user.Validate(tt.args.ctx, tt.args.username)

			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}

func TestPermissions(t *testing.T) {
	type args struct {
		ctx      context.Context
		username string
		orgId    uuid.UUID
	}
	type res struct {
		ok  bool
		err error
	}
	tests := []struct {
		name    string
		args    args
		res     res
		wantErr error
	}{
		{
			name: "allowed",
			res: res{
				ok:  true,
				err: nil,
			},
			wantErr: nil,
		},
		{
			name: "not allowed",
			res: res{
				ok:  false,
				err: nil,
			},
			wantErr: service.ErrNotEnoughPrivileges,
		},
		{
			name: "Unknown error",
			res: res{
				ok:  false,
				err: errors.New("sql error"),
			},
			wantErr: errors.New("User.Permission: sql error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			employeeStorage := mocks.NewEmployeeStorage(t)

			employeeStorage.
				On("VerifyUserPermission", mock.Anything, tt.args.username, tt.args.orgId).
				Return(tt.res.ok, tt.res.err)

			user := User{
				log: slog.New(slog.NewJSONHandler(
					os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
				employeeStorage: employeeStorage,
			}

			err := user.Permission(tt.args.ctx, tt.args.username, tt.args.orgId)

			if tt.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr.Error())
			}
		})
	}
}
