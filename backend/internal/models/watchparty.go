package models

import "time"


type WatchParty struct {
	ID            int64     `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Date          time.Time `json:"date"`
	Status        string    `json:"status"`
	CreatorID     int64     `json:"creatorId"`
	ChosenMovieID int64     `json:"chosenMovieId,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

