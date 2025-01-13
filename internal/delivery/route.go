package delivery

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type Repository struct {
	DB *gorm.DB
}

func (r *Repository) SetupRoutes(app *fiber.App) {
	api := app.Group("api/v1")
	api.Post("/register", r.Register)
	api.Post("/login", r.Login)
	api.Get("/balance", r.GetBalance)
	api.Post("/wallet/deposit", r.DepositBalance)
	api.Post("/wallet/withdraw", r.WithdrawBalance)
	api.Get("/exchange/rates", r.GetExchangeRates)
	api.Get("/exchange", r.GetExchange)
}
