package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port      int
	DBDSN     string
	JWTSecret string
}

func NewConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading env")
	}
	port := 8080

	portStr := os.Getenv("PORT")
	if portStr != "" {
		if val, err := strconv.Atoi(portStr); err == nil {
			port = val
		}
	}

	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		log.Fatal("DB_DSN is required")
	}

	JWTSecret := os.Getenv("JWT_SECRET")
	if JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	return &Config{
		Port:      port,
		DBDSN:     dbDSN,
		JWTSecret: JWTSecret,
	}
}
