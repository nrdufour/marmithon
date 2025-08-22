package command

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	hbot "github.com/whyrusleeping/hellabot"
)

type Airport struct {
	OACI    string
	Name    string
	Country string
}

func ContainsI(a string, b string) bool {
	return strings.Contains(
		strings.ToLower(a),
		strings.ToLower(b),
	)
}

func (core Core) SearchForOACI(m *hbot.Message, args []string) {
	if len(args) == 0 {
		core.Bot.Reply(m, "Dites moi au moins qqchose sur cet aéroport")
		return
	}

	searchingFor := args[0]
	countryLimiter := ""
	limit := 5

	if len(args) == 2 {
		countryLimiter = args[1]
	}

	f, err := os.Open("/data/airports.csv")
	if err != nil {
		core.Bot.Reply(m, "Désolé, je ne peux lire ma base d'aéroports!")
		return
	}
	defer f.Close()

	r := csv.NewReader(f)
	counter := 0
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			core.Bot.Reply(m, "Désolé, je ne peux lire ma base d'aéroports!")
			return
		}

		oaci := record[1]
		name := record[3]
		country := record[8]

		if ContainsI(name, searchingFor) && len(oaci) == 4 {
			if countryLimiter != "" {
				if country != countryLimiter {
					continue
				}
			}
			if counter <= limit {
				core.Bot.Reply(m, fmt.Sprintf("%s : %s (%s)", oaci, name, country))
			}
			counter++
		}
	}

	if counter == 0 {
		core.Bot.Reply(m, "Désolé, je n'ai pas trouvé d'aéroports")
	} else {
		if counter > limit {
			core.Bot.Reply(m, fmt.Sprintf("--- Total: %d (limit: %d)", counter, limit))
		} else {
			core.Bot.Reply(m, fmt.Sprintf("--- Total: %d", counter))
		}
	}
}

func searchAirports(searchTerm, countryLimiter string, maxResults int) ([]Airport, error) {
	f, err := os.Open("/data/airports.csv")
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir la base de données d'aéroports: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	var airports []Airport

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("erreur lors de la lecture de la base de données: %w", err)
		}

		if len(record) < 9 {
			continue
		}

		oaci := strings.TrimSpace(record[1])
		name := strings.TrimSpace(record[3])
		country := strings.TrimSpace(record[8])

		if len(oaci) != 4 || name == "" {
			continue
		}

		if !ContainsI(name, searchTerm) {
			continue
		}

		if countryLimiter != "" && strings.ToUpper(country) != countryLimiter {
			continue
		}

		airports = append(airports, Airport{
			OACI:    oaci,
			Name:    name,
			Country: country,
		})
	}

	return airports, nil
}
