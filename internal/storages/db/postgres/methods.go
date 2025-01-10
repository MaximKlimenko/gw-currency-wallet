package postgres

import (
	"database/sql"
	"errors"
	"time"

	"github.com/MaximKlimenko/gw-currency-wallet/internal/storages"
)

type PostgresStorage struct {
	conn *Connector
}

func NewPostgresStorage(conn *Connector) *PostgresStorage {
	return &PostgresStorage{conn: conn}
}

func (s *PostgresStorage) CreateUser(user *storages.User) error {
	query := `INSERT INTO users (name, email, password, created_at) VALUES ($1, $2, $3, $4) RETURNING id`
	return s.conn.DB.QueryRow(query, user.Name, user.Email, user.Password, time.Now()).Scan(&user.ID)
}

func (s *PostgresStorage) GetUserByID(userID int64) (*storages.User, error) {
	query := `SELECT id, name, email, password, created_at FROM users WHERE id = $1`
	user := &storages.User{}
	err := s.conn.DB.QueryRow(query, userID).Scan(&user.ID, &user.Name, &user.Email, &user.Password, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *PostgresStorage) CreateWallet(wallet *storages.Wallet) error {
	query := `INSERT INTO wallets (user_id, currency, balance, created_at) VALUES ($1, $2, $3, $4) RETURNING id`
	return s.conn.DB.QueryRow(query, wallet.UserID, wallet.Currency, wallet.Balance, time.Now()).Scan(&wallet.ID)
}

func (s *PostgresStorage) UpdateWalletBalance(walletID int64, balance float64) error {
	query := `UPDATE wallets SET balance = $1 WHERE id = $2`
	result, err := s.conn.DB.Exec(query, balance, walletID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("кошелек с указанным ID не найден")
	}

	return nil
}

func (s *PostgresStorage) GetWalletByUserID(userID int64) (*storages.Wallet, error) {
	query := `SELECT id, user_id, currency, balance, created_at FROM wallets WHERE user_id = $1`
	var wallet storages.Wallet

	err := s.conn.DB.QueryRow(query, userID).Scan(
		&wallet.ID,
		&wallet.UserID,
		&wallet.Currency,
		&wallet.Balance,
		&wallet.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.New("кошелек для указанного пользователя не найден")
	}
	if err != nil {
		return nil, err
	}

	return &wallet, nil
}

func (s *PostgresStorage) CreateTransaction(userID int64, walletID int64, amount float64, transactionType string, description string) error {
	query := `INSERT INTO transactions (user_id, wallet_id, amount, transaction_type, description, created_at)
              VALUES ($1, $2, $3, $4, $5, NOW())`

	_, err := s.conn.DB.Exec(query, userID, walletID, amount, transactionType, description)
	if err != nil {
		return err
	}

	return nil
}

func (s *PostgresStorage) GetTransactionsByUserID(userID int64) ([]storages.Transaction, error) {
	query := `SELECT id, user_id, wallet_id, amount, transaction_type, description, created_at 
              FROM transactions WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := s.conn.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []storages.Transaction
	for rows.Next() {
		var tx storages.Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.UserID,
			&tx.WalletID,
			&tx.Amount,
			&tx.TransactionType,
			&tx.Description,
			&tx.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return transactions, nil
}
