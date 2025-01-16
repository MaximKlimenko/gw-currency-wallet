package tests

import (
	"io"
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

// Mock базы данных
type MockDB struct {
	mock.Mock
}

// Реализация метода CreateUser
func (m *MockDB) CreateUser(user *storages.User) error {
	args := m.Called(user)
	return args.Error(0)
}

// Реализация метода AuthenticateUser
func (m *MockDB) AuthenticateUser(user *storages.User) (bool, error) {
	args := m.Called(user)
	return args.Bool(0), args.Error(1)
}

// Реализация метода ExistingUser
func (m *MockDB) ExistingUser(username, email string) bool {
	args := m.Called(username, email)
	return args.Bool(0)
}

// Реализация метода CreateWallet
func (m *MockDB) CreateWallet(wallet *storages.Wallet) error {
	args := m.Called(wallet)
	return args.Error(0)
}

// Реализация метода GetBalanceByUsername
func (m *MockDB) GetBalanceByUsername(username string) (*storages.Wallet, error) {
	args := m.Called(username)
	return args.Get(0).(*storages.Wallet), args.Error(1)
}

// Реализация метода ChangeBalance
func (m *MockDB) ChangeBalance(amount float64, currency, username string) (*storages.Wallet, error) {
	args := m.Called(amount, currency, username)
	return args.Get(0).(*storages.Wallet), args.Error(1)
}

// Реализация метода CreateJWTToken
func (m *MockDB) CreateJWTToken(username string) (string, error) {
	args := m.Called(username)
	return args.String(0), args.Error(1)
}

func TestRegisterUser(t *testing.T) {
	mockDB := new(MockDB)
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

	// Проверка
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	// Проверяем вызовы мока
	mockDB.AssertExpectations(t)
}

func TestLogin(t *testing.T) {
	// Создаем мок
	mockDB := new(MockDB)
	repo := &delivery.Repository{DB: mockDB}

	// Настраиваем мок
	mockDB.On("AuthenticateUser", &storages.User{
		Username: "testuser",
		Password: "password",
	}).Return(true, nil)
	mockDB.On("CreateJWTToken", "testuser").Return("mocked_jwt_token", nil)

	// Создаем приложение Fiber
	app := fiber.New()
	app.Post("/login", repo.Login)

	// Отправляем запрос
	requestBody := `{
		"username": "testuser",
		"password_hash": "password"
	}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	// Проверяем статус ответа
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Проверяем тело ответа
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()

	expectedResponse := `{"token":"mocked_jwt_token"}`
	assert.JSONEq(t, expectedResponse, string(body))

	// Проверяем вызовы мока
	mockDB.AssertExpectations(t)
}
