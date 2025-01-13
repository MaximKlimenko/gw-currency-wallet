package storages

import (
	"time"
)

type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `gorm:"column:password_hash"`
	CreatedAt time.Time `json:"created_at"`
}

type Wallet struct {
	ID        int64
	UserID    int64
	Currency  string
	Balance   float64
	CreatedAt time.Time
}
