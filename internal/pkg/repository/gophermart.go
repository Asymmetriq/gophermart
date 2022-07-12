package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Asymmetriq/gophermart/internal/config"
	"github.com/Asymmetriq/gophermart/internal/pkg/model"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	// Auth
	SaveUser(ctx context.Context, user model.User) error
	GetUser(ctx context.Context, user model.User) (model.User, error)

	// Orders
	SaveOrder(ctx context.Context, orderNumber model.Order, tx *sqlx.Tx) error
	UpdateOrder(ctx context.Context, order model.Order, tx *sqlx.Tx) error
	GetOrders(ctx context.Context, userID string) ([]model.Order, error)
	GetUnprocessedOrders(ctx context.Context) ([]model.Order, error)

	// Balances
	UpsertBalance(ctx context.Context, userID string, value *float64, tx *sqlx.Tx) error
	WithdrawBalance(ctx context.Context, userID string, value float64, tx *sqlx.Tx) error
	GetAllBalance(ctx context.Context, userID string) (model.Balance, error)
	GetCurrentBalance(ctx context.Context, userID string, tx *sqlx.Tx) (float64, error)

	// Withdrawals
	SaveWithdrawal(ctx context.Context, wth model.Withdrawal, tx *sqlx.Tx) error
	GetWithdrawals(ctx context.Context, userID string) ([]model.Withdrawal, error)

	// Transaction wrapper
	DoInTransaction(ctx context.Context, f func(ctx context.Context, tx *sqlx.Tx) error) (err error)
}

func NewRepository(cfg config.Config, db *sqlx.DB) Repository {
	return &martRepository{
		DB: db,
	}
}

type martRepository struct {
	DB *sqlx.DB
}

func (r *martRepository) SaveUser(ctx context.Context, user model.User) error {
	insertStatement := `INSERT INTO users(id, login, password) 
	VALUES(:id, :login, :password)
	ON CONFLICT (login) DO NOTHING`

	res, err := r.DB.NamedExecContext(ctx, insertStatement, user)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return model.ErrUserAlreadyExists
	}
	return nil
}

func (r *martRepository) GetUser(ctx context.Context, user model.User) (model.User, error) {
	selectStament := "SELECT id, login, password, created_at FROM users WHERE login=$1"

	var dbUser model.User
	if err := r.DB.GetContext(ctx, &dbUser, selectStament, user.Login); err != nil {
		return model.User{}, fmt.Errorf("select user: %w", err)
	}
	return dbUser, nil
}

func (r *martRepository) SaveOrder(ctx context.Context, newOrder model.Order, tx *sqlx.Tx) error {
	selectStatement := `SELECT * FROM orders WHERE order_number=$1 LIMIT 1`
	var order model.Order
	err := tx.GetContext(ctx, &order, selectStatement, newOrder.Number)
	if err == nil {
		if order.UserID == newOrder.UserID {
			return model.ErrExistsForThisUser
		}
		return model.ErrExistsForOtherUser
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("select order: %w", err)
	}

	insertStatement := `INSERT INTO orders(order_number, order_status, user_id, accrual) 
	VALUES (:order_number, :order_status, :user_id, :accrual)`
	_, err = r.DB.NamedExecContext(ctx, insertStatement, &newOrder)
	return err
}

func (r *martRepository) UpdateOrder(ctx context.Context, order model.Order, tx *sqlx.Tx) error {
	updateStatement := `UPDATE orders SET order_status=:order_status, accrual=:accrual`
	_, err := tx.NamedExecContext(ctx, updateStatement, &order)
	return err
}

func (r *martRepository) GetOrders(ctx context.Context, userID string) ([]model.Order, error) {
	selectStatement := `SELECT order_number, order_status, accrual, uploaded_at FROM orders WHERE user_id=$1`
	var orders []model.Order
	err := r.DB.SelectContext(ctx, &orders, selectStatement, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNoOrders
		}
		return nil, fmt.Errorf("select orders: %w", err)
	}
	if len(orders) == 0 {
		return nil, model.ErrNoOrders
	}
	return orders, nil
}

func (r *martRepository) GetAllBalance(ctx context.Context, userID string) (balance model.Balance, err error) {
	selectStatement := `SELECT * FROM balances WHERE user_id=$1`
	if err := r.DB.GetContext(ctx, &balance, selectStatement, userID); err != nil {
		return model.Balance{}, fmt.Errorf("select all balance: %w", err)
	}
	return balance, nil
}

func (r *martRepository) GetCurrentBalance(ctx context.Context, userID string, tx *sqlx.Tx) (balance float64, err error) {
	selectStatement := `SELECT current_balance FROM balances WHERE user_id=$1 LIMIT 1`
	if err := tx.GetContext(ctx, &balance, selectStatement, userID); err != nil {
		return balance, fmt.Errorf("select current balance: %w", err)
	}
	return balance, nil
}

func (r *martRepository) SaveWithdrawal(ctx context.Context, wth model.Withdrawal, tx *sqlx.Tx) error {
	insertStatement := `INSERT INTO withdrawals(order_number, user_id, sum) VALUES(:order_number, :user_id, :sum)`
	_, err := tx.NamedExecContext(ctx, insertStatement, &wth)
	return err
}

func (r *martRepository) GetWithdrawals(ctx context.Context, userID string) (withdrawals []model.Withdrawal, err error) {
	selectStatement := `SELECT order_number, sum, processed_at FROM withdrawals WHERE user_id=$1`
	if err := r.DB.SelectContext(ctx, &withdrawals, selectStatement, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, model.ErrNoWithdrawals
		}
		return nil, fmt.Errorf("select withdrawals: %w", err)
	}
	if len(withdrawals) == 0 {
		return nil, model.ErrNoWithdrawals
	}
	return withdrawals, nil
}

func (r *martRepository) UpsertBalance(ctx context.Context, userID string, value *float64, tx *sqlx.Tx) error {
	var balance float64
	if value != nil {
		balance = *value
	}
	upsertStatement := `INSERT INTO balances(current_balance, user_id) VALUES($1, $2)
	ON CONFLICT(user_id) DO UPDATE SET
	current_balance=(balances.current_balance+$1)
	WHERE balances.user_id=$2`

	_, err := tx.ExecContext(ctx, upsertStatement, balance, userID)
	return err
}
func (r *martRepository) WithdrawBalance(ctx context.Context, userID string, value float64, tx *sqlx.Tx) error {
	updateStatement := `UPDATE balances SET current_balance=current_balance-$1, withdrawn=withdrawn+$1 WHERE user_id=$2`
	_, err := tx.ExecContext(ctx, updateStatement, value, userID)
	return err
}

func (r *martRepository) GetUnprocessedOrders(ctx context.Context) ([]model.Order, error) {
	selectStatement := `SELECT order_number, user_id FROM orders WHERE order_status IN ('PROCESSING', 'NEW')`
	var orders []model.Order
	if err := r.DB.SelectContext(ctx, &orders, selectStatement); err != nil {
		return nil, fmt.Errorf("select unprocessed orders: %v", err)
	}
	return orders, nil
}
