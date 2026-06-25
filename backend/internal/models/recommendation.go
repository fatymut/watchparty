package models

import "time"

type Recommendation struct {
	ID           int64     `json:"id"`
	WatchPartyID int64     `json:"watchPartyId"`
	MovieID      int64     `json:"movieId"`
	LikesCount   int       `json:"likesCount"`
	CreatedAt    time.Time `json:"createdAt"`

	// Champ pratique pour le frontend : le film recommandé complet (rempli via JOIN)
	Movie *Movie `json:"movie,omitempty"`
}
