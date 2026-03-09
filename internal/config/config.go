package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port int
}

func NewConfig() *Config {
	port := 8080
	
	portStr := os.Getenv("PORT")
	if portStr != "" {
		if val, err := strconv.Atoi(portStr); err == nil {
			port = val
		}
	}

	return &Config{
		Port: port,
	}
}
