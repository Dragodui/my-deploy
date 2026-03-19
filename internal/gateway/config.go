package gateway

import (
	"log"
	"net/url"
	"os"
	"strconv"
)

type Config struct {
	Port      int
	JWTSecret string
	AuthURL   *url.URL
	// AgentURL    *url.URL
	// DeployURL   *url.URL
	// TemplateURL *url.URL
}

func LoadConfig() *Config {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	if port == 0 {
		port = 8080
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	authURL, err := url.Parse(os.Getenv("AUTH_SERVICE_URL"))
	if err != nil || authURL.Host == "" {
		log.Fatal("AUTH_SERVICE_URL is required")
	}

	// agentURL, err := url.Parse(os.Getenv("AGENT_SERVICE_URL"))
	// if err != nil || agentURL.Host == "" {
	// 	log.Fatal("AGENT_SERVICE_URL is required")
	// }

	// deployURL, err := url.Parse(os.Getenv("DEPLOY_SERVICE_URL"))
	// if err != nil || deployURL.Host == "" {
	// 	log.Fatal("DEPLOY_SERVICE_URL is required")
	// }

	// templateURL, err := url.Parse(os.Getenv("TEMPLATE_SERVICE_URL"))
	// if err != nil || templateURL.Host == "" {
	// 	log.Fatal("TEMPLATE_SERVICE_URL is required")
	// }

	return &Config{
		Port:      port,
		JWTSecret: jwtSecret,
		AuthURL:   authURL,
		// AgentURL:    agentURL,
		// DeployURL:   deployURL,
		// TemplateURL: templateURL,
	}
}
