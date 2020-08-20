package storage

import "time"

// User defines database user model and json tags for marshaling
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// Chat defines database chat model and json tags for marshaling
type Chat struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Users     []User    `json:"users"`
	CreatedAt time.Time `json:"created_at"`
}

// Message defines database message model and json tags for marshaling
type Message struct {
	ID        int64     `json:"id"`
	Chat      int64     `json:"chat"`
	Author    int64     `json:"author"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
