package models

import "time"

// Comment représente un commentaire laissé par un utilisateur sur une watch party.
type Comment struct {
	ID           int64     `json:"id"`
	WatchPartyID int64     `json:"watchPartyId"`
	UserID       int64     `json:"userId"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"createdAt"`

	// Champ pratique pour le frontend, rempli via JOIN avec users
	Username string `json:"username,omitempty"`
}
