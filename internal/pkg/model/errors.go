package model

import (
	"errors"
	"net/http"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrExistsForThisUser  = errors.New("order already exists for this user")
	ErrExistsForOtherUser = errors.New("order already exists for another user")
	ErrNotEnoughPoints    = errors.New("not enough accrual points")
)

func GetErrorCode(err error) int {
	switch {
	case errors.Is(err, ErrUserAlreadyExists), errors.Is(err, ErrExistsForOtherUser):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}

}
