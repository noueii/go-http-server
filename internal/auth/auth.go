package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	if err != nil {
		return "", err
	}

	return string(hashed), nil
}

func CheckPasswordHash(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))

	return err

}

func MakeJWT(userID uuid.UUID, tokenSecret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1).UTC()),
		Subject:   userID.String(),
	})

	signed, err := token.SignedString([]byte(tokenSecret))

	if err != nil {
		return "", err
	}

	return signed, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil

	})

	if err != nil {
		return uuid.Nil, err
	}

	val, err := token.Claims.GetSubject()

	if err != nil {
		return uuid.Nil, err
	}

	res, err := uuid.Parse(val)

	if err != nil {
		return uuid.Nil, err
	}

	return res, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	str := headers.Get("Authorization")

	token := strings.TrimSpace(strings.Replace(str, "Bearer", "", 1))

	if token == "" {
		return "", fmt.Errorf("Token not provided")
	}

	return token, nil
}

func MakeRefreshToken() (string, error) {
	random := make([]byte, 32)
	_, err := rand.Read(random)

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(random), err

}

func GetApiKey(header http.Header) (string, error) {
	str := header.Get("Authorization")

	key := strings.TrimSpace(strings.Replace(str, "ApiKey", "", 1))

	if key == "" {
		return "", fmt.Errorf("Key not found")
	}

	return key, nil
}
