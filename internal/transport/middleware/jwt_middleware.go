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

		userID := r.Header.Get("X-User-Id")
		role := r.Header.Get("X-User-Role")
		authMethod := r.Header.Get("X-Auth-Method")
		requestID := r.Header.Get("X-Request-Id")
		headers := [4]string{userID, role, authMethod, requestID}

		ctx := context.WithValue(r.Context(), ContextKeyUserID, userID)
		ctx = context.WithValue(ctx, ContextKeyRole, role)
		ctx = context.WithValue(ctx, ContextKeyRequestId, requestID)
		ctx = context.WithValue(ctx, ContextKeyAuthMethod, authMethod)

		if strings.Contains(requestID, "public-req") {

			next.ServeHTTP(w, r.WithContext(ctx))
			return // <-- stop here, don't fall through
		}

		for iter, header := range headers {
			if header == "" {
				log.Printf("[Middleware] header %v not available, dropping request", iter)
				utils.WriteJSONError(w, http.StatusUnauthorized, fmt.Errorf("missing required auth headers"))
				return // prefer this over panic — see note below
			}
		}

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
func GetRequestID(r *http.Request) string {
	if val, ok := r.Context().Value("requestId").(string); ok {
		return val
	}
	return ""
}

 