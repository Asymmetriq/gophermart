package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Asymmetriq/gophermart/internal/pkg/auth"
	"github.com/Asymmetriq/gophermart/internal/pkg/model"
)

type ContextKey string

const (
	userStructKey ContextKey = "userStruct"
	userIDKey     ContextKey = "userID"
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

		ctx := context.WithValue(r.Context(), userStructKey, user)
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

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUser(ctx context.Context) (model.User, error) {
	user, ok := ctx.Value(userStructKey).(model.User)
	if !ok {
		return model.User{}, errors.New("no user data provided")
	}
	return user, nil
}

func GetUserID(ctx context.Context) (string, error) {
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return "", errors.New("no user id in token foun")
	}
	return userID, nil
}
