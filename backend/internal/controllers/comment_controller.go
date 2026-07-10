package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"watchparty/backend/internal/models"
)

// createCommentInput : corps attendu pour ajouter un commentaire.
type createCommentInput struct {
	UserID  int64  `json:"userId"`
	Content string `json:"content"`
}

// CreateComment ajoute un commentaire sur une watch party.
// POST /api/parties/{id}/comments
func CreateComment(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		var in createCommentInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}
		if in.UserID == 0 || in.Content == "" {
			WriteError(w, http.StatusBadRequest, "userId et content sont obligatoires")
			return
		}

		res, err := db.Exec(
			`INSERT INTO comments (watch_party_id, user_id, content) VALUES (?, ?, ?)`,
			partyID, in.UserID, in.Content,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "ajout du commentaire échoué")
			return
		}

		id, _ := res.LastInsertId()
		WriteJSON(w, http.StatusCreated, map[string]any{
			"id":      id,
			"userId":  in.UserID,
			"content": in.Content,
		})
	}
}

// GetComments renvoie les commentaires d'une watch party, du plus récent au plus ancien.
// GET /api/parties/{id}/comments
func GetComments(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		rows, err := db.Query(
			`SELECT c.id, c.watch_party_id, c.user_id, c.content, c.created_at, u.username
			 FROM comments c
			 JOIN users u ON u.id = c.user_id
			 WHERE c.watch_party_id = ?
			 ORDER BY c.created_at DESC`,
			partyID,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture commentaires")
			return
		}
		defer rows.Close()

		comments := []models.Comment{}
		for rows.Next() {
			var c models.Comment
			if err := rows.Scan(
				&c.ID, &c.WatchPartyID, &c.UserID, &c.Content, &c.CreatedAt, &c.Username,
			); err != nil {
				WriteError(w, http.StatusInternalServerError, "erreur scan commentaire")
				return
			}
			comments = append(comments, c)
		}

		WriteJSON(w, http.StatusOK, comments)
	}
}
