package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"watchparty/backend/internal/models"
)

// scanParty lit une ligne watch_parties en gérant les colonnes NULL (date, creator_id).
func scanParty(scanner interface {
	Scan(dest ...any) error
}) (models.WatchParty, error) {
	var p models.WatchParty
	var date sql.NullTime
	var creatorID sql.NullInt64

	err := scanner.Scan(
		&p.ID, &p.Title, &p.Description, &date, &p.Status, &creatorID, &p.CreatedAt,
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
	return p, nil
}

// GetParties renvoie la liste des watch parties.
// GET /api/parties
func GetParties(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query(
			`SELECT id, title, description, date, status, creator_id, created_at
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
			`SELECT id, title, description, date, status, creator_id, created_at
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
		WriteJSON(w, http.StatusCreated, map[string]any{
			"id":          id,
			"title":       in.Title,
			"description": in.Description,
			"status":      "draft",
		})
	}
}
