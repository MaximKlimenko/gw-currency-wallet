// @title           gw-currency-wallet
// @version         1.0
// @description     Это описание моего API.

// @contact.email  max.klim59@gmail.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:3000
// @BasePath  /api/v1

package main

import (
	"log"

	_ "github.com/MaximKlimenko/gw-currency-wallet/docs"
	"github.com/MaximKlimenko/gw-currency-wallet/internal/config"
	"github.com/MaximKlimenko/gw-currency-wallet/internal/delivery"
	"github.com/MaximKlimenko/gw-currency-wallet/internal/storages/db/postgres"
	"github.com/gofiber/fiber/v2"
)

func main() {
	cfg := config.LoadConfig()

	cnt, err := postgres.NewConnector(cfg)
	if err != nil {
		log.Fatal("\033[31mcould not load the database\033[0m")
	}

	pgrep := postgres.NewPostgresStorage(cnt, cfg)

	r := delivery.Repository{
		DB: pgrep,
	}

	app := fiber.New()
	r.SetupRoutes(app)
	app.Listen(":3000")
}
