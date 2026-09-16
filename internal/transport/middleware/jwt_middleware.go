package middleware

import (
	"context"
	"fmt"
	"llm-inference-service/internal/utils"
	"log"
	"net/http"
	"strings"
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

		// Extract authentication headers
		userID := r.Header.Get("X-User-Id")
		role := r.Header.Get("X-User-Role")
		authMethod := r.Header.Get("X-Auth-Method")
		requestID := r.Header.Get("X-Request-Id")

		log.Printf("[Middleware] userID: %s", userID)
		log.Printf("[Middleware] role: %s", role)
		log.Printf("[Middleware] authMethod: %s", authMethod)
		log.Printf("[Middleware] requestID: %s", requestID)
		log.Printf("[Middleware] url: %s", r.URL)

		// Build the context.
		// IMPORTANT: each value is added to the previous ctx,
		// not r.Context(), otherwise previously added values are lost.
		ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
		ctx = context.WithValue(ctx, ContextKeyRole, role)
		ctx = context.WithValue(ctx, ContextKeyRequestId, requestID)
		ctx = context.WithValue(ctx, ContextKeyAuthMethod, authMethod)

		// Public requests don't require authentication headers.
		if strings.Contains(requestID, "public-req") {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}



		// Validate required headers.
		headers := map[string]string{
			"X-User-Id":      userID,
			"X-User-Role":    role,
			"X-Auth-Method":  authMethod,
			"X-Request-Id":   requestID,
		}

		for name, value := range headers {
			if value == "" {
				log.Printf(
					"[Middleware] required header %s not available, dropping request",
					name,
				)

				utils.WriteJSONError(
					w,
					http.StatusUnauthorized,
					fmt.Errorf("missing required auth headers"),
				)
				return
			}
		}

		// Pass the request with the populated context to the next handler.
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetOwnerID extracts the user ID from the request context.
func GetOwnerID(r *http.Request) string {
	if val, ok := r.Context().Value(ContextKeyUserID).(string); ok {
		return val
	}

	return ""
}

// GetRequestID extracts the request ID from the request context.
func GetRequestID(r *http.Request) string {
	if val, ok := r.Context().Value(ContextKeyRequestId).(string); ok {
		return val
	}

	return ""
}

// GetRole extracts the user role from the request context.
func GetRole(r *http.Request) string {
	if val, ok := r.Context().Value(ContextKeyRole).(string); ok {
		return val
	}

	return ""
}

// GetAuthMethod extracts the authentication method from the request context.
func GetAuthMethod(r *http.Request) string {
	if val, ok := r.Context().Value(ContextKeyAuthMethod).(string); ok {
		return val
	}

	return ""
}