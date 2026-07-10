package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"watchparty/backend/internal/models"
)

// createParticipantInput : corps attendu pour ajouter un participant.
type createParticipantInput struct {
	UserID int64 `json:"userId"`
}

// CreateParticipant ajoute un utilisateur comme participant d'une watch party.
// POST /api/parties/{id}/participants
func CreateParticipant(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		var in createParticipantInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}
		if in.UserID == 0 {
			WriteError(w, http.StatusBadRequest, "userId est obligatoire")
			return
		}

		_, err := db.Exec(
			`INSERT INTO participants (watch_party_id, user_id, role) VALUES (?, ?, 'member')`,
			partyID, in.UserID,
		)
		if err != nil {
			// contrainte UNIQUE(watch_party_id, user_id) -> déjà participant
			if strings.Contains(err.Error(), "Duplicate") {
				WriteError(w, http.StatusConflict, "cet utilisateur participe déjà à cette party")
				return
			}
			WriteError(w, http.StatusInternalServerError, "ajout du participant échoué")
			return
		}

		WriteJSON(w, http.StatusCreated, map[string]string{"message": "participant ajouté"})
	}
}

// GetParticipants renvoie la liste des participants d'une watch party, avec leurs infos user.
// GET /api/parties/{id}/participants
func GetParticipants(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		rows, err := db.Query(
			`SELECT p.id, p.watch_party_id, p.user_id, p.role, p.joined_at, u.username, u.email
			 FROM participants p
			 JOIN users u ON u.id = p.user_id
			 WHERE p.watch_party_id = ?
			 ORDER BY p.joined_at ASC`,
			partyID,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture participants")
			return
		}
		defer rows.Close()

		participants := []models.Participant{}
		for rows.Next() {
			var p models.Participant
			if err := rows.Scan(
				&p.ID, &p.WatchPartyID, &p.UserID, &p.Role, &p.JoinedAt, &p.Username, &p.Email,
			); err != nil {
				WriteError(w, http.StatusInternalServerError, "erreur scan participant")
				return
			}
			participants = append(participants, p)
		}

		WriteJSON(w, http.StatusOK, participants)
	}
}
