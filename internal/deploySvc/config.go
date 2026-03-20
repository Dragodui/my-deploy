package deploysvc

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	AgentURL string
	Port     int
	DBDsn    string
}

func NewConfig() *Config {
	agentURL := os.Getenv("AGENT_URL")
	portStr := os.Getenv("PORT")
	dbDsn := os.Getenv("DB_DSN")

	if portStr == "" {
		log.Fatal("no PORT in config")
	}

	if dbDsn == "" {
		log.Fatal("no DB_DSN in config")
	}

	if agentURL == "" {
		log.Fatal("no AGENT_URL in config")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatal("incorrect port value")
	}

	return &Config{
		Port:     port,
		DBDsn:    dbDsn,
		AgentURL: agentURL,
	}
}
