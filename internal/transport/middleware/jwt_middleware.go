package middleware

import (
	"context"
	"log"
	"net/http"
)
type contextKey string

const (
	ContextKeyUserID     contextKey = "userId"
	ContextKeyRole       contextKey = "role"
	ContextKeyRequestId  contextKey = "requestId"
	ContextKeyAuthMethod contextKey = "authMethod"
)


func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		userID := r.Header.Get("X-User-Id")
		role := r.Header.Get("X-User-Role")
		authMethod := r.Header.Get("X-Auth-Method")
		requestID := r.Header.Get("X-Request-Id")

		if userID == "" || role == "" || authMethod == "" || requestID == "" {
			log.Printf("[Middleware] missing required headers, dropping request")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Inject claims into request context for downstream handlers.
		ctx := context.WithValue(r.Context(), "userId", userID)
		ctx = context.WithValue(ctx, "role", role)
		ctx = context.WithValue(ctx, "requestId", requestID)
		ctx = context.WithValue(ctx, "authMethod", authMethod)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetOwnerID extracts the user ID from the request context.
func GetOwnerID(r *http.Request) string {
	if val, ok := r.Context().Value("userId").(string); ok {
		return val
	}
	return ""
}
