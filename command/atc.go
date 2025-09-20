package command

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	hbot "github.com/whyrusleeping/hellabot"
)

type Airport struct {
	ICAO    string `json:"icao_code"`
	Name    string `json:"name"`
	Country string `json:"iso_country"`
}

type APIResponse struct {
	Airports []Airport `json:"airports"`
	Count    int       `json:"count"`
}

type DistanceResponse struct {
	Distance float64 `json:"distance_nm"`
}

func (core Core) SearchForOACI(m *hbot.Message, args []string) {
	if len(args) == 0 {
		core.Bot.Reply(m, "Dites moi au moins qqchose sur cet aéroport")
		return
	}

	searchingFor := args[0]
	countryLimiter := ""
	if len(args) == 2 {
		countryLimiter = args[1]
	}

	airports, err := core.searchAirports(searchingFor, countryLimiter, 10)
	if err != nil {
		core.Bot.Reply(m, fmt.Sprintf("Désolé, erreur lors de la recherche: %s", err.Error()))
		return
	}

	if len(airports) == 0 {
		core.Bot.Reply(m, "Désolé, je n'ai pas trouvé d'aéroports")
		return
	}

	// Limit to 10 results to avoid flooding
	limit := 10
	displayed := 0
	for _, airport := range airports {
		if displayed >= limit {
			break
		}
		core.Bot.Reply(m, fmt.Sprintf("%s : %s (%s)", airport.ICAO, airport.Name, airport.Country))
		displayed++
	}

	if len(airports) > limit {
		core.Bot.Reply(m, fmt.Sprintf("--- Total: %d (limit: %d)", len(airports), limit))
	} else {
		core.Bot.Reply(m, fmt.Sprintf("--- Total: %d", len(airports)))
	}
}

func (core Core) searchAirports(searchTerm, countryLimiter string, maxResults int) ([]Airport, error) {
	// Build the URL
	baseURL := fmt.Sprintf("%s/api/airport/search", core.Config.AirportAPIURL)
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("URL invalide: %w", err)
	}

	// Add query parameters
	q := u.Query()
	q.Set("name", searchTerm)
	if countryLimiter != "" {
		q.Set("country", countryLimiter)
	}
	u.RawQuery = q.Encode()

	// Make the HTTP request
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la requête HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("réponse HTTP %d", resp.StatusCode)
	}

	// Parse the JSON response
	var response APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("erreur lors du décodage JSON: %w", err)
	}

	// Filter out airports without valid ICAO codes
	var validAirports []Airport
	for _, airport := range response.Airports {
		if len(airport.ICAO) == 4 && airport.Name != "" {
			validAirports = append(validAirports, airport)
		}
	}

	return validAirports, nil
}

func (core Core) CalculateDistance(m *hbot.Message, args []string) {
	if len(args) != 2 {
		core.Bot.Reply(m, "Usage: !distance <departure_ICAO> <destination_ICAO>")
		return
	}

	departure := strings.ToUpper(args[0])
	destination := strings.ToUpper(args[1])

	if len(departure) != 4 || len(destination) != 4 {
		core.Bot.Reply(m, "Les codes ICAO doivent contenir exactement 4 caractères")
		return
	}

	distance, err := core.getDistance(departure, destination)
	if err != nil {
		core.Bot.Reply(m, fmt.Sprintf("Erreur lors du calcul de la distance: %s", err.Error()))
		return
	}

	core.Bot.Reply(m, fmt.Sprintf("Distance entre %s et %s: %.0f milles nautiques", departure, destination, distance))
}

func (core Core) getDistance(departure, destination string) (float64, error) {
	baseURL := fmt.Sprintf("%s/api/airport/distance", core.Config.AirportAPIURL)
	u, err := url.Parse(baseURL)
	if err != nil {
		return 0, fmt.Errorf("URL invalide: %w", err)
	}

	q := u.Query()
	q.Set("departure", departure)
	q.Set("destination", destination)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return 0, fmt.Errorf("erreur lors de la requête HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("réponse HTTP %d", resp.StatusCode)
	}

	var response DistanceResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return 0, fmt.Errorf("erreur lors du décodage JSON: %w", err)
	}

	return response.Distance, nil
}
