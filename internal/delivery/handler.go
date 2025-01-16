package delivery

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MaximKlimenko/gw-currency-wallet/internal/storages"
	"github.com/MaximKlimenko/gw-currency-wallet/pkg/grpc/exchanger"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
)

type DepositRequest struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}
type ExchangeRequest struct {
	FromCurrency string  `json:"from_currency"`
	ToCurrency   string  `json:"to_currency"`
	Amount       float64 `json:"amount"`
}
type RegisterUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password_hash"`
	Email    string `json:"email"`
}
type LoginUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password_hash"`
}

// Register регистрирует нового пользователя и создаёт для него кошелёк
// @Summary Регистрация нового пользователя
// @Description Создаёт нового пользователя в системе и инициализирует кошелёк с нулевым балансом
// @Tags users
// @Accept json
// @Produce json
// @Param user body RegisterUserRequest true "Данные пользователя для регистрации"
// @Success 201 {object} map[string]string "User registered successfully"
// @Failure 400 {object} map[string]string "Неверный формат входных данных"
// @Failure 409 {object} map[string]string "Username or email already exists"
// @Failure 500 {object} map[string]string "Ошибка при создании пользователя или кошелька"
// @Router /register [post]
func (r *Repository) Register(ctx *fiber.Ctx) error {
	var input RegisterUserRequest

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

// Login авторизация пользователя
// @Summary Авторизация старого пользователя
// @Description Авторизация пользователя с помощью JWT токена
// @Tags users
// @Accept json
// @Produce json
// @Param user body LoginUserRequest true "Данные пользователя для авторизации"
// @Success 200 {object} map[string]string "Токен авторизации"
// @Failure 400 {object} map[string]string "Неверный формат входных данных"
// @Failure 401 {object} map[string]string "Ошибка авторизации или неверные учетные данные"
// @Failure 500 {object} map[string]string "Ошибка сервера при создании токена"
// @Router /login [post]
func (r *Repository) Login(ctx *fiber.Ctx) error {
	var input LoginUserRequest

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

// GetBalance возвращает баланс пользователя
// @Summary Получение баланса пользователя
// @Description Возвращает текущий баланс пользователя по всем валютам
// @Tags balance
// @Accept json
// @Produce json
// @Success 200 {object} map[string]float64 "Баланс по валютам (USD, RUB, EUR)"
// @Failure 401 {object} map[string]string "Ошибка авторизации или токен недействителен"
// @Failure 500 {object} map[string]string "Ошибка на стороне сервера"
// @Router /balance [get]
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

// DepositBalance пополняет баланс пользователя
// @Summary Пополнение баланса
// @Description Добавляет указанную сумму на баланс пользователя в выбранной валюте
// @Tags balance
// @Accept json
// @Produce json
// @Param deposit body DepositRequest true "Сумма и валюта пополнения"
// @Success 200 {object} map[string]interface{} "Сообщение об успехе и обновленный баланс"
// @Failure 400 {object} map[string]string "Некорректный запрос или неверная сумма"
// @Failure 500 {object} map[string]string "Ошибка на стороне сервера"
// @Router /wallet/deposit [post]
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

// WithdrawBalance уменьшает баланс пользователя
// @Summary Снятие баланса
// @Description Уменьшает указанную сумму на баланс пользователя в выбранной валюте
// @Tags balance
// @Accept json
// @Produce json
// @Param deposit body DepositRequest true "Сумма и валюта пополнения"
// @Success 200 {object} map[string]interface{} "Сообщение об успехе и обновленный баланс"
// @Failure 400 {object} map[string]string "Некорректный запрос или неверная сумма"
// @Failure 500 {object} map[string]string "Ошибка на стороне сервера"
// @Router /wallet/Withdraw [post]
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

// GetExchangeRates возвращает все доступные курсы валют
// @Summary Получение курсов валют
// @Description Возвращает список всех курсов валют, доступных в системе
// @Tags exchange
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Список курсов валют"
// @Failure 500 {object} map[string]string "Ошибка на стороне сервера"
// @Router /exchange/rates [get]
func (r *Repository) GetExchangeRates(ctx *fiber.Ctx) error { // Пока-что метод выводит вообще все курсы валют, хранящихся в бд
	conn, err := grpc.NewClient("localhost:50051", grpc.WithInsecure())
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

// GetExchange выполняет обмен валюты для пользователя
// @Summary Обмен валюты
// @Description Выполняет обмен указанной суммы одной валюты на другую для пользователя
// @Tags exchange
// @Accept json
// @Produce json
// @Param exchange body ExchangeRequest true "Параметры обмена (сумма, из какой валюты, в какую)"
// @Success 200 {object} map[string]interface{} "Информация об успешном обмене и новый баланс"
// @Failure 400 {object} map[string]string "Некорректный запрос или недостаточно средств"
// @Failure 500 {object} map[string]string "Ошибка на стороне сервера"
// @Router /exchange [post]
func (r *Repository) GetExchange(ctx *fiber.Ctx) error {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithInsecure())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to connect to exchange service",
		})
	}
	defer conn.Close()

	username, err := getUsernameFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err,
		})
	}

	var exchangeReq ExchangeRequest
	if err := ctx.BodyParser(&exchangeReq); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}
	if exchangeReq.Amount <= 0 {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Insufficient funds or invalid amount",
		})
	}

	client := exchanger.NewExchangerClient(conn)
	fmt.Println(exchangeReq)
	rate, err := client.GetExchangeRate(exchangeReq.FromCurrency, exchangeReq.ToCurrency)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve exchange rates",
			"err":   err,
		})
	}

	changedBalance, err := r.DB.ChangeBalance(-1*exchangeReq.Amount, exchangeReq.FromCurrency, username)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Insufficient funds or invalid amount",
		})
	}

	changedBalance, err = r.DB.ChangeBalance(exchangeReq.Amount*rate, exchangeReq.ToCurrency, username) // Говнокод
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Insufficient funds or invalid amount",
		})
	}

	return ctx.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":          "Exchange successful",
		"exchanged_amount": exchangeReq.Amount * rate,
		"new_balance": fiber.Map{
			"USD": changedBalance.USD,
			"RUB": changedBalance.RUB,
			"EUR": changedBalance.EUR,
		},
	})
}

func getUsernameFromJWT(ctx *fiber.Ctx) (string, error) {
	authHeader := ctx.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization token is required")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == "" {
		return "", fmt.Errorf("invalid token format")
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
		return "", fmt.Errorf("invalid or expired token")
	}

	return claims.Username, nil
}
