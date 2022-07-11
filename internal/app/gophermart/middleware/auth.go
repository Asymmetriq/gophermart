package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Asymmetriq/gophermart/internal/pkg/auth"
	"github.com/Asymmetriq/gophermart/internal/pkg/model"
)

type ContextKey string

const (
	UserStructKey ContextKey = "userStruct"
	UserIDKey     ContextKey = "userID"
)

// UserValidation validates the user in the request
func UserValidation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		var user model.User
		if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// validate the user
		if err := user.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), UserStructKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TokenValidation(secret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			token := r.Header.Get("Authorization")
			if len(token) == 0 {
				http.Error(w, "no auth token", http.StatusUnauthorized)
				return
			}
			splitted := strings.Split(token, "Bearer")
			if len(splitted) != 2 {
				http.Error(w, "wrong token format", http.StatusUnauthorized)
				return
			}
			// validate the token
			token = strings.TrimSpace(splitted[1])
			userID, err := auth.ValidateToken(token, secret)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
