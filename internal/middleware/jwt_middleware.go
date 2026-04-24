package middleware

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret []byte

func init() {
	var err error
	jwtSecret, err = base64.StdEncoding.DecodeString("404E635266556A586E3272357538782F413F4428472B4B6250645367566B5970404E635266556A586E3272357538782F413F4428472B4B6250645367566B5970")
	if err != nil {
		log.Fatalf("failed to parse jwt secret: %v", err)
	}
}

func GetOwnerID(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	tokenStr := parts[1]

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		log.Println("Invalid token:", err)
		return ""
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ""
	}

	// 👇 adjust depending on your auth service
	if userID, ok := claims["userId"].(string); ok {
		log.Println("Owner ID:", userID)
		return userID
	}

	// fallback (common JWT field)
	if userID, ok := claims["sub"].(string); ok {
		log.Println("Owner ID (sub):", userID)
		return userID
	}

	return ""
}