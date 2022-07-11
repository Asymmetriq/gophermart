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

const (
	MaxReqNum = 5
	BatchSize = 20
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

	GetAll(ctx context.Context, tx *sqlx.Tx) ([]model.Balance, error)
}

func NewRepository(cfg config.Config, db *sqlx.DB) Repository {
	return &martRepository{
		DB: db,
	}
}

type martRepository struct {
	DB *sqlx.DB
}

func (r *martRepository) GetAll(ctx context.Context, tx *sqlx.Tx) ([]model.Balance, error) {
	temp := `SELECT * from balances`
	var balances []model.Balance
	err := tx.SelectContext(ctx, &balances, temp)
	if err != nil {
		return nil, err
	}
	return balances, nil
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
	// TODO
	err := r.DB.GetContext(ctx, &order, selectStatement, newOrder.Number)
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

func (r *martRepository) GetAllBalance(ctx context.Context, userID string) (model.Balance, error) {
	selectStatement := `SELECT * FROM balances WHERE user_id=$1`
	var balance model.Balance
	if err := r.DB.GetContext(ctx, &balance, selectStatement, userID); err != nil {
		return model.Balance{}, fmt.Errorf("select all balance: %w", err)
	}
	return balance, nil
}

func (r *martRepository) GetCurrentBalance(ctx context.Context, userID string, tx *sqlx.Tx) (float64, error) {
	selectStatement := `SELECT current_balance FROM balances WHERE user_id=$1 LIMIT 1`
	var balance float64
	if err := tx.GetContext(ctx, &balance, selectStatement, userID); err != nil {
		return 0, fmt.Errorf("select current balance: %w", err)
	}
	return balance, nil
}

func (r *martRepository) SaveWithdrawal(ctx context.Context, wth model.Withdrawal, tx *sqlx.Tx) error {
	insertStatement := `INSERT INTO withdrawals(order_number, user_id, sum) VALUES(:order_number, :user_id, :sum)`
	_, err := tx.NamedExecContext(ctx, insertStatement, &wth)
	return err
}

func (r *martRepository) GetWithdrawals(ctx context.Context, userID string) ([]model.Withdrawal, error) {
	selectStatement := `SELECT order_number, sum, processed_at FROM withdrawals WHERE user_id=$1`
	var withdrawals []model.Withdrawal
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

// func (dbr *dbRepository) SetURL(ctx context.Context, entry models.StorageEntry) error {
// 	stmnt := `
// 	INSERT INTO urls(id, short_url, original_url, user_id)
// 	VALUES (:id, :short_url, :original_url, :user_id)
// 	ON CONFLICT (original_url) DO NOTHING`

// 	res, err := dbr.DB.NamedExecContext(ctx, stmnt, &entry)
// 	if err != nil {
// 		return err
// 	}
// 	if n, e := res.RowsAffected(); e == nil && n == 0 {
// 		return models.ErrAlreadyExists
// 	}
// 	return err
// }

// func (dbr *dbRepository) SetBatchURLs(ctx context.Context, entries []models.StorageEntry) error {
// 	if len(entries) == 0 {
// 		return nil
// 	}
// 	stmnt := `
// 	INSERT INTO urls(id, short_url, original_url, user_id)
// 	VALUES (:id, :short_url, :original_url, :user_id)
// 	ON CONFLICT (id) DO NOTHING`

// 	res, err := dbr.DB.NamedExecContext(ctx, stmnt, entries)
// 	if err != nil {
// 		return err
// 	}
// 	if n, e := res.RowsAffected(); e == nil && n == 0 {
// 		return models.ErrAlreadyExists
// 	}
// 	return err
// }

// func (dbr *dbRepository) GetURL(ctx context.Context, id string) (string, error) {
// 	var row models.StorageEntry
// 	if err := dbr.DB.GetContext(ctx, &row, "SELECT original_url, deleted FROM urls WHERE id=$1", id); err != nil {
// 		return "", fmt.Errorf("no original url found with shortcut %q", id)
// 	}
// 	if row.Deleted {
// 		return "", models.ErrDeleted
// 	}
// 	return row.OriginalURL, nil
// }

// func (dbr *dbRepository) GetAllURLs(ctx context.Context, userID string) ([]models.StorageEntry, error) {
// 	stmnt := "SELECT original_url, short_url FROM urls WHERE user_id=$1 AND deleted=false"

// 	var rows []models.StorageEntry
// 	if err := dbr.DB.SelectContext(ctx, &rows, stmnt, userID); err != nil {
// 		return nil, fmt.Errorf("no data  found with userID %q", userID)
// 	}
// 	return rows, nil
// }

// func (dbr *dbRepository) BatchDelete(req models.DeleteRequest) {
// 	go func(delReq models.DeleteRequest) {
// 		dbr.batchChannel <- delReq
// 	}(req)
// 	dbr.Signal()
// 	dbr.once.Do(func() {
// 		dbr.backgroundDelete()
// 	})
// }

// func (dbr *dbRepository) PingContext(ctx context.Context) error {
// 	return dbr.DB.PingContext(ctx)
// }

// func (dbr *dbRepository) deleteBatch(userID string, IDs []string) {
// 	stmnt := "UPDATE urls SET deleted=true WHERE user_id=$1 AND id=any($2);"

// 	for i := 0; i < len(IDs); i += BatchSize {
// 		end := i + BatchSize
// 		if end > len(IDs) {
// 			end = len(IDs)
// 		}
// 		_, err := dbr.DB.Exec(stmnt, userID, IDs[i:end])
// 		if err != nil {
// 			log.Printf("async delete: %v", err)
// 		}
// 	}
// }

// func (dbr *dbRepository) backgroundDelete() {
// 	go func() {
// 		defer func() {
// 			if p := recover(); p != nil {
// 				log.Printf("recovered from %v", p)
// 			}
// 		}()

// 		for {
// 			select {
// 			case <-dbr.signalTimer.C:
// 				for userID, IDs := range dbr.groupedRequests {
// 					dbr.deleteBatch(userID, IDs)
// 				}
// 				dbr.groupedRequests = make(map[string][]string)
// 				dbr.isTimerRunning = false

// 			case req, ok := <-dbr.batchChannel:
// 				if !ok {
// 					return
// 				}
// 				dbr.groupRequests(req)
// 			}
// 		}
// 	}()
// }
// func (dbr *dbRepository) groupRequests(req models.DeleteRequest) {
// 	if _, ok := dbr.groupedRequests[req.UserID]; ok {
// 		dbr.groupedRequests[req.UserID] = append(dbr.groupedRequests[req.UserID], req.IDs...)
// 	}
// 	dbr.groupedRequests[req.UserID] = req.IDs
// }

// func (dbr *dbRepository) Close() error {
// 	close(dbr.batchChannel)
// 	return dbr.DB.Close()
// }
