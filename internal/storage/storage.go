package storage

import (
	"errors"
)

var (
	ErrOrgNotFound     = errors.New("org not found")
	ErrTenderNotFound  = errors.New("tender not found")
	ErrBidNotFound     = errors.New("bid not found")
	ErrVersionNotFound = errors.New("version not found")
)
