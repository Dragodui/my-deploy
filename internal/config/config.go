package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port             int
	DockerHost       string
	DockerAPIVersion string
}

func NewConfig() *Config {
	port := 8080

	portStr := os.Getenv("PORT")
	if portStr != "" {
		if val, err := strconv.Atoi(portStr); err == nil {
			port = val
		}
	}

	dockerHost := os.Getenv("DOCKER_HOST")
	dockerAPIVersion := os.Getenv("DOCKER_API_VERSION")
	if dockerHost == "" || dockerAPIVersion == "" {
		log.Fatal("DOCKER_HOST or DOCKER_API_VERSION are not in env")
	}

	return &Config{
		Port:             port,
		DockerHost:       dockerHost,
		DockerAPIVersion: dockerAPIVersion,
	}
}
