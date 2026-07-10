package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"watchparty/backend/internal/models"
)

// GetMovies renvoie la liste de tous les films.
// GET /api/movies
func GetMovies(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(
			`SELECT id, title, genre, duration, release_year, synopsis, poster_url, created_at
			 FROM movies
			 ORDER BY id DESC`,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture films")
			return
		}
		defer rows.Close()

		movies := []models.Movie{}
		for rows.Next() {
			var m models.Movie
			if err := rows.Scan(
				&m.ID, &m.Title, &m.Genre, &m.Duration,
				&m.ReleaseYear, &m.Synopsis, &m.PosterURL, &m.CreatedAt,
			); err != nil {
				WriteError(w, http.StatusInternalServerError, "erreur scan film")
				return
			}
			movies = append(movies, m)
		}

		WriteJSON(w, http.StatusOK, movies)
	}
}

// CreateMovie ajoute un film.
// POST /api/movies
func CreateMovie(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m models.Movie
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}
		if m.Title == "" {
			WriteError(w, http.StatusBadRequest, "le titre est obligatoire")
			return
		}

		res, err := db.Exec(
			`INSERT INTO movies (title, genre, duration, release_year, synopsis, poster_url)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			m.Title, m.Genre, m.Duration, m.ReleaseYear, m.Synopsis, m.PosterURL,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "insertion film échouée")
			return
		}

		id, _ := res.LastInsertId()
		m.ID = id
		WriteJSON(w, http.StatusCreated, m)
	}
}

// UpdateMovie modifie un film existant (remplace tous les champs métier).
// PUT /api/movies/{id}
func UpdateMovie(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var m models.Movie
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}
		if m.Title == "" {
			WriteError(w, http.StatusBadRequest, "le titre est obligatoire")
			return
		}

		res, err := db.Exec(
			`UPDATE movies SET title = ?, genre = ?, duration = ?, release_year = ?, synopsis = ?, poster_url = ?
			 WHERE id = ?`,
			m.Title, m.Genre, m.Duration, m.ReleaseYear, m.Synopsis, m.PosterURL, id,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "modification du film échouée")
			return
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			WriteError(w, http.StatusNotFound, "film introuvable")
			return
		}

		WriteJSON(w, http.StatusOK, map[string]string{"message": "film modifié"})
	}
}

// DeleteMovie supprime un film. Les swipes/recommandations/notations liés sont
// supprimés en cascade (FK ON DELETE CASCADE) ; les parties qui l'avaient
// comme chosen_movie_id le perdent proprement (FK ON DELETE SET NULL).
// DELETE /api/movies/{id}
func DeleteMovie(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		res, err := db.Exec(`DELETE FROM movies WHERE id = ?`, id)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "suppression du film échouée")
			return
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			WriteError(w, http.StatusNotFound, "film introuvable")
			return
		}

		WriteJSON(w, http.StatusOK, map[string]string{"message": "film supprimé"})
	}
}
