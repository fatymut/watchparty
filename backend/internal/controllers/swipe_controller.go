package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"watchparty/backend/internal/models"
)

// createSwipeInput : corps attendu pour enregistrer un swipe.
// value = "like" (swipe droite) ou "dislike" (swipe gauche).
type createSwipeInput struct {
	UserID  int64  `json:"userId"`
	MovieID int64  `json:"movieId"`
	Value   string `json:"value"`
}

// CreateSwipe enregistre (ou met à jour) le swipe d'un utilisateur sur un film,
// pour une watch party donnée.
// POST /api/parties/{id}/swipes
func CreateSwipe(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		var in createSwipeInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}
		if in.UserID == 0 || in.MovieID == 0 {
			WriteError(w, http.StatusBadRequest, "userId et movieId sont obligatoires")
			return
		}
		if in.Value != "like" && in.Value != "dislike" {
			WriteError(w, http.StatusBadRequest, `value doit être "like" ou "dislike"`)
			return
		}

		// Grâce à la contrainte UNIQUE (user_id, movie_id, watch_party_id),
		// re-swiper le même film met à jour le choix au lieu de créer un doublon.
		_, err := db.Exec(
			`INSERT INTO swipes (user_id, movie_id, watch_party_id, value)
			 VALUES (?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE value = VALUES(value), swiped_at = CURRENT_TIMESTAMP`,
			in.UserID, in.MovieID, partyID, in.Value,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "enregistrement du swipe échoué")
			return
		}

		WriteJSON(w, http.StatusCreated, map[string]string{"message": "swipe enregistré"})
	}
}

// GetSwipes renvoie tous les swipes d'une watch party.
// GET /api/parties/{id}/swipes
func GetSwipes(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		rows, err := db.Query(
			`SELECT id, user_id, movie_id, watch_party_id, value, swiped_at
			 FROM swipes
			 WHERE watch_party_id = ?
			 ORDER BY swiped_at DESC`,
			partyID,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture swipes")
			return
		}
		defer rows.Close()

		swipes := []models.Swipe{}
		for rows.Next() {
			var s models.Swipe
			if err := rows.Scan(
				&s.ID, &s.UserID, &s.MovieID, &s.WatchPartyID, &s.Value, &s.SwipedAt,
			); err != nil {
				WriteError(w, http.StatusInternalServerError, "erreur scan swipe")
				return
			}
			swipes = append(swipes, s)
		}

		WriteJSON(w, http.StatusOK, swipes)
	}
}
