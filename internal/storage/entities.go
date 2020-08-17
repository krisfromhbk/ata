package storage

import "time"

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

type Chat struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Users     []User    `json:"users"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID        int64     `json:"id"`
	Chat      int64     `json:"chat"`
	Author    int64     `json:"author"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
