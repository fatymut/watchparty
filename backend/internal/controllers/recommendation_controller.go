package controllers

import (
	"database/sql"
	"net/http"
	"strconv"

	"watchparty/backend/internal/models"
)

// GenerateRecommendation calcule le film ayant reçu le plus de "like" pour une
// watch party, puis enregistre le résultat dans la table recommendations.
// POST /api/parties/{id}/recommendation/generate
func GenerateRecommendation(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		// Cœur de l'algorithme : on compte les likes par film, on garde le meilleur.
		var movieID int64
		var likes int
		err := db.QueryRow(
			`SELECT movie_id, COUNT(*) AS likes
			 FROM swipes
			 WHERE watch_party_id = ? AND value = 'like'
			 GROUP BY movie_id
			 ORDER BY likes DESC, movie_id ASC
			 LIMIT 1`,
			partyID,
		).Scan(&movieID, &likes)

		if err == sql.ErrNoRows {
			WriteError(w, http.StatusNotFound, "aucun like pour cette party, impossible de recommander")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur calcul recommandation")
			return
		}

		// On enregistre la recommandation (historique)
		res, err := db.Exec(
			`INSERT INTO recommendations (watch_party_id, movie_id, likes_count)
			 VALUES (?, ?, ?)`,
			partyID, movieID, likes,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "enregistrement recommandation échoué")
			return
		}
		id, _ := res.LastInsertId()
		partyIDInt, _ := strconv.ParseInt(partyID, 10, 64)

		WriteJSON(w, http.StatusOK, models.Recommendation{
			ID:           id,
			WatchPartyID: partyIDInt,
			MovieID:      movieID,
			LikesCount:   likes,
		})
	}
}

// GetRecommendation renvoie la dernière recommandation générée pour une party,
// avec le film complet (JOIN movies).
// GET /api/parties/{id}/recommendation
func GetRecommendation(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		var rec models.Recommendation
		var m models.Movie
		err := db.QueryRow(
			`SELECT r.id, r.watch_party_id, r.movie_id, r.likes_count, r.created_at,
			        m.id, m.title, m.genre, m.duration, m.release_year, m.synopsis, m.poster_url, m.created_at
			 FROM recommendations r
			 JOIN movies m ON m.id = r.movie_id
			 WHERE r.watch_party_id = ?
			 ORDER BY r.created_at DESC, r.id DESC
			 LIMIT 1`,
			partyID,
		).Scan(
			&rec.ID, &rec.WatchPartyID, &rec.MovieID, &rec.LikesCount, &rec.CreatedAt,
			&m.ID, &m.Title, &m.Genre, &m.Duration, &m.ReleaseYear, &m.Synopsis, &m.PosterURL, &m.CreatedAt,
		)

		if err == sql.ErrNoRows {
			WriteError(w, http.StatusNotFound, "aucune recommandation pour cette party")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture recommandation")
			return
		}

		rec.Movie = &m
		WriteJSON(w, http.StatusOK, rec)
	}
}
