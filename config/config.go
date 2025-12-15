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

	// Set default reconnection settings
	if config.ReconnectEnabled && config.ReconnectDelaySeconds == 0 {
		config.ReconnectDelaySeconds = 30
	}
	if config.ReconnectEnabled && config.ReconnectMaxAttempts == 0 {
		config.ReconnectMaxAttempts = 0 // 0 means infinite
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

	// Reconnection configuration
	ReconnectEnabled      bool
	ReconnectDelaySeconds int
	ReconnectMaxAttempts  int
}

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
