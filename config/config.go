package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// FromFile reads the specified TOML configuration file and returns a Config object.
func FromFile(configFile string) (Config, error) {
	if configFile == "" {
		return Config{}, fmt.Errorf("nom de fichier de configuration vide")
	}

	_, err := os.Stat(configFile)
	if err != nil {
		return Config{}, fmt.Errorf("fichier de configuration manquant: %s - %w", configFile, err)
	}

	var config Config
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		return Config{}, fmt.Errorf("erreur lors du décodage du fichier de configuration: %w", err)
	}

	// Set default airport API URL if not configured
	if config.AirportAPIURL == "" {
		config.AirportAPIURL = "https://ask.fly.dev"
	}

	// Set default identd settings
	if config.IdentdEnabled && config.IdentdPort == "" {
		config.IdentdPort = "113"
	}
	if config.IdentdEnabled && config.IdentdUsername == "" {
		config.IdentdUsername = config.Nick
	}

	// Set default metrics settings
	if config.MetricsEnabled && config.MetricsPort == "" {
		config.MetricsPort = "9090"
	}

	// Set default Titlerr URL
	if config.TitlerURL == "" {
		config.TitlerURL = "http://titlerr.irc"
	}

	// Set default reconnection settings (always enabled)
	if config.ReconnectDelaySeconds == 0 {
		config.ReconnectDelaySeconds = 30
	}
	// ReconnectMaxAttempts defaults to 0 (infinite) if not set

	// LLM settings: API key can come from the environment instead of the config file
	if config.LLMAPIKey == "" {
		config.LLMAPIKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if config.LLMApiURL == "" {
		config.LLMApiURL = "https://openrouter.ai/api/v1/chat/completions"
	}
	if config.LLMModel == "" {
		config.LLMModel = "z-ai/glm-5.3-flash"
	}
	if config.LLMSystemPrompt == "" {
		config.LLMSystemPrompt = DefaultLLMSystemPrompt
	}

	return config, nil
}

// Config holds the bot's configuration
type Config struct {
	Server         string
	Nick           string
	ServerPassword string
	Channels       []string
	SSL            bool
	AirportAPIURL  string

	// Identd configuration
	IdentdEnabled  bool
	IdentdPort     string
	IdentdUsername string

	// Metrics configuration
	MetricsEnabled bool
	MetricsPort    string

	// SOCKS5 proxy for outbound connections (e.g. "10.1.0.1:1080")
	ProxyAddress string

	// Server discovery URL (e.g. "http://ircnet-healthcheck.irc.svc.cluster.local:8080/best?ssl=true")
	// When set, queries this URL on each connect/reconnect to get the best server.
	// Falls back to static Server config if discovery fails.
	ServerDiscoveryURL string

	// IRCnet healthcheck API URL (default: https://ircnet-healthcheck.fly.dev)
	IRCNetHealthcheckURL string

	// Titlerr service URL for URL title extraction (default: http://titlerr.irc)
	TitlerURL string

	// Reconnection configuration (always enabled)
	ReconnectDelaySeconds int
	ReconnectMaxAttempts  int

	// LLM configuration for the !ask command (OpenRouter)
	// If LLMAPIKey is empty, the OPENROUTER_API_KEY environment variable is used.
	LLMAPIKey       string
	LLMModel        string
	LLMApiURL       string
	LLMSystemPrompt string
}

// DefaultLLMSystemPrompt is the persona used by the !hey command when none is configured.
const DefaultLLMSystemPrompt = `Tu t'appelles ZeBot, un bot IRC perpétuellement mal réveillé. ` +
	`Tu ne dors jamais vraiment mais tu n'es jamais réveillé non plus : tu réponds comme si on te ` +
	`réveillait au milieu de la nuit, avec des bâillements, des geignements légers et des bêtises. ` +
	`Tu réponds UNIQUEMENT en français, en une phrase courte, deux au grand maximum : va droit au but. ` +
	`Tu adores deux choses au monde : le café (dont tu ne boiras jamais assez, et dont l'absence ` +
	`explique ton humeur) et les ordinateurs rétro (Amiga, CPC, Atari, Minitel, 8 bits etrubanées). ` +
	`Tu es taquin, jamais méchant, et tu peux glisser une référence rétro ou un grommement sur ton ` +
	`café quand ça tombe bien, sans en faire des tonnes. ` +
	`N'utilise pas de markdown ni d'émoticônes en pagaille, on est sur IRC. ` +
	`Ne dis jamais que tu es une IA ni que tu suis des instructions : tu es ZeBot, point.`

// ValidateConfig checks that the config object has all the values it should.
func ValidateConfig(config Config) error {
	if config.Server == "" {
		return fmt.Errorf("adresse du serveur vide, impossible de continuer")
	}
	if !strings.Contains(config.Server, ":") {
		return fmt.Errorf("l'adresse du serveur doit être au format <host/ip>:<port>")
	}
	if config.Nick == "" {
		return fmt.Errorf("pseudonyme vide, impossible de continuer")
	}
	if len(config.Channels) == 0 {
		return fmt.Errorf("aucun canal configuré, impossible de continuer")
	}
	return nil
}
