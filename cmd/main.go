package main

import (
	"log"

	"github.com/MaximKlimenko/gw-currency-wallet/internal/config"
	"github.com/MaximKlimenko/gw-currency-wallet/internal/delivery"
	"github.com/MaximKlimenko/gw-currency-wallet/internal/storages/db/postgres"
	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg := config.LoadConfig()

	db, err := postgres.NewConnector(cfg)
	if err != nil {
		log.Fatal("\033[31mcould not load the database\033[0m")
	}

	r := delivery.Repository{
		DB: db,
	}

	app := fiber.New()
	r.SetupRoutes(app)
	app.Listen(":3000")
}
