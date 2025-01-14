package postgres

import (
	"errors"
	"fmt"
	"time"

	"github.com/MaximKlimenko/gw-currency-wallet/internal/config"
	"github.com/MaximKlimenko/gw-currency-wallet/internal/storages"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type PostgresStorage struct {
	conn   *Connector
	Config *config.Config
}

func NewPostgresStorage(conn *Connector) *PostgresStorage {
	return &PostgresStorage{conn: conn}
}

func (s *PostgresStorage) CreateUser(user *storages.User) error {
	user.CreatedAt = time.Now()
	return s.conn.DB.Create(user).Error
}

func (s *PostgresStorage) AuthenticateUser(user *storages.User) (bool, error) {
	var dbUser storages.User

	err := s.conn.DB.Where("username = ?", user.Username).First(&dbUser).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil // Пользователь не найден
		}
		return false, fmt.Errorf("\033[31mОшибка аутентификации: %w\033[0m", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password)); err != nil {
		return false, nil
	}

	return true, nil
}

func (s *PostgresStorage) ExistingUser(username, email string) bool {
	var existingUser storages.User
	if err := s.conn.DB.Where("username = ? OR email = ?", username, email).First(&existingUser).Error; err == nil {
		return true
	}

	return false
}

func (s *PostgresStorage) CreateWallet(wallet *storages.Wallet) error {
	return s.conn.DB.Create(wallet).Error
}

func (s *PostgresStorage) CreateJWTToken(username string) (string, error) {
	// Настраиваем параметры токена
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 72).Unix(), // Токен действует 72 часа
	}

	// Создаём токен
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен с помощью секретного ключа
	return token.SignedString([]byte(s.Config.JWTSecret))
}
