package storages

type Storage interface {
	CreateUser(user *User) error
	AuthenticateUser(user *User) (bool, error)
	ExistingUser(username, email string) bool

	CreateWallet(wallet *Wallet) error
	GetBalanceByUsername(username string) (*Wallet, error)
	ChangeBalance(amount float64, currency, username string) (*Wallet, error)

	CreateJWTToken(username string) (string, error)
}
