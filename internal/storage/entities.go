package storage

import "time"

type User struct {
	ID        int64
	Username  string
	CreatedAt time.Time
}

type Chat struct {
	ID        int64
	Name      string
	Users     []User
	CreatedAt time.Time
}

type Message struct {
	ID        int64
	Chat      int64
	Author    int64
	Text      string
	CreatedAt time.Time
}
