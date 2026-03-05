// actuator-g — Botster Actuator in Go
//
// Connects to the broker via WebSocket, executes commands (shell, files, process management),
// and optionally runs in brain-actuator mode (wake delivery only).
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/TheBotsters/botster-actuator-g/internal/actuator"
)

var version = "dev"

func usage() {
	fmt.Fprintf(os.Stderr, `actuator-g %s — Botster Actuator (Go)

Environment:
  SEKS_BROKER_URL          Broker URL (required)
  SEKS_BROKER_TOKEN        Agent token (required)
  EGO_BRAIN_MODE=1         Alias for --brain-actuator
  EGO_WEBHOOK_PORT         Webhook port for wake delivery

Options:
  --id <name>              Actuator ID (default: hostname)
  --cwd <path>             Working directory (default: cwd)
  --capabilities <c>       Comma-separated capabilities
  --brain-actuator         Brain-actuator mode (wake delivery only, no commands)
  --brain                  Deprecated alias for --brain-actuator
  --webhook-port <port>    Webhook port for wake delivery
  --webhook-token <token>  Webhook auth token
  --token-file <path>      Path to persist rotated tokens (read on startup)
  --version                Show version
  --help                   Show this help
`, version)
	os.Exit(1)
}

type opts struct {
	id            string
	cwd           string
	capabilities  []string
	brainActuator bool
	webhookPort   int
	webhookToken  string
	tokenFile     string
}

func parseArgs(args []string) opts {
	var o opts
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--id":
			i++
			if i < len(args) {
				o.id = args[i]
			}
		case "--cwd":
			i++
			if i < len(args) {
				o.cwd = args[i]
			}
		case "--capabilities":
			i++
			if i < len(args) {
				o.capabilities = strings.Split(args[i], ",")
			}
		case "--brain-actuator":
			o.brainActuator = true
		case "--brain":
			log.Println("[actuator] WARNING: --brain is deprecated, use --brain-actuator")
			o.brainActuator = true
		case "--webhook-port":
			i++
			if i < len(args) {
				port, err := strconv.Atoi(args[i])
				if err == nil {
					o.webhookPort = port
				}
			}
		case "--webhook-token":
			i++
			if i < len(args) {
				o.webhookToken = args[i]
			}
		case "--token-file":
			i++
			if i < len(args) {
				o.tokenFile = args[i]
			}
		case "--version":
			fmt.Println(version)
			os.Exit(0)
		case "--help", "-h":
			usage()
		}
	}
	return o
}

func main() {
	o := parseArgs(os.Args[1:])

	brokerURL := os.Getenv("SEKS_BROKER_URL")
	agentToken := os.Getenv("SEKS_BROKER_TOKEN")

	// If a token file exists, its contents override the env var
	if o.tokenFile != "" {
		if data, err := os.ReadFile(o.tokenFile); err == nil {
			fileToken := strings.TrimSpace(string(data))
			if fileToken != "" {
				log.Printf("[actuator] Loaded token from %s", o.tokenFile)
				agentToken = fileToken
			}
		}
	}

	if brokerURL == "" || agentToken == "" {
		fmt.Fprintln(os.Stderr, "Error: SEKS_BROKER_URL and SEKS_BROKER_TOKEN must be set")
		os.Exit(1)
	}

	// Environment overrides
	if os.Getenv("EGO_BRAIN_MODE") == "1" {
		o.brainActuator = true
	}
	if envPort := os.Getenv("EGO_WEBHOOK_PORT"); envPort != "" && o.webhookPort == 0 {
		if port, err := strconv.Atoi(envPort); err == nil {
			o.webhookPort = port
		}
	}

	// Defaults
	if o.id == "" {
		hostname, err := os.Hostname()
		if err != nil {
			o.id = "actuator"
		} else {
			o.id = hostname
		}
	}
	if o.cwd == "" {
		o.cwd, _ = os.Getwd()
	}

	config := actuator.Config{
		BrokerURL:     brokerURL,
		AgentToken:    agentToken,
		ActuatorID:    o.id,
		Capabilities:  o.capabilities,
		Cwd:           o.cwd,
		BrainActuator: o.brainActuator,
		WebhookPort:   o.webhookPort,
		WebhookToken:  o.webhookToken,
		TokenFile:     o.tokenFile,
	}

	if o.brainActuator {
		log.Printf("[actuator] Brain-actuator mode — webhook delivery to localhost:%d", o.webhookPort)
	}
	log.Printf("[actuator] Starting — broker: %s, id: %s, version: %s", brokerURL, o.id, version)

	a := actuator.New(config)
	a.Start()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("\n[actuator] Received %s, shutting down...", sig)
	a.Stop()
}
