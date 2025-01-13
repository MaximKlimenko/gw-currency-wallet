package storages

type Storage interface {
	CreateUser(user *User) error
	AuthenticateUser(user *User) (bool, error)
}
