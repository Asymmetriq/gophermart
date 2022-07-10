package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func DoInTransaction(ctx context.Context, db sqlx.DB, f func(ctx context.Context, tx *sqlx.Tx) error) (err error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("can't open transaction: %w", err)
	}

	defer func() {
		p := recover()
		switch {
		case p != nil:
			_ = tx.Rollback()
			panic(p)
		case err != nil:
			_ = tx.Rollback()
		default:
			err = tx.Commit()
		}
	}()

	return f(ctx, tx)
}
