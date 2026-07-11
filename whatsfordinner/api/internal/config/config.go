// Package config loads runtime configuration from environment variables.
package config

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration. All values can be overridden
// through environment variables, which makes the service easy to configure
// when running inside a container on the NAS.
type Config struct {
	Host         string        // WFD_HOST    (default: "" -> all interfaces)
	Port         string        // WFD_PORT    (default: "8080")
	ReadTimeout  time.Duration // WFD_READ_TIMEOUT  (default: 5s)
	WriteTimeout time.Duration // WFD_WRITE_TIMEOUT (default: 10s)
	IdleTimeout  time.Duration // WFD_IDLE_TIMEOUT  (default: 120s)

	// Database connection. DatabaseURL takes precedence; when it is empty the
	// DSN is assembled from the discrete DB* fields (see DatabaseDSN).
	DatabaseURL   string        // DATABASE_URL (e.g. postgres://user:pass@host:5432/db)
	DBHost        string        // WFD_DB_HOST      (default: "localhost")
	DBPort        string        // WFD_DB_PORT      (default: "5432")
	DBUser        string        // WFD_DB_USER      (default: "whatsfordinner")
	DBPassword    string        // WFD_DB_PASSWORD  (default: "")
	DBName        string        // WFD_DB_NAME      (default: "whatsfordinner")
	DBSSLMode     string        // WFD_DB_SSLMODE   (default: "disable")
	DBMaxConns    int32         // WFD_DB_MAX_CONNS (default: 10)
	DBConnTimeout time.Duration // WFD_DB_CONN_TIMEOUT (default: 10s)

	// Gemini (receipt scanning). GeminiAPIKey has no default — the "Scan
	// receipt" feature simply fails per-request until one is set.
	GeminiAPIKey string // WFD_GEMINI_API_KEY (default: "")
	GeminiModel  string // WFD_GEMINI_MODEL   (default: "gemini-3.1-flash-lite")
}

// Load reads configuration from environment variables, applying sensible
// defaults whenever a variable is not set.
func Load() Config {
	return Config{
		Host:         getEnv("WFD_HOST", ""),
		Port:         getEnv("WFD_PORT", "8080"),
		ReadTimeout:  getEnvDuration("WFD_READ_TIMEOUT", 5*time.Second),
		WriteTimeout: getEnvDuration("WFD_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:  getEnvDuration("WFD_IDLE_TIMEOUT", 120*time.Second),

		DatabaseURL:   getEnv("DATABASE_URL", ""),
		DBHost:        getEnv("WFD_DB_HOST", "localhost"),
		DBPort:        getEnv("WFD_DB_PORT", "5432"),
		DBUser:        getEnv("WFD_DB_USER", "whatsfordinner"),
		DBPassword:    getEnv("WFD_DB_PASSWORD", ""),
		DBName:        getEnv("WFD_DB_NAME", "whatsfordinner"),
		DBSSLMode:     getEnv("WFD_DB_SSLMODE", "disable"),
		DBMaxConns:    int32(getEnvInt("WFD_DB_MAX_CONNS", 10)),
		DBConnTimeout: getEnvDuration("WFD_DB_CONN_TIMEOUT", 10*time.Second),

		GeminiAPIKey: getEnv("WFD_GEMINI_API_KEY", ""),
		GeminiModel:  getEnv("WFD_GEMINI_MODEL", "gemini-3.1-flash-lite"),
	}
}

// Addr returns the address the HTTP server should listen on.
func (c Config) Addr() string {
	return net.JoinHostPort(c.Host, c.Port)
}

// DatabaseDSN returns the PostgreSQL connection string. DatabaseURL is used as
// is when set; otherwise a DSN is built from the discrete DB* settings.
func (c Config) DatabaseDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}

	dsn := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DBUser, c.DBPassword),
		Host:   net.JoinHostPort(c.DBHost, c.DBPort),
		Path:   "/" + c.DBName,
	}

	query := url.Values{}
	if c.DBSSLMode != "" {
		query.Set("sslmode", c.DBSSLMode)
	}
	dsn.RawQuery = query.Encode()

	return dsn.String()
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}
