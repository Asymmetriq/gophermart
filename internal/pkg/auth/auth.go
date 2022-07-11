package auth

import (
	"errors"
	"time"

	models "github.com/Asymmetriq/gophermart/internal/pkg/model"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

type UserClaims struct {
	UserID string `json:"user_id"`
	jwt.StandardClaims
}

func GenerateToken(user models.User, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(1 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func ValidateToken(signedToken, secret string) (string, error) {
	token, err := jwt.Parse(signedToken, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("couldn't parse claims")
	}
	if !token.Valid {
		return "", errors.New("token expired")
	}
	return claims["user_id"].(string), nil
}

func Authenticate(dbUser models.User, reqUser models.User) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(reqUser.Password)); err != nil {
		return false
	}
	return true

}
