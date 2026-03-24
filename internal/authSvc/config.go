package authsvc

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port      int
	GRPCPort  int
	JWTSecret string
	DBDsn     string
}

func NewConfig() *Config {
	portStr := os.Getenv("PORT")
	grpcPortStr := os.Getenv("GRPC_PORT")
	jwtSecret := os.Getenv("JWT_SECRET")
	dbDsn := os.Getenv("DB_DSN")

	if portStr == "" {
		log.Fatal("no PORT in config")
	}

	if grpcPortStr == "" {
		log.Fatal("no GRPC_PORT in config")
	}
	if jwtSecret == "" {
		log.Fatal("no JWT_SECRET in config")
	}
	if dbDsn == "" {
		log.Fatal("no DB_DSN in config")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatal("incorrect port value")
	}

	grpcPort, err := strconv.Atoi(grpcPortStr)
	if err != nil {
		log.Fatal("incorrect grpc port value")
	}

	return &Config{
		Port:      port,
		GRPCPort:  grpcPort,
		JWTSecret: jwtSecret,
		DBDsn:     dbDsn,
	}
}
