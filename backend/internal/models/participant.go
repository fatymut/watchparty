package models

import "time"

// Participant représente un utilisateur qui fait partie d'une watch party.
// C'est la table qui relie User et WatchParty (relation many-to-many).
type Participant struct {
	ID           int64     `json:"id"`
	WatchPartyID int64     `json:"watchPartyId"`
	UserID       int64     `json:"userId"`
	Role         string    `json:"role"` // "creator" ou "member"
	JoinedAt     time.Time `json:"joinedAt"`

	// Champs pratiques pour le frontend, remplis via JOIN avec users
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
}
