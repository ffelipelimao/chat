package entity

import "time"

type Message struct {
	Username  string    `json:"Username"`
	Content   string    `json:"Content"`
	Timestamp time.Time `json:"Timestamp"`
}

type UserActivity struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	EventType string    `json:"eventType"`
	Timestamp time.Time `json:"timestamp"`
}
