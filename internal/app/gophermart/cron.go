package gophermart

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	jobInterval = 10 * time.Second
)

func (s *Service) updateOrdersBackground(ctx context.Context) {
	ticker := time.NewTicker(jobInterval)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				log.Printf("recovered from panic: %v", p)
			}
		}()

		for {
			select {
			case <-ticker.C:
				s.updateOrders(ctx)
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *Service) updateOrders(ctx context.Context) {
	orders, err := s.Storage.GetUnprocessedOrders(ctx)
	if err != nil {
		log.Printf("fetching unprocessed orders: %v", err)
		return
	}
	if len(orders) == 0 {
		log.Print("no unprocessed orders")
	}

	for i := range orders {
		orders[i], err = s.AccrualClient.GetOrderInfo(orders[i])
		if err != nil {
			log.Printf("accrual order info: %v", err)
			return
		}
		if err = s.Storage.DoInTransaction(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
			if err := s.Storage.UpdateOrder(ctx, orders[i], tx); err != nil {
				return fmt.Errorf("update order info: %v", err)
			}
			if orders[i].Accrual == nil {
				return nil
			}
			if err := s.Storage.UpsertBalance(ctx, orders[i].UserID, orders[i].Accrual, tx); err != nil {
				return fmt.Errorf("update order info: %v", err)
			}
			return nil
		}); err != nil {
			log.Printf("background tx: %v", err)
		}
	}

}
