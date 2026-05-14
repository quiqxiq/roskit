package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port int
	Env  string

	// PostgreSQL — individual credentials
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis — individual credentials
	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// InfluxDB 3 Core (optional)
	InfluxURL      string
	InfluxToken    string
	InfluxDatabase string

	// Public URL
	PublicAPIURL string

	// Security
	JWTSecret        string
	JWTRefreshSecret string
	AESEncKey        string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	return &Config{
		Port: getEnvInt("PORT", 8080),
		Env:  getEnv("APP_ENV", "development"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnvInt("DB_PORT", 5432),
		DBUser:     getEnv("DB_USER", "mikhmon"),
		DBPassword: getEnv("DB_PASSWORD", "mikhmon"),
		DBName:     getEnv("DB_NAME", "mikhmon"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),

		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		InfluxURL:      getEnv("INFLUXDB_URL", ""),
		InfluxToken:    getEnv("INFLUXDB_TOKEN", ""),
		InfluxDatabase: getEnv("INFLUXDB_DATABASE", "mikhmon"),

		PublicAPIURL: getEnv("PUBLIC_API_URL", ""),

		JWTSecret:        getEnv("JWT_SECRET", ""),
		JWTRefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),
		AESEncKey:        getEnv("AES_ENCRYPTION_KEY", ""),
	}, nil
}

func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if c.JWTRefreshSecret == "" {
		return fmt.Errorf("JWT_REFRESH_SECRET is required")
	}
	if c.AESEncKey == "" {
		return fmt.Errorf("AES_ENCRYPTION_KEY is required")
	}
	return nil
}

// PostgresDSN returns a GORM-compatible DSN string.
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

// RedisURL returns a redis:// URL string for go-redis ParseURL.
func (c *Config) RedisURL() string {
	if c.RedisPassword != "" {
		return fmt.Sprintf("redis://:%s@%s:%d/%d", c.RedisPassword, c.RedisHost, c.RedisPort, c.RedisDB)
	}
	return fmt.Sprintf("redis://%s:%d/%d", c.RedisHost, c.RedisPort, c.RedisDB)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return fallback
}
