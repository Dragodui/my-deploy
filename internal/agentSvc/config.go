package agentsvc

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	Port     int
	GRPCPort int
	DBDsn    string
}

func NewConfig() *Config {
	portStr := os.Getenv("PORT")
	grpcPortStr := os.Getenv("GRPC_PORT")
	dbDsn := os.Getenv("DB_DSN")

	if portStr == "" {
		log.Fatal("no PORT in config")
	}

	if grpcPortStr == "" {
		log.Fatal("no GRPC_PORT in config")
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
		Port:     port,
		GRPCPort: grpcPort,
		DBDsn:    dbDsn,
	}
}
