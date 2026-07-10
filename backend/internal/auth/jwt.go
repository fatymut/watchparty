package auth

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// secret vient de la variable d'environnement JWT_SECRET, avec une valeur par
// défaut pour le développement local (à changer en production).
func secret() []byte {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s)
	}
	return []byte("watchparty_dev_secret_change_me")
}

// GenerateToken crée un JWT signé (HMAC-SHA256) contenant l'id de l'utilisateur,
// valable 24h.
func GenerateToken(userID int64) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret())
}

// ValidateToken vérifie la signature et l'expiration du token, et renvoie l'id
// de l'utilisateur contenu dans le claim "sub".
func ValidateToken(tokenString string) (int64, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("méthode de signature inattendue")
		}
		return secret(), nil
	})
	if err != nil || !token.Valid {
		return 0, errors.New("token invalide ou expiré")
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok {
		return 0, errors.New("claims invalides")
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, errors.New("subject invalide")
	}
	return userID, nil
}
