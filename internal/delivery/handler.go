package delivery

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MaximKlimenko/gw-currency-wallet/internal/grpc/exchanger"
	"github.com/MaximKlimenko/gw-currency-wallet/internal/storages"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
)

type DepositRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

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
	username, err := getUsernameFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err,
		})
	}

	wal, err := r.DB.GetBalanceByUsername(username)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Something went wrong",
		})
	}
	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"USD": wal.USD,
		"RUB": wal.RUB,
		"EUR": wal.EUR,
	})
}
func (r *Repository) DepositBalance(ctx *fiber.Ctx) error {
	username, err := getUsernameFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err,
		})
	}

	var depositReq DepositRequest
	if err := ctx.BodyParser(&depositReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	if depositReq.Amount <= 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid amount or currency",
		})
	}

	updatedWallet, err := r.DB.ChangeBalance(depositReq.Amount, depositReq.Currency, username)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Invalid amount or currency",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Account topped up successfully",
		"wallet": fiber.Map{
			"USD": updatedWallet.USD,
			"RUB": updatedWallet.RUB,
			"EUR": updatedWallet.EUR,
		},
	})
}
func (r *Repository) WithdrawBalance(ctx *fiber.Ctx) error {
	username, err := getUsernameFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err,
		})
	}

	var depositReq DepositRequest
	if err := ctx.BodyParser(&depositReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	if depositReq.Amount <= 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Insufficient funds or invalid amount",
		})
	}

	updatedWallet, err := r.DB.ChangeBalance(-1*depositReq.Amount, depositReq.Currency, username) //Таже самая функция, только amount * -1
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Insufficient funds or invalid amount",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Withdrawal successful",
		"wallet": fiber.Map{
			"USD": updatedWallet.USD,
			"RUB": updatedWallet.RUB,
			"EUR": updatedWallet.EUR,
		},
	})
}

func (r *Repository) GetExchangeRates(ctx *fiber.Ctx) error {
	conn, err := grpc.Dial("your_grpc_service_address:50051", grpc.WithInsecure())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to connect to exchange service",
		})
	}
	defer conn.Close()

	client := exchanger.NewExchangerClient(conn)

	rates, err := client.GetExchangeRates()
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve exchange rates",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"rates": rates,
	})
}
func (r *Repository) GetExchange(ctx *fiber.Ctx) error {
	return nil
}

func getUsernameFromJWT(ctx *fiber.Ctx) (string, error) {
	authHeader := ctx.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("Authorization token is required")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return "", fmt.Errorf("Invalid token format")
	}

	claims := &struct {
		Username string `json:"username"`
		jwt.RegisteredClaims
	}{}

	secretKey := []byte("qwerty")

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Проверка, что алгоритм токена правильный
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secretKey, nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("Invalid or expired token")
	}

	return claims.Username, nil
}
