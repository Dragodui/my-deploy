package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/dragodui/my-deploy/internal/agent"
)

func runAgentInstallCommand(args []string) int {
	fs := flag.NewFlagSet("agent-install-command", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	name := fs.String("name", "", "remote agent name")
	publicURL := fs.String("public-url", "", "optional override for public gateway URL")
	binaryURL := fs.String("binary-url", "", "optional override for mydeploy-agent linux amd64 binary URL")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if strings.TrimSpace(*name) == "" {
		fmt.Fprintln(os.Stderr, "--name is required")
		return 2
	}
	cfg, err := agent.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		return 1
	}
	if cfg == nil || cfg.JWT == "" || agent.IsJWTExpired(cfg.JWT) {
		fmt.Fprintln(os.Stderr, "login required in local CLI before generating install command")
		return 1
	}

	apiBaseURL := strings.TrimSpace(cfg.URL)
	if apiBaseURL == "" {
		apiBaseURL = agent.DefaultServerURL
	}
	apiBaseURL = strings.TrimRight(apiBaseURL, "/")

	api := agent.NewAPIClient(apiBaseURL)
	meta, err := api.GetInstallMeta(cfg.JWT)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get install defaults: %v\n", err)
		return 1
	}

	serverURL := strings.TrimRight(strings.TrimSpace(meta.PublicURL), "/")
	if strings.TrimSpace(*publicURL) != "" {
		serverURL = strings.TrimRight(strings.TrimSpace(*publicURL), "/")
	}

	effectiveBinaryURL := strings.TrimSpace(meta.DefaultBinaryURL)
	if strings.TrimSpace(*binaryURL) != "" {
		effectiveBinaryURL = strings.TrimSpace(*binaryURL)
	}
	if serverURL == "" {
		fmt.Fprintln(os.Stderr, "gateway returned empty public URL")
		return 1
	}
	if effectiveBinaryURL == "" {
		fmt.Fprintln(os.Stderr, "gateway returned empty agent binary URL")
		return 1
	}

	bootstrapToken, expiresAt, err := api.CreateBootstrapToken(cfg.JWT, strings.TrimSpace(*name))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create bootstrap token: %v\n", err)
		return 1
	}

	installURL := serverURL + "/install/agent.sh?token=" + url.QueryEscape(bootstrapToken)
	if strings.TrimSpace(*publicURL) != "" {
		installURL += "&server_url=" + url.QueryEscape(serverURL)
	}
	if strings.TrimSpace(*binaryURL) != "" {
		installURL += "&binary_url=" + url.QueryEscape(effectiveBinaryURL)
	}

	fmt.Printf("# Agent: %s\n", strings.TrimSpace(*name))
	fmt.Printf("# Expires: %s\n", expiresAt)
	fmt.Printf("# Linux only\n")
	fmt.Printf("curl -fsSL %q | sh\n", installURL)

	return 0
}
