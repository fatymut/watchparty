package models

import "time"

// Notation représente la note (1 à 5) qu'un participant donne au film choisi pour une watch party.
type Notation struct {
	ID           int64     `json:"id"`
	WatchPartyID int64     `json:"watchPartyId"`
	MovieID      int64     `json:"movieId"`
	UserID       int64     `json:"userId"`
	Rating       int       `json:"rating"` // 1 à 5
	CreatedAt    time.Time `json:"createdAt"`
}
