package storages

type Storage interface {
	CreateUser(user *User) error
	GetUserByID(userID int64) (*User, error)
	CreateWallet(wallet *Wallet) error
	GetWalletByUserID(userID int64) (*Wallet, error)
	UpdateWalletBalance(walletID int64, balance float64) error
	CreateTransaction(transaction *Transaction) error
	GetTransactionsByUserID(userID int64) ([]Transaction, error)
}
