package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort        string
	DatabaseURL       string
	MinIOEndpoint     string
	MinIOAccessKey    string
	MinIOSecretKey    string
	MinIOBucket       string
	MQTTBroker        string
	MQTTEnabled       bool
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime int
}

func Load() (*Config, error) {
	_ = godotenv.Load() // Load .env file if it exists

	return &Config{
		ServerPort:        getEnv("SERVER_PORT", "8000"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgresql://root@localhost:26257/defaultdb?sslmode=disable"),
		MinIOEndpoint:     getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:    getEnv("MINIO_ACCESS_KEY", "admin"),
		MinIOSecretKey:    getEnv("MINIO_SECRET_KEY", "password"),
		MinIOBucket:       getEnv("MINIO_BUCKET", "wasm-functions"),
		MQTTBroker:        getEnv("MQTT_BROKER", "tcp://localhost:1883"),
		MQTTEnabled:       getEnvAsBool("MQTT_ENABLED", false),
		DBMaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
		DBConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 300), // seconds
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		b, err := strconv.ParseBool(value)
		if err == nil {
			return b
		}
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		i, err := strconv.Atoi(value)
		if err == nil {
			return i
		}
	}
	return fallback
}
