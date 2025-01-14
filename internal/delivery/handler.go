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

	// Создаём кошелёк с нулевым балансом для нового пользователя
	newWallet := &storages.Wallet{
		UserID: newUser.ID, // Привязываем кошелёк к ID пользователя
		USD:    0.0,        // Начальный баланс в долларах
		RUB:    0.0,        // Начальный баланс в рублях
		EUR:    0.0,        // Начальный баланс в евро
	}

	// Сохраняем кошелёк в базе данных
	if err := r.DB.CreateWallet(newWallet); err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Ошибка при создании кошелька",
		})
	}

	// Возвращаем успешный ответ
	return ctx.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
	})
}
func (r *Repository) Login(ctx *fiber.Ctx) error {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password_hash"`
	}

	if err := ctx.BodyParser(&input); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Неверный формат входных данных",
		})
	}

	oldUser := &storages.User{
		Username: input.Username,
		Password: input.Password, //Нет необходимости хэшировать пароль, так как в методе AuthenticateUser происходит сравнение хэша из бд и пароля
	}

	authenticated, err := r.DB.AuthenticateUser(oldUser)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Ошибка авторизации",
		})
	}

	if !authenticated {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid username or password",
		})
	}

	token, err := r.DB.CreateJWTToken(oldUser.Username)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create token",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"token": token,
	})
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
