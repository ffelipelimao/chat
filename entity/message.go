package entity

import "time"

type Message struct {
	Username  string
	Content   string
	Timestamp time.Time
}

type UserActivity struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	EventType string    `json:"eventType"`
	Timestamp time.Time `json:"timestamp"`
}
