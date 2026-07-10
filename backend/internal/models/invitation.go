package models

import "time"

// Invitation représente une invitation envoyée à un email pour rejoindre une watch party.
// Le token permet de créer un lien unique du type /invite/{token} côté frontend.
type Invitation struct {
	ID           int64     `json:"id"`
	WatchPartyID int64     `json:"watchPartyId"`
	InvitedEmail string    `json:"invitedEmail"`
	Token        string    `json:"token"`
	Status       string    `json:"status"` // "pending" ou "accepted"
	CreatedAt    time.Time `json:"createdAt"`
}
