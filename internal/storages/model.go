package storages

import "time"

type User struct {
	ID        int64
	Name      string
	Email     string
	Password  string
	CreatedAt time.Time
}

type Wallet struct {
	ID        int64
	UserID    int64
	Currency  string
	Balance   float64
	CreatedAt time.Time
}

type Transaction struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	WalletID        int64     `json:"wallet_id"`
	Amount          float64   `json:"amount"`
	TransactionType string    `json:"transaction_type"`
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"created_at"`
}
