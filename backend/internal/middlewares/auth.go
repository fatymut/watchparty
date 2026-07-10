package middlewares

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"watchparty/backend/internal/auth"
)

// userIDKey est la clé utilisée pour stocker l'id de l'utilisateur authentifié
// dans le contexte de la requête.
type contextKey string

const userIDKey contextKey = "userID"

// RequireAuth protège une route : elle exige un header "Authorization: Bearer <token>"
// valide, et injecte l'id de l'utilisateur dans le contexte de la requête.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeUnauthorized(w, "token manquant")
			return
		}

		tokenString := strings.TrimPrefix(header, "Bearer ")
		userID, err := auth.ValidateToken(tokenString)
		if err != nil {
			writeUnauthorized(w, "token invalide ou expiré")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

// UserIDFromContext récupère l'id de l'utilisateur authentifié, injecté par RequireAuth.
func UserIDFromContext(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
