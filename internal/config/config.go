package config

import (
	"os"
)

type Config struct {
	Host      string
	Port      string
	User      string
	Password  string
	DBName    string
	SSLMode   string
	GRPCPort  string
	JWTSecret string
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func LoadConfig() *Config {
	cfg := &Config{
		Host:      getEnv("DB_HOST", DefaultHost),
		Port:      getEnv("DB_PORT", DefaultPort),
		User:      getEnv("DB_USER", DefaultUser),
		Password:  getEnv("DB_PASSWORD", DefaultPassword),
		DBName:    getEnv("DB_NAME", DefaultName),
		SSLMode:   getEnv("DB_SSLMODE", DefaultSSLMode),
		GRPCPort:  getEnv("GRPC_PORT", DefaultGRPCPort),
		JWTSecret: getEnv("JWT_SECRET", DefaultJWTSecret),
	}
	return cfg
}
