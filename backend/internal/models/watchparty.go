package models

import "time"


type WatchParty struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Status      string    `json:"status"`
	CreatorID   int64     `json:"creatorId"`
	CreatedAt   time.Time `json:"createdAt"`
}

