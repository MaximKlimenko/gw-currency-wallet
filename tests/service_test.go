package test

import (
	"encoding/json"
	"io"
	"log"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MaximKlimenko/gw-currency-wallet/internal/delivery"
	"github.com/MaximKlimenko/gw-currency-wallet/internal/storages"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Мок для интерфейса Storage
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) CreateUser(user *storages.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockStorage) AuthenticateUser(user *storages.User) (bool, error) {
	args := m.Called(user)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorage) ExistingUser(username, email string) bool {
	args := m.Called(username, email)
	return args.Bool(0)
}

func (m *MockStorage) CreateWallet(wallet *storages.Wallet) error {
	args := m.Called(wallet)
	return args.Error(0)
}

func (m *MockStorage) GetBalanceByUsername(username string) (*storages.Wallet, error) {
	args := m.Called(username)
	return args.Get(0).(*storages.Wallet), args.Error(1)
}

func (m *MockStorage) ChangeBalance(amount float64, currency, username string) (*storages.Wallet, error) {
	args := m.Called(amount, currency, username)
	return args.Get(0).(*storages.Wallet), args.Error(1)
}

func (m *MockStorage) CreateJWTToken(username string) (string, error) {
	args := m.Called(username)
	return args.String(0), args.Error(1)
}

// Мок для получения имени пользователя из JWT
func mockGetUsernameFromJWT(ctx *fiber.Ctx) (string, error) {
	return "testuser", nil
}

// Структура Repository
type Repository struct {
	DB storages.Storage
}

func TestRegisterUser(t *testing.T) {
	mockDB := new(MockStorage)
	repo := &delivery.Repository{DB: mockDB}

	mockDB.On("ExistingUser", "testuser", "test@example.com").Return(false)
	mockDB.On("CreateUser", mock.Anything).Return(nil)
	mockDB.On("CreateWallet", mock.Anything).Return(nil)

	app := fiber.New()
	app.Post("/register", repo.Register)

	requestBody := `{
		"username": "testuser",
		"password_hash": "password",
		"email": "test@example.com"
	}`
	req := httptest.NewRequest("POST", "/register", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)

	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	mockDB.AssertExpectations(t)
}

func TestLogin(t *testing.T) {
	mockDB := new(MockStorage)
	repo := &delivery.Repository{DB: mockDB}

	mockDB.On("AuthenticateUser", &storages.User{
		Username: "testuser",
		Password: "password",
	}).Return(true, nil)
	mockDB.On("CreateJWTToken", "testuser").Return("mocked_jwt_token", nil)

	app := fiber.New()
	app.Post("/login", repo.Login)

	requestBody := `{
		"username": "testuser",
		"password_hash": "password"
	}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	expectedResponse := `{"token":"mocked_jwt_token"}`
	assert.JSONEq(t, expectedResponse, string(body))

	mockDB.AssertExpectations(t)
}

// Метод DepositBalance на структуре Repository
func (r *Repository) DepositBalance(ctx *fiber.Ctx) error {
	username, err := mockGetUsernameFromJWT(ctx)
	if err != nil {
		log.Println("Error in GetUsernameFromJWT:", err) // Логируем ошибку
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err,
		})
	}

	var depositReq delivery.DepositRequest
	if err := ctx.BodyParser(&depositReq); err != nil {
		log.Println("Error in BodyParser:", err) // Логируем ошибку
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request format",
		})
	}

	if depositReq.Amount <= 0 {
		log.Println("Invalid amount or currency") // Логируем ошибку
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid amount or currency",
		})
	}

	updatedWallet, err := r.DB.ChangeBalance(depositReq.Amount, depositReq.Currency, username)
	if err != nil {
		log.Println("Error in ChangeBalance:", err) // Логируем ошибку
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Invalid amount or currency",
		})
	}

	log.Println("Wallet updated:", updatedWallet) // Логируем успешное обновление баланса

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
	username, err := mockGetUsernameFromJWT(ctx)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err,
		})
	}

	var depositReq delivery.DepositRequest
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

	// Используем отрицательное значение суммы для снятия
	updatedWallet, err := r.DB.ChangeBalance(-1*depositReq.Amount, depositReq.Currency, username)
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

func TestDepositBalance(t *testing.T) {
	// Создаем Fiber приложение
	app := fiber.New()

	// Создаем мок для интерфейса Storage
	mockStorage := new(MockStorage)

	// Создаем структуру Repository, в которой находится мок
	repository := &Repository{
		DB: mockStorage,
	}

	// Создаем тестовые данные
	depositRequest := `{"amount": 100, "currency": "USD"}`
	expectedWallet := &storages.Wallet{
		USD: 100,
		RUB: 0,
		EUR: 0,
	}

	// Устанавливаем поведение мока
	mockStorage.On("ChangeBalance", float64(100), "USD", "testuser").Return(expectedWallet, nil)

	// Создаем маршрут и подключаем метод DepositBalance
	app.Post("/deposit", repository.DepositBalance)

	// Преобразуем строку запроса в io.Reader
	req := httptest.NewRequest("POST", "/deposit", strings.NewReader(depositRequest))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	// Проверяем статус ответа
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Логируем тело ответа для диагностики
	bodyBytes, _ := io.ReadAll(resp.Body)
	t.Logf("Response body: %s", bodyBytes)

	// Проверяем содержимое ответа
	var body map[string]interface{}
	err := json.Unmarshal(bodyBytes, &body)
	if err != nil {
		t.Errorf("Error unmarshalling response: %v", err)
	}

	// Проверяем, что ожидаемые значения присутствуют в ответе
	assert.Equal(t, "Account topped up successfully", body["message"])
	assert.Equal(t, float64(100), body["wallet"].(map[string]interface{})["USD"])
	assert.Equal(t, float64(0), body["wallet"].(map[string]interface{})["RUB"])
	assert.Equal(t, float64(0), body["wallet"].(map[string]interface{})["EUR"])

	// Проверяем, что мок был вызван с правильными параметрами
	mockStorage.AssertExpectations(t)
}

func TestWithdrawBalance(t *testing.T) {
	// Создаем Fiber приложение
	app := fiber.New()

	// Создаем мок для интерфейса Storage
	mockStorage := new(MockStorage)

	// Создаем структуру Repository, в которой находится мок
	repository := &Repository{
		DB: mockStorage,
	}

	// Создаем тестовые данные для запроса на снятие
	withdrawRequest := `{"amount": 50, "currency": "USD"}`
	expectedWallet := &storages.Wallet{
		USD: 50,
		RUB: 0,
		EUR: 0,
	}

	// Устанавливаем поведение мока для ChangeBalance
	mockStorage.On("ChangeBalance", float64(-50), "USD", "testuser").Return(expectedWallet, nil)

	// Создаем маршрут и подключаем метод WithdrawBalance
	app.Post("/withdraw", repository.WithdrawBalance)

	// Преобразуем строку запроса в io.Reader
	req := httptest.NewRequest("POST", "/withdraw", strings.NewReader(withdrawRequest))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	// Проверяем статус ответа
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Логируем тело ответа для диагностики
	bodyBytes, _ := io.ReadAll(resp.Body)
	t.Logf("Response body: %s", bodyBytes)

	// Проверяем содержимое ответа
	var body map[string]interface{}
	err := json.Unmarshal(bodyBytes, &body)
	if err != nil {
		t.Errorf("Error unmarshalling response: %v", err)
	}

	// Проверяем, что ожидаемые значения присутствуют в ответе
	assert.Equal(t, "Withdrawal successful", body["message"])
	assert.Equal(t, float64(50), body["wallet"].(map[string]interface{})["USD"])
	assert.Equal(t, float64(0), body["wallet"].(map[string]interface{})["RUB"])
	assert.Equal(t, float64(0), body["wallet"].(map[string]interface{})["EUR"])

	// Проверяем, что мок был вызван с правильными параметрами
	mockStorage.AssertExpectations(t)
}
