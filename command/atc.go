package command

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	hbot "github.com/whyrusleeping/hellabot"
)

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
