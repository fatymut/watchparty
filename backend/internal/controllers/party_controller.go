package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"watchparty/backend/internal/middlewares"
	"watchparty/backend/internal/models"
)

// scanParty lit une ligne watch_parties en gérant les colonnes NULL (date, creator_id, chosen_movie_id).
func scanParty(scanner interface {
	Scan(dest ...any) error
}) (models.WatchParty, error) {
	var p models.WatchParty
	var date sql.NullTime
	var creatorID sql.NullInt64
	var chosenMovieID sql.NullInt64

	err := scanner.Scan(
		&p.ID, &p.Title, &p.Description, &date, &p.Status, &creatorID, &chosenMovieID, &p.CreatedAt,
	)
	if err != nil {
		return p, err
	}
	if date.Valid {
		p.Date = date.Time
	}
	if creatorID.Valid {
		p.CreatorID = creatorID.Int64
	}
	if chosenMovieID.Valid {
		p.ChosenMovieID = chosenMovieID.Int64
	}
	return p, nil
}

// GetParties renvoie la liste des watch parties.
// GET /api/parties
func GetParties(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(
			`SELECT id, title, description, date, status, creator_id, chosen_movie_id, created_at
			 FROM watch_parties
			 ORDER BY id DESC`,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture parties")
			return
		}
		defer rows.Close()

		parties := []models.WatchParty{}
		for rows.Next() {
			p, err := scanParty(rows)
			if err != nil {
				WriteError(w, http.StatusInternalServerError, "erreur scan party")
				return
			}
			parties = append(parties, p)
		}

		WriteJSON(w, http.StatusOK, parties)
	}
}

// GetPartyByID renvoie le détail d'une party.
// GET /api/parties/{id}
func GetPartyByID(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		row := db.QueryRow(
			`SELECT id, title, description, date, status, creator_id, chosen_movie_id, created_at
			 FROM watch_parties
			 WHERE id = ?`,
			id,
		)
		p, err := scanParty(row)
		if err == sql.ErrNoRows {
			WriteError(w, http.StatusNotFound, "party introuvable")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture party")
			return
		}

		WriteJSON(w, http.StatusOK, p)
	}
}

// createPartyInput représente le corps attendu lors de la création.
type createPartyInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Date        string `json:"date"` // format ISO optionnel, ex: "2026-07-01T20:00:00Z"
	CreatorID   int64  `json:"creatorId"`
}

// CreateParty crée une nouvelle watch party.
// POST /api/parties
func CreateParty(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var in createPartyInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}
		if in.Title == "" {
			WriteError(w, http.StatusBadRequest, "le titre est obligatoire")
			return
		}

		// Date optionnelle
		var date sql.NullTime
		if in.Date != "" {
			t, err := time.Parse(time.RFC3339, in.Date)
			if err != nil {
				WriteError(w, http.StatusBadRequest, "date invalide (format attendu RFC3339)")
				return
			}
			date = sql.NullTime{Time: t, Valid: true}
		}

		// creator_id optionnel
		var creator sql.NullInt64
		if in.CreatorID != 0 {
			creator = sql.NullInt64{Int64: in.CreatorID, Valid: true}
		}

		res, err := db.Exec(
			`INSERT INTO watch_parties (title, description, date, status, creator_id)
			 VALUES (?, ?, ?, 'draft', ?)`,
			in.Title, in.Description, date, creator,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "création party échouée")
			return
		}

		id, _ := res.LastInsertId()

		// Le créateur devient automatiquement participant de sa propre party.
		if in.CreatorID != 0 {
			db.Exec(
				`INSERT INTO participants (watch_party_id, user_id, role) VALUES (?, ?, 'creator')`,
				id, in.CreatorID,
			)
		}

		WriteJSON(w, http.StatusCreated, map[string]any{
			"id":          id,
			"title":       in.Title,
			"description": in.Description,
			"status":      "draft",
		})
	}
}

// updatePartyInput représente le corps attendu lors de la modification.
type updatePartyInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Date        string `json:"date"` // format ISO optionnel, ex: "2026-07-01T20:00:00Z"
}

// UpdateParty modifie le titre, la description et la date d'une watch party.
// PUT /api/parties/{id}
func UpdateParty(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var in updatePartyInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}
		if in.Title == "" {
			WriteError(w, http.StatusBadRequest, "le titre est obligatoire")
			return
		}

		var date sql.NullTime
		if in.Date != "" {
			t, err := time.Parse(time.RFC3339, in.Date)
			if err != nil {
				WriteError(w, http.StatusBadRequest, "date invalide (format attendu RFC3339)")
				return
			}
			date = sql.NullTime{Time: t, Valid: true}
		}

		res, err := db.Exec(
			`UPDATE watch_parties SET title = ?, description = ?, date = ? WHERE id = ?`,
			in.Title, in.Description, date, id,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "modification de la party échouée")
			return
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			WriteError(w, http.StatusNotFound, "party introuvable")
			return
		}

		WriteJSON(w, http.StatusOK, map[string]string{"message": "party modifiée"})
	}
}

// DeleteParty supprime une watch party. Participants, invitations, swipes, notations,
// recommandations et commentaires liés sont supprimés en cascade (FK ON DELETE CASCADE).
// DELETE /api/parties/{id}
func DeleteParty(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		res, err := db.Exec(`DELETE FROM watch_parties WHERE id = ?`, id)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "suppression de la party échouée")
			return
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			WriteError(w, http.StatusNotFound, "party introuvable")
			return
		}

		WriteJSON(w, http.StatusOK, map[string]string{"message": "party supprimée"})
	}
}

// CloseParty ferme une watch party (passe son status à "closed").
// Une party fermée n'est plus modifiable : plus de nouveaux swipes/participants/invitations attendus côté frontend.
// POST /api/parties/{id}/close
func CloseParty(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var status string
		err := db.QueryRow(`SELECT status FROM watch_parties WHERE id = ?`, id).Scan(&status)
		if err == sql.ErrNoRows {
			WriteError(w, http.StatusNotFound, "party introuvable")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture party")
			return
		}
		if status == "closed" {
			WriteError(w, http.StatusConflict, "cette party est déjà fermée")
			return
		}

		if _, err := db.Exec(`UPDATE watch_parties SET status = 'closed' WHERE id = ?`, id); err != nil {
			WriteError(w, http.StatusInternalServerError, "fermeture de la party échouée")
			return
		}

		WriteJSON(w, http.StatusOK, map[string]string{"message": "party fermée", "status": "closed"})
	}
}

// StartSwipeSession lance la session de swipe d'une watch party (status -> "active").
// Réservé au créateur de la party : l'utilisateur est déduit du token JWT
// (middlewares.RequireAuth), pas d'un champ envoyé par le client.
// POST /api/parties/{id}/start-swipe (route protégée)
func StartSwipeSession(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		userID, ok := middlewares.UserIDFromContext(r.Context())
		if !ok {
			WriteError(w, http.StatusUnauthorized, "non authentifié")
			return
		}

		row := db.QueryRow(
			`SELECT id, title, description, date, status, creator_id, chosen_movie_id, created_at
			 FROM watch_parties
			 WHERE id = ?`,
			partyID,
		)
		p, err := scanParty(row)
		if err == sql.ErrNoRows {
			WriteError(w, http.StatusNotFound, "party introuvable")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture party")
			return
		}

		if p.CreatorID != userID {
			WriteError(w, http.StatusForbidden, "seul le créateur peut lancer la session de swipe")
			return
		}
		if p.Status == "closed" {
			WriteError(w, http.StatusConflict, "cette party est fermée, impossible de lancer le swipe")
			return
		}

		if _, err := db.Exec(`UPDATE watch_parties SET status = 'active' WHERE id = ?`, partyID); err != nil {
			WriteError(w, http.StatusInternalServerError, "lancement de la session échoué")
			return
		}

		WriteJSON(w, http.StatusOK, map[string]string{"message": "session de swipe lancée", "status": "active"})
	}
}

// chooseMovieInput : corps attendu pour désigner le film choisi d'une party.
type chooseMovieInput struct {
	MovieID int64 `json:"movieId"`
}

// ChooseMovie désigne le film choisi pour la watch party (celui qui sera regardé et noté).
// POST /api/parties/{id}/choose-movie
func ChooseMovie(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		var in chooseMovieInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}
		if in.MovieID == 0 {
			WriteError(w, http.StatusBadRequest, "movieId est obligatoire")
			return
		}

		res, err := db.Exec(
			`UPDATE watch_parties SET chosen_movie_id = ? WHERE id = ?`,
			in.MovieID, partyID,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "sélection du film échouée")
			return
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			WriteError(w, http.StatusNotFound, "party introuvable")
			return
		}

		WriteJSON(w, http.StatusOK, map[string]any{"message": "film choisi enregistré", "movieId": in.MovieID})
	}
}
