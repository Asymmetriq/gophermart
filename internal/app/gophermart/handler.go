package gophermart

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Asymmetriq/gophermart/internal/app/gophermart/middleware"
	"github.com/Asymmetriq/gophermart/internal/pkg/auth"
	"github.com/Asymmetriq/gophermart/internal/pkg/model"
	models "github.com/Asymmetriq/gophermart/internal/pkg/model"
	"github.com/google/uuid"
	"github.com/rs/xid"
	"golang.org/x/crypto/bcrypt"
)

func (s *Service) registerHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserStructKey).(models.User)
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
	user.TokenHash = xid.New().String()

	if err = s.Storage.SaveUser(r.Context(), user); err != nil {
		http.Error(w, err.Error(), models.GetErrorCode(err))
		return
	}

	token, err := auth.GenerateToken(user, s.Config.GetTokenSignKey())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("token", token)
	w.WriteHeader(http.StatusOK)
}

func (s *Service) loginHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(middleware.UserStructKey).(models.User)
	if !ok {
		http.Error(w, "no user data provided", http.StatusBadRequest)
		return
	}

	dbUser, err := s.Storage.GetUser(r.Context(), user)
	if err != nil {
		http.Error(w, err.Error(), models.GetErrorCode(err))
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
	w.Header().Set("token", token)
	w.WriteHeader(http.StatusOK)
}

func (s *Service) processOrderHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIdKey).(string)
	if !ok {
		http.Error(w, "no user id in token found", http.StatusUnauthorized)
		return
	}

	order := model.Order{
		Status: "new",
		UserID: userID,
	}
	err := json.NewDecoder(r.Body).Decode(&order.Number)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !Valid(order.Number) {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
	}

	if err = s.Storage.SaveOrder(r.Context(), order); err != nil {
		if errors.Is(err, model.ErrExistsForThisUser) {
			w.WriteHeader(http.StatusOK)
		} else {
			http.Error(w, err.Error(), models.GetErrorCode(err))
		}
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// func (s *Service) getHandler(w http.ResponseWriter, r *http.Request) {
// 	shortID := chi.URLParam(r, "id")

// 	ogURL, err := s.Storage.GetURL(r.Context(), shortID)
// 	code := models.ParseGetError(err)
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

// 	entry := models.NewStorageEntry(string(b), host, userID)
// 	err = s.Storage.SetURL(r.Context(), entry)
// 	code := models.ParsePostError(err)
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

// 	entry := models.NewStorageEntry(result.URL, host, userID)
// 	err = s.Storage.SetURL(r.Context(), entry)
// 	code := models.ParsePostError(err)
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

// 	var entries []models.StorageEntry
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
// 	code := models.ParsePostError(err)
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

// 	s.Storage.BatchDelete(models.DeleteRequest{UserID: userID, IDs: ids})

// 	w.WriteHeader(http.StatusAccepted)
// }
