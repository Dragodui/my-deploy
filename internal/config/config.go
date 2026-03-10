package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port  int
	DBDSN string
}

func NewConfig() *Config {
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

	return &Config{
		Port:  port,
		DBDSN: dbDSN,
	}
}
