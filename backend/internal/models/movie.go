package models

import "time"


type Movie struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Genre       string    `json:"genre"`
	Duration    int       `json:"duration"`
	ReleaseYear int       `json:"releaseYear"`
	Synopsis    string    `json:"synopsis"`
	PosterURL   string    `json:"posterUrl"`
	CreatedAt   time.Time `json:"createdAt"`
}

