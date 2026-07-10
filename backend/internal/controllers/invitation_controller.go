package controllers

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"watchparty/backend/internal/models"
)

// generateToken crée un token aléatoire (32 caractères hexadécimaux) pour une invitation.
// crypto/rand (et pas math/rand) car le token doit être imprévisible.
func generateToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// createInvitationInput : corps attendu pour créer une invitation.
type createInvitationInput struct {
	Email string `json:"email"`
}

// CreateInvitation invite un email à rejoindre une watch party (génère un token unique).
// POST /api/parties/{id}/invitations
func CreateInvitation(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partyID := r.PathValue("id")

		var in createInvitationInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}
		in.Email = strings.TrimSpace(strings.ToLower(in.Email))
		if in.Email == "" {
			WriteError(w, http.StatusBadRequest, "email est obligatoire")
			return
		}

		token, err := generateToken()
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "génération du token échouée")
			return
		}

		res, err := db.Exec(
			`INSERT INTO invitations (watch_party_id, invited_email, token, status)
			 VALUES (?, ?, ?, 'pending')`,
			partyID, in.Email, token,
		)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "création de l'invitation échouée")
			return
		}

		id, _ := res.LastInsertId()
		WriteJSON(w, http.StatusCreated, map[string]any{
			"id":           id,
			"invitedEmail": in.Email,
			"token":        token,
			"status":       "pending",
		})
	}
}

// acceptInvitationInput : corps attendu pour accepter une invitation.
type acceptInvitationInput struct {
	UserID int64 `json:"userId"`
}

// AcceptInvitation valide une invitation via son token et ajoute l'utilisateur comme participant.
// POST /api/invitations/{token}/accept
func AcceptInvitation(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.PathValue("token")

		var in acceptInvitationInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			WriteError(w, http.StatusBadRequest, "JSON invalide")
			return
		}
		if in.UserID == 0 {
			WriteError(w, http.StatusBadRequest, "userId est obligatoire")
			return
		}

		var inv models.Invitation
		err := db.QueryRow(
			`SELECT id, watch_party_id, invited_email, token, status, created_at
			 FROM invitations WHERE token = ?`,
			token,
		).Scan(&inv.ID, &inv.WatchPartyID, &inv.InvitedEmail, &inv.Token, &inv.Status, &inv.CreatedAt)

		if err == sql.ErrNoRows {
			WriteError(w, http.StatusNotFound, "invitation introuvable")
			return
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "erreur lecture invitation")
			return
		}
		if inv.Status != "pending" {
			WriteError(w, http.StatusConflict, "invitation déjà utilisée")
			return
		}

		if _, err := db.Exec(
			`INSERT INTO participants (watch_party_id, user_id, role) VALUES (?, ?, 'member')`,
			inv.WatchPartyID, in.UserID,
		); err != nil && !strings.Contains(err.Error(), "Duplicate") {
			WriteError(w, http.StatusInternalServerError, "ajout du participant échoué")
			return
		}

		if _, err := db.Exec(`UPDATE invitations SET status = 'accepted' WHERE id = ?`, inv.ID); err != nil {
			WriteError(w, http.StatusInternalServerError, "mise à jour de l'invitation échouée")
			return
		}

		WriteJSON(w, http.StatusOK, map[string]string{"message": "invitation acceptée, participant ajouté"})
	}
}
