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
	"github.com/Asymmetriq/gophermart/internal/pkg/model"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

func (s *Service) registerHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserStructKey).(model.User)
	if !ok {
		http.Error(w, "no user data provided", http.StatusBadRequest)
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
	user, ok := r.Context().Value(middleware.UserStructKey).(model.User)
	if !ok {
		http.Error(w, "no user data provided", http.StatusBadRequest)
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

	token, err := auth.GenerateToken(user, "test")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token))
	w.WriteHeader(http.StatusOK)
}

func (s *Service) processOrderHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "no user id in token found", http.StatusUnauthorized)
		return
	}

	var number int
	err := json.NewDecoder(r.Body).Decode(&number)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	order := model.Order{
		Number: strconv.Itoa(number),
		Status: model.StatusNew,
		UserID: userID,
	}
	if !Valid(order.Number) {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
	}
	// log.Println("CALLING ACCRUAL")
	// order = s.callAcural(order)

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

// func (s *Service) callAcural(order model.Order) model.Order {
// 	resp, err := s.AccrualClient.GetOrderInfo(order.Number)
// 	if err != nil {
// 		log.Printf("ACCRUAL: %v", err)
// 	}
// 	defer resp.Body.Close()
// 	if resp.StatusCode == http.StatusOK {
// 		var newInfo model.Order
// 		if err := json.NewDecoder(resp.Body).Decode(&newInfo); err != nil {
// 			log.Printf("ACCRUAL: %v", err)
// 		}
// 		if newInfo.Status == model.StatusProcessed {
// 			newInfo.Number = order.Number
// 			newInfo.UserID = order.UserID
// 			order = newInfo
// 		}
// 	}
// 	return order
// }

func (s *Service) getOrdersHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "no user id in token found", http.StatusUnauthorized)
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
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "no user id in token found", http.StatusUnauthorized)
		return
	}

	wth := model.Withdrawal{
		UserID: userID,
	}
	err := json.NewDecoder(r.Body).Decode(&wth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !Valid(wth.OrderNumber) {
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
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "no user id in token found", http.StatusUnauthorized)
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
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		http.Error(w, "no user id in token found", http.StatusUnauthorized)
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

// func (s *Service) getHandler(w http.ResponseWriter, r *http.Request) {
// 	shortID := chi.URLParam(r, "id")

// 	ogURL, err := s.Storage.GetURL(r.Context(), shortID)
// 	code := model.ParseGetError(err)
// 	if code == http.StatusBadRequest {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	http.Redirect(w, r, ogURL, code)
// }

// func (s *Service) postHandler(w http.ResponseWriter, r *http.Request) {
// 	userID, ok := r.Context().Value(cookie.Name).(string)
// 	if !ok {
// 		http.Error(w, "no userID provided", http.StatusBadRequest)
// 		return
// 	}

// 	defer r.Body.Close()
// 	b, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}
// 	if len(b) == 0 {
// 		http.Error(w, "no request body", http.StatusBadRequest)
// 		return
// 	}
// 	host := r.Host
// 	if u := s.Config.GetBaseURL(); len(u) != 0 {
// 		host = u
// 	}

// 	entry := model.NewStorageEntry(string(b), host, userID)
// 	err = s.Storage.SetURL(r.Context(), entry)
// 	code := model.ParsePostError(err)
// 	if code == http.StatusBadRequest {
// 		http.Error(w, err.Error(), code)
// 		return
// 	}
// 	w.Header().Set("Content-Type", "application/text")
// 	w.WriteHeader(code)
// 	w.Write([]byte(entry.ShortURL))
// }

// func (s *Service) jsonHandler(w http.ResponseWriter, r *http.Request) {
// 	userID, ok := r.Context().Value(cookie.Name).(string)
// 	if !ok {
// 		http.Error(w, "no userID provided", http.StatusBadRequest)
// 		return
// 	}

// 	var result struct {
// 		URL string `json:"url"`
// 	}
// 	err := json.NewDecoder(r.Body).Decode(&result)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}
// 	host := r.Host
// 	if u := s.Config.GetBaseURL(); len(u) != 0 {
// 		host = u
// 	}

// 	entry := model.NewStorageEntry(result.URL, host, userID)
// 	err = s.Storage.SetURL(r.Context(), entry)
// 	code := model.ParsePostError(err)
// 	if code == http.StatusBadRequest {
// 		http.Error(w, err.Error(), code)
// 		return
// 	}
// 	resp, err := json.Marshal(struct {
// 		Result string `json:"result"`
// 	}{
// 		Result: entry.ShortURL,
// 	})
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(code)
// 	w.Write(resp)
// }

// func (s *Service) batchHandler(w http.ResponseWriter, r *http.Request) {
// 	userID, ok := r.Context().Value(cookie.Name).(string)
// 	if !ok {
// 		http.Error(w, "no userID provided", http.StatusBadRequest)
// 		return
// 	}

// 	var entries []model.StorageEntry
// 	err := json.NewDecoder(r.Body).Decode(&entries)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}
// 	host := r.Host
// 	if u := s.Config.GetBaseURL(); len(u) != 0 {
// 		host = u
// 	}

// 	for i := range entries {
// 		entries[i].UserID = userID
// 		if err = entries[i].BuildShortURL(host); err != nil {
// 			http.Error(w, err.Error(), http.StatusBadRequest)
// 			return
// 		}
// 	}

// 	err = s.Storage.SetBatchURLs(r.Context(), entries)
// 	code := model.ParsePostError(err)
// 	if code == http.StatusBadRequest {
// 		http.Error(w, err.Error(), code)
// 		return
// 	}
// 	for i := range entries {
// 		entries[i].ID = ""
// 		entries[i].OriginalURL = ""
// 		entries[i].UserID = ""
// 	}

// 	value, err := json.Marshal(entries)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(code)
// 	w.Write(value)
// }

// func (s *Service) userURLsHandler(w http.ResponseWriter, r *http.Request) {
// 	userID, ok := r.Context().Value(cookie.Name).(string)
// 	if !ok {
// 		http.Error(w, "no userID provided", http.StatusBadRequest)
// 		return
// 	}

// 	urls, err := s.Storage.GetAllURLs(r.Context(), userID)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusNoContent)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(http.StatusOK)
// 	if err = json.NewEncoder(w).Encode(urls); err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}
// }

// func (s *Service) pingHandler(w http.ResponseWriter, r *http.Request) {
// 	err := s.Storage.PingContext(r.Context())
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	w.WriteHeader(http.StatusOK)
// }

// func (s *Service) asyncDeleteHandler(w http.ResponseWriter, r *http.Request) {
// 	userID, ok := r.Context().Value(cookie.Name).(string)
// 	if !ok {
// 		http.Error(w, "no userID provided", http.StatusBadRequest)
// 		return
// 	}

// 	ids := []string{}
// 	err := json.NewDecoder(r.Body).Decode(&ids)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	s.Storage.BatchDelete(model.DeleteRequest{UserID: userID, IDs: ids})

// 	w.WriteHeader(http.StatusAccepted)
// }
