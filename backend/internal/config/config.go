package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	defaultPort              = "8080"
	defaultAppEnv            = "development"
	defaultSessionCookieName = "coffee_pos_session"
	defaultDBMaxOpenConns    = 3
	defaultDBMaxIdleConns    = 1
	jakartaTimezone          = "Asia/Jakarta"
	productionAppEnv         = "production"
)

type Config struct {
	Port                string
	AppEnv              string
	CashierPINHash      string
	SessionCookieName   string
	SessionCookieSecure bool
	BusinessLocation    *time.Location
}

type DatabaseConfig struct {
	URL          string
	MaxOpenConns int
	MaxIdleConns int
}

func Load() (Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = defaultAppEnv
	}

	cashierPINHash := strings.TrimSpace(os.Getenv("CASHIER_PIN_HASH"))
	if cashierPINHash == "" {
		return Config{}, fmt.Errorf("load config: CASHIER_PIN_HASH is required")
	}
	if _, err := bcrypt.Cost([]byte(cashierPINHash)); err != nil {
		return Config{}, fmt.Errorf("load config: CASHIER_PIN_HASH is invalid: %w", err)
	}

	sessionCookieName, err := sessionCookieNameFromEnv()
	if err != nil {
		return Config{}, err
	}

	sessionCookieSecure, err := sessionCookieSecureFromEnv(appEnv)
	if err != nil {
		return Config{}, err
	}

	location, err := time.LoadLocation(jakartaTimezone)
	if err != nil {
		return Config{}, fmt.Errorf("load config: load %s timezone: %w", jakartaTimezone, err)
	}

	return Config{
		Port:                port,
		AppEnv:              appEnv,
		CashierPINHash:      cashierPINHash,
		SessionCookieName:   sessionCookieName,
		SessionCookieSecure: sessionCookieSecure,
		BusinessLocation:    location,
	}, nil
}

func LoadDatabase() (DatabaseConfig, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return DatabaseConfig{}, fmt.Errorf("load database config: DATABASE_URL is required")
	}

	return DatabaseConfig{
		URL:          databaseURL,
		MaxOpenConns: defaultDBMaxOpenConns,
		MaxIdleConns: defaultDBMaxIdleConns,
	}, nil
}

func sessionCookieNameFromEnv() (string, error) {
	value, exists := os.LookupEnv("SESSION_COOKIE_NAME")
	if !exists {
		return defaultSessionCookieName, nil
	}

	if strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("load config: SESSION_COOKIE_NAME cannot be empty")
	}

	return value, nil
}

func sessionCookieSecureFromEnv(appEnv string) (bool, error) {
	if appEnv == productionAppEnv {
		return true, nil
	}

	value, exists := os.LookupEnv("SESSION_COOKIE_SECURE")
	if !exists {
		return false, nil
	}

	secure, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("load config: SESSION_COOKIE_SECURE must be a boolean: %w", err)
	}

	return secure, nil
}
