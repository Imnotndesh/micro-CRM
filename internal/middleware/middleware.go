package middleware

import (
	"context"
	"micro-CRM/internal/models"
	"micro-CRM/internal/utils"
	"net/http"
	"strings"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.RespondError(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
		if tokenString == authHeader { // "Bearer " prefix not found
			utils.RespondError(w, http.StatusUnauthorized, "Bearer token required")
			return
		}

		userID, err := utils.ParseJWT(tokenString)
		if err != nil {
			utils.RespondError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Store user ID in request context for subsequent handlers
		ctx := context.WithValue(r.Context(), models.UserIDContextKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
