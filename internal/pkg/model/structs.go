package model

import (
	"encoding/json"
	"fmt"
	"time"
)

type Status string

const (
	StatusNew        Status = "NEW"
	StatusProcessing Status = "PROCESSING"
	StatusProcessed  Status = "PROCESSED"
	StatusInvalid    Status = "INVALID"
)

type User struct {
	ID        string    `json:"id" db:"id"`
	Login     string    `json:"login" db:"login"`
	Password  string    `json:"password" db:"password"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Order struct {
	Number     string    `json:"number" db:"order_number"`
	Status     Status    `json:"status" db:"order_status"`
	UserID     string    `json:"-" db:"user_id"`
	Accrual    *float64  `json:"accrual,omitempty" db:"accrual"`
	UploadedAt time.Time `json:"uploaded_at" db:"uploaded_at"`
}

type Balance struct {
	UserID    string  `json:"-" db:"user_id"`
	Current   float64 `json:"current" db:"current_balance"`
	Withdrawn float64 `json:"withdrawn" db:"withdrawn"`
}

type Withdrawal struct {
	OrderNumber string    `json:"order" db:"order_number"`
	UserID      string    `json:"-" db:"user_id"`
	Sum         float64   `json:"sum" db:"sum"`
	ProcessedAt time.Time `json:"processed_at" db:"processed_at"`
}

func (o *Order) Marshal() ([]byte, error) {
	m, err := json.Marshal(o)
	if err != nil {
		return nil, fmt.Errorf("marshal Order: %w", err)
	}
	return m, nil
}
