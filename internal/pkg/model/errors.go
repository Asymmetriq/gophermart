package model

import (
	"errors"
	"net/http"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrExistsForThisUser  = errors.New("order already exists for this user")
	ErrExistsForOtherUser = errors.New("order already exists for another user")
	ErrNotEnoughBalance   = errors.New("not enough accrual points")
	ErrNoOrders           = errors.New("no orders for user")
	ErrNoWithdrawals      = errors.New("no withdrawals for user")
)

func GetErrorCode(err error) int {
	switch {
	case errors.Is(err, ErrNotEnoughBalance):
		return http.StatusPaymentRequired
	case errors.Is(err, ErrUserAlreadyExists), errors.Is(err, ErrExistsForOtherUser):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}

}
