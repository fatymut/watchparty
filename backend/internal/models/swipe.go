package models

import "time"



type Swipe struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"userId"`
	MovieID      int64     `json:"movieId"`
	WatchPartyID int64     `json:"watchPartyId"`
	Value        string    `json:"value"`
	SwipedAt     time.Time `json:"swipedAt"`
}