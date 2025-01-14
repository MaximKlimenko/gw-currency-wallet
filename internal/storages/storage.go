package storages

type Storage interface {
	CreateUser(user *User) error
	AuthenticateUser(user *User) (bool, error)
	ExistingUser(username, email string) bool

	CreateWallet(wallet *Wallet) error

	CreateJWTToken(username string) (string, error)
}
