package gophermart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Asymmetriq/gophermart/internal/app/gophermart/middleware"
	"github.com/Asymmetriq/gophermart/internal/pkg/auth"
	"github.com/Asymmetriq/gophermart/internal/pkg/luhn"
	"github.com/Asymmetriq/gophermart/internal/pkg/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

func (s *Service) registerHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetUser(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	hashedPass, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user.ID = uuid.NewString()
	user.Password = string(hashedPass)

	if err = s.Storage.SaveUser(r.Context(), user); err != nil {
		http.Error(w, err.Error(), model.GetErrorCode(err))
		return
	}

	token, err := auth.GenerateToken(user, s.Config.GetTokenSignKey())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w.WriteHeader(http.StatusOK)
}

func (s *Service) loginHandler(w http.ResponseWriter, r *http.Request) {
	user, err := middleware.GetUser(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	dbUser, err := s.Storage.GetUser(r.Context(), user)
	if err != nil {
		http.Error(w, err.Error(), model.GetErrorCode(err))
		return
	}
	if !auth.Authenticate(dbUser, user) {
		http.Error(w, "wrong login / password", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user, s.Config.GetTokenSignKey())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w.WriteHeader(http.StatusOK)
}

func (s *Service) processOrderHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var number int
	if err = json.NewDecoder(r.Body).Decode(&number); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	order := model.Order{
		Number: strconv.Itoa(number),
		Status: model.StatusNew,
		UserID: userID,
	}
	if !luhn.Valid(order.Number) {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
	}

	if err = s.Storage.DoInTransaction(r.Context(), func(ctx context.Context, tx *sqlx.Tx) error {
		if err = s.Storage.SaveOrder(r.Context(), order, tx); err != nil {
			return fmt.Errorf("save order tx: %w", err)
		}
		if err = s.Storage.UpsertBalance(ctx, userID, order.Accrual, tx); err != nil {
			return fmt.Errorf("save balance tx: %w", err)
		}
		return nil
	}); err != nil {
		if errors.Is(err, model.ErrExistsForThisUser) {
			w.WriteHeader(http.StatusOK)
		} else {
			http.Error(w, err.Error(), model.GetErrorCode(err))
		}
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (s *Service) getOrdersHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	orders, err := s.Storage.GetOrders(r.Context(), userID)
	if err != nil {
		if errors.Is(err, model.ErrNoOrders) {
			w.WriteHeader(http.StatusNoContent)
		} else {
			http.Error(w, err.Error(), model.GetErrorCode(err))
		}
		return
	}
	marshalled, err := json.Marshal(orders)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(marshalled)
}

func (s *Service) withdrawHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	wth := model.Withdrawal{
		UserID: userID,
	}
	if err = json.NewDecoder(r.Body).Decode(&wth); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !luhn.Valid(wth.OrderNumber) {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
	}

	if err = s.Storage.DoInTransaction(r.Context(), func(ctx context.Context, tx *sqlx.Tx) error {
		balance, err := s.Storage.GetCurrentBalance(ctx, userID, tx)
		if err != nil {
			return fmt.Errorf("get balance tx: %w", err)
		}
		if balance < wth.Sum {
			return model.ErrNotEnoughBalance
		}
		if err = s.Storage.SaveWithdrawal(ctx, wth, tx); err != nil {
			return fmt.Errorf("save withdrawal tx: %w", err)
		}
		if err = s.Storage.WithdrawBalance(ctx, userID, wth.Sum, tx); err != nil {
			return fmt.Errorf("save withdrawal tx: %w", err)
		}
		return nil
	}); err != nil {
		http.Error(w, err.Error(), model.GetErrorCode(err))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Service) getWithdrawalsHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	withdrawals, err := s.Storage.GetWithdrawals(r.Context(), userID)
	if err != nil {
		if errors.Is(err, model.ErrNoWithdrawals) {
			w.WriteHeader(http.StatusNoContent)
		} else {
			http.Error(w, err.Error(), model.GetErrorCode(err))
		}
		return
	}
	marshalled, err := json.Marshal(withdrawals)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(marshalled)
}

func (s *Service) getBalanceHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserID(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	balance, err := s.Storage.GetAllBalance(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	marshalled, err := json.Marshal(balance)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(marshalled)
}
