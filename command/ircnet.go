package command

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	hbot "github.com/whyrusleeping/hellabot"
)

type ircnetServer struct {
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	SSL           bool    `json:"ssl"`
	Score         float64 `json:"score"`
	Up            bool    `json:"up"`
	LatencyMs     float64 `json:"latency_ms"`
	ReliabilityPct float64 `json:"reliability_pct"`
}

// HelpIRCNet displays the top 3 best IRCnet servers from the healthcheck API.
func (core Core) HelpIRCNet(bot *hbot.Bot, m *hbot.Message, args []string) {
	apiURL := "https://ircnet-healthcheck.fly.dev"
	if core.Config.IRCNetHealthcheckURL != "" {
		apiURL = core.Config.IRCNetHealthcheckURL
	}

	servers, err := fetchIRCNetServers(apiURL + "/servers")
	if err != nil {
		bot.Reply(m, fmt.Sprintf("Erreur lors de la récupération des serveurs IRCnet: %s", err.Error()))
		return
	}

	if len(servers) == 0 {
		bot.Reply(m, "Aucun serveur IRCnet disponible pour le moment.")
		return
	}

	bot.Reply(m, "Top serveurs IRCnet:")
	limit := 3
	if len(servers) < limit {
		limit = len(servers)
	}
	for i := 0; i < limit; i++ {
		s := servers[i]
		ssl := ""
		if s.SSL {
			ssl = " [SSL]"
		}
		status := "UP"
		if !s.Up {
			status = "DOWN"
		}
		bot.Reply(m, fmt.Sprintf(" %d. %s:%d%s — score: %.1f, latence: %.0fms, fiabilité: %.0f%% (%s)",
			i+1, s.Host, s.Port, ssl, s.Score, s.LatencyMs, s.ReliabilityPct, status))
	}
}

func fetchIRCNetServers(url string) ([]ircnetServer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("impossible de créer la requête: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Marmithon IRC Bot")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur de requête: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("code de réponse HTTP inattendu: %d", resp.StatusCode)
	}

	var servers []ircnetServer
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		return nil, fmt.Errorf("erreur de décodage JSON: %w", err)
	}

	return servers, nil
}
