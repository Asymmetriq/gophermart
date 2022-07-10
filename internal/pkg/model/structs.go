package model

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type User struct {
	ID        string    `json:"id" db:"id"`
	Login     string    `json:"login" db:"login"`
	Password  string    `json:"password" db:"password"`
	TokenHash string    `json:"token_hash" db:"token_hash"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Order struct {
	Number     int       `json:"number" db:"order_number"`
	Status     string    `json:"status" db:"order_status"`
	UserID     string    `json:"-" db:"user_id"`
	Accrual    string    `json:"accrual,omitempty" db:"accrual"`
	UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
}

type Balance struct {
	UserID    string  `json:"-" db:"user_id"`
	Current   float64 `json:"current_balance"`
	Withdrawn float64 `json:"withdrawn"`
}

type Withdrawal struct {
	OrderNumber string    `json:"order" db:"order_number"`
	UserID      string    `json:"-" db:"user_id"`
	Sum         float64   `json:"sum" db:"sum"`
	ProcessedAt time.Time `json:"processed_at" db:"processed_at"`
}

func (u *User) Validate() error {
	return validation.ValidateStruct(u,
		validation.Field(&u.Login),
		validation.Field(&u.Password, validation.Required),
	)
}
