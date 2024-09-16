package service

import "errors"

var (
	ErrUserNotFound         = errors.New("user not found")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrTenderNotFound       = errors.New("tender not found")
	ErrBidNotFound          = errors.New("bid not found")
	ErrVersionNotFound      = errors.New("version not found")
	ErrReviewsNotFound      = errors.New("reviews not found")
	ErrAuthorNotFound       = errors.New("author not found")

	ErrNotEnoughPrivileges = errors.New("not enought privileges")
)
