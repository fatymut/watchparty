package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"watchparty/backend/internal/auth"
	"watchparty/backend/internal/middlewares"
	"watchparty/backend/internal/models"
)

// registerInput : corps attendu pour l'inscription.
type registerInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginInput : corps attendu pour la connexion.
type loginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// authResponse embarque l'utilisateur (mêmes champs qu'avant) + un token JWT.
// L'embedding garde le format de réponse plat, pour ne rien casser côté frontend.
type authResponse struct {
	models.User
	Token string `json:"token"`
}

// Register crée un compte utilisateur (mot de passe hashé avec bcrypt).
// POST /api/register
func Register(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in registerInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}

		in.Username = strings.TrimSpace(in.Username)
		in.Email = strings.TrimSpace(strings.ToLower(in.Email))
		if in.Username == "" || in.Email == "" || in.Password == "" {
			WriteError(w, http.StatusBadRequest, "username, email et password sont obligatoires")
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur de hachage du mot de passe")
			return
		}

		res, err := db.Exec(
			`INSERT INTO users (username, email, password_hash) VALUES (?, ?, ?)`,
			in.Username, in.Email, string(hash),
		)
		if err != nil {
			// email UNIQUE -> doublon probable
			if strings.Contains(err.Error(), "Duplicate") {
				WriteError(w, http.StatusConflict, "cet email est déjà utilisé")
				return
			}
			WriteError(w, http.StatusInternalServerError, "création du compte échouée")
			return
		}

		id, _ := res.LastInsertId()

		token, err := auth.GenerateToken(id)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "génération du token échouée")
			return
		}

		WriteJSON(w, http.StatusCreated, authResponse{
			User: models.User{
				ID:       id,
				Username: in.Username,
				Email:    in.Email,
			},
			Token: token,
		})
	}
}

// Login vérifie l'email + mot de passe et renvoie l'utilisateur.
// POST /api/login
func Login(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in loginInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}
		in.Email = strings.TrimSpace(strings.ToLower(in.Email))

		var u models.User
		var hash string
		err := db.QueryRow(
			`SELECT id, username, email, password_hash, created_at FROM users WHERE email = ?`,
			in.Email,
		).Scan(&u.ID, &u.Username, &u.Email, &hash, &u.CreatedAt)

		if err == sql.ErrNoRows {
			WriteError(w, http.StatusUnauthorized, "email ou mot de passe incorrect")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur de connexion")
			return
		}

		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
			WriteError(w, http.StatusUnauthorized, "email ou mot de passe incorrect")
			return
		}

		token, err := auth.GenerateToken(u.ID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "génération du token échouée")
			return
		}

		WriteJSON(w, http.StatusOK, authResponse{User: u, Token: token})
	}
}

// GetUsers renvoie la liste des utilisateurs (pratique pour le front : choisir qui swipe).
// GET /api/users
func GetUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(`SELECT id, username, email, created_at FROM users ORDER BY id`)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture utilisateurs")
			return
		}
		defer rows.Close()

		users := []models.User{}
		for rows.Next() {
			var u models.User
			if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt); err != nil {
				WriteError(w, http.StatusInternalServerError, "erreur scan utilisateur")
				return
			}
			users = append(users, u)
		}

		WriteJSON(w, http.StatusOK, users)
	}
}

// GetMe renvoie l'utilisateur authentifié (déduit du token JWT par middlewares.RequireAuth).
// GET /api/me (route protégée)
func GetMe(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := middlewares.UserIDFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusUnauthorized, "non authentifié")
			return
		}

		var u models.User
		err := db.QueryRow(
			`SELECT id, username, email, created_at FROM users WHERE id = ?`,
			userID,
		).Scan(&u.ID, &u.Username, &u.Email, &u.CreatedAt)

		if err == sql.ErrNoRows {
			WriteError(w, http.StatusNotFound, "utilisateur introuvable")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture utilisateur")
			return
		}

		WriteJSON(w, http.StatusOK, u)
	}
}
