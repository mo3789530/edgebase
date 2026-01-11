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
	// AWS S3 Config
	S3Enabled   bool
	S3Region    string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string

	MQTTBroker        string
	MQTTEnabled       bool
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime int
	JWTSecret         string
	TokenExpiryHours  int

	// TimeSeries Config
	TimeSeriesEnabled       bool
	TimeSeriesDBURL         string
	TimeSeriesDBToken       string
	TimeSeriesDBOrg         string
	TimeSeriesDBBucket      string
	TimeSeriesBatchSize     int
	TimeSeriesBatchTimeout  int // seconds
	TimeSeriesRetentionDays int
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

		S3Enabled:   getEnvAsBool("S3_ENABLED", false),
		S3Region:    getEnv("AWS_REGION", "us-east-1"),
		S3AccessKey: getEnv("AWS_ACCESS_KEY_ID", ""),
		S3SecretKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		S3Bucket:    getEnv("AWS_BUCKET", "wasm-functions"),

		MQTTBroker:        getEnv("MQTT_BROKER", "tcp://localhost:1883"),
		MQTTEnabled:       getEnvAsBool("MQTT_ENABLED", false),
		DBMaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 25),
		DBConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 300), // seconds
		JWTSecret:         getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		TokenExpiryHours:  getEnvAsInt("TOKEN_EXPIRY_HOURS", 24),

		TimeSeriesEnabled:       getEnvAsBool("TIMESERIES_ENABLED", false),
		TimeSeriesDBURL:         getEnv("TIMESERIES_DB_URL", "http://localhost:8086"),
		TimeSeriesDBToken:       getEnv("TIMESERIES_DB_TOKEN", ""),
		TimeSeriesDBOrg:         getEnv("TIMESERIES_DB_ORG", ""),
		TimeSeriesDBBucket:      getEnv("TIMESERIES_DB_BUCKET", "metrics"),
		TimeSeriesBatchSize:     getEnvAsInt("TIMESERIES_BATCH_SIZE", 100),
		TimeSeriesBatchTimeout:  getEnvAsInt("TIMESERIES_BATCH_TIMEOUT", 5),
		TimeSeriesRetentionDays: getEnvAsInt("TIMESERIES_RETENTION_DAYS", 30),
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
