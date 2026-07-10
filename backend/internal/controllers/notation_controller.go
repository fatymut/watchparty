package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"watchparty/backend/internal/models"
)

// createNotationInput : corps attendu pour noter le film choisi d'une party.
type createNotationInput struct {
	UserID int64 `json:"userId"`
	Rating int   `json:"rating"` // 1 à 5
}

// CreateNotation enregistre (ou met à jour) la note d'un participant sur le film
// choisi pour la watch party. Fonctionnalité bonus, en plus de la recommandation
// automatique (voir recommendation_controller.go). Le film noté est toujours celui
// de watch_parties.chosen_movie_id, renseigné soit par GenerateRecommendation
// (automatique, via les likes), soit manuellement via ChooseMovie.
// POST /api/parties/{id}/notations
func CreateNotation(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		var in createNotationInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}
		if in.UserID == 0 {
			WriteError(w, http.StatusBadRequest, "userId est obligatoire")
			return
		}
		if in.Rating < 1 || in.Rating > 5 {
			WriteError(w, http.StatusBadRequest, "rating doit être compris entre 1 et 5")
			return
		}

		var chosenMovieID sql.NullInt64
		err := db.QueryRow(`SELECT chosen_movie_id FROM watch_parties WHERE id = ?`, partyID).Scan(&chosenMovieID)
		if err == sql.ErrNoRows {
			WriteError(w, http.StatusNotFound, "party introuvable")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture party")
			return
		}
		if !chosenMovieID.Valid {
			WriteError(w, http.StatusBadRequest, "aucun film choisi pour cette party, impossible de noter")
			return
		}

		// Contrainte UNIQUE (watch_party_id, user_id) -> un re-vote met à jour la note.
		_, err = db.Exec(
			`INSERT INTO notations (watch_party_id, movie_id, user_id, rating)
			 VALUES (?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE rating = VALUES(rating), movie_id = VALUES(movie_id)`,
			partyID, chosenMovieID.Int64, in.UserID, in.Rating,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "enregistrement de la note échoué")
			return
		}

		WriteJSON(w, http.StatusCreated, map[string]string{"message": "note enregistrée"})
	}
}

// notationsResponse regroupe les notes individuelles et la moyenne du film choisi.
type notationsResponse struct {
	MovieID   int64             `json:"movieId,omitempty"`
	Average   float64           `json:"average"`
	Count     int               `json:"count"`
	Notations []models.Notation `json:"notations"`
}

// GetNotations renvoie toutes les notes d'une party ainsi que la moyenne du film choisi.
// GET /api/parties/{id}/notations
func GetNotations(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		rows, err := db.Query(
			`SELECT id, watch_party_id, movie_id, user_id, rating, created_at
			 FROM notations
			 WHERE watch_party_id = ?
			 ORDER BY created_at DESC`,
			partyID,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture notations")
			return
		}
		defer rows.Close()

		resp := notationsResponse{Notations: []models.Notation{}}
		var total int
		for rows.Next() {
			var n models.Notation
			if err := rows.Scan(&n.ID, &n.WatchPartyID, &n.MovieID, &n.UserID, &n.Rating, &n.CreatedAt); err != nil {
				WriteError(w, http.StatusInternalServerError, "erreur scan notation")
				return
			}
			resp.Notations = append(resp.Notations, n)
			resp.MovieID = n.MovieID
			total += n.Rating
			resp.Count++
		}
		if resp.Count > 0 {
			resp.Average = float64(total) / float64(resp.Count)
		}

		WriteJSON(w, http.StatusOK, resp)
	}
}
