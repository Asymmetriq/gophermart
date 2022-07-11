package gophermart

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Asymmetriq/gophermart/internal/pkg/model"
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
		log.Printf("updateOrders: %v", err)
		return
	}
	if len(orders) == 0 {
		return
	}

	newOrders := make([]model.Order, 0, len(orders))
	for _, order := range orders {
		resp, err := s.AccrualClient.GetOrderInfo(order.Number)
		if err != nil {
			log.Printf("accrual order info: %v", err)
			return
		}
		defer resp.Body.Close()

		var newInfo model.Order
		if err := json.NewDecoder(resp.Body).Decode(&newInfo); err != nil {
			log.Printf("accrual order info: %v", err)
			return
		}

		newInfo.UserID = order.UserID
		newOrders = append(newOrders, newInfo)
	}
	if err = s.Storage.DoInTransaction(ctx, func(ctx context.Context, tx *sqlx.Tx) error {
		for _, order := range newOrders {
			if err := s.Storage.UpdateOrder(ctx, order, tx); err != nil {
				return fmt.Errorf("update order info: %v", err)
			}
			if order.Accrual == nil {
				return nil
			}
			if err := s.Storage.UpsertBalance(ctx, order.UserID, order.Accrual, tx); err != nil {
				return fmt.Errorf("update order info: %v", err)
			}
		}
		return nil
	}); err != nil {
		log.Printf("background tx: %v", err)
	}
}
