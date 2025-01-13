package delivery

import (
	"time"

	"github.com/MaximKlimenko/gw-currency-wallet/internal/storages"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

func (r *Repository) Register(ctx *fiber.Ctx) error {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password_hash"`
		Email    string `json:"email"`
	}

	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат входных данных",
		})
	}

	if r.DB.ExistingUser(input.Username, input.Email) {
		return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Username or email already exists",
		})
	}

	// Хэшируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Ошибка при обработке пароля",
		})
	}

	// Создаём новую сущность пользователя
	newUser := &storages.User{
		Username:  input.Username,
		Password:  string(hashedPassword), // Сохраняем хэш пароля
		Email:     input.Email,
		CreatedAt: time.Now(),
	}

	// Используем метод CreateUser для сохранения пользователя
	if err := r.DB.CreateUser(newUser); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Ошибка при создании пользователя",
		})
	}

	// Возвращаем успешный ответ
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
	})
}
func (r *Repository) Login(ctx *fiber.Ctx) error {
	return nil
}
func (r *Repository) GetBalance(ctx *fiber.Ctx) error {
	return nil
}
func (r *Repository) DepositBalance(ctx *fiber.Ctx) error {
	return nil
}
func (r *Repository) WithdrawBalance(ctx *fiber.Ctx) error {
	return nil
}
func (r *Repository) GetExchangeRates(ctx *fiber.Ctx) error {
	return nil
}
func (r *Repository) GetExchange(ctx *fiber.Ctx) error {
	return nil
}
