package postgres

import (
	"database/sql"
	"fmt"

	_ "github.com/MaximKlimenko/proto-exchange/exchange"
)

type Connector struct {
	DB *sql.DB
}

func NewConnector(host, port, user, password, dbname string) (*Connector, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Connector{DB: db}, nil
}

func (c *Connector) Close() error {
	return c.DB.Close()
}
