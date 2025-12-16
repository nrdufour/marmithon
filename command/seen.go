package command

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	hbot "github.com/whyrusleeping/hellabot"
)

var seenDB *sql.DB

// InitSeenDB initializes the SQLite database for tracking user activity
func InitSeenDB(dbPath string) error {
	var err error
	seenDB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("erreur lors de l'ouverture de la base de données: %w", err)
	}

	// Create the table if it doesn't exist
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS user_seen (
		nickname TEXT PRIMARY KEY,
		channel TEXT NOT NULL,
		last_seen_at DATETIME NOT NULL,
		last_message TEXT
	)`

	_, err = seenDB.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("erreur lors de la création de la table: %w", err)
	}

	return nil
}

// UpdateUserSeen updates the last seen time for a user
func UpdateUserSeen(nickname, channel, message string) error {
	if seenDB == nil {
		return fmt.Errorf("base de données non initialisée")
	}

	query := `
	INSERT OR REPLACE INTO user_seen (nickname, channel, last_seen_at, last_message)
	VALUES (?, ?, datetime('now'), ?)`

	_, err := seenDB.Exec(query, strings.ToLower(nickname), channel, message)
	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour de l'utilisateur: %w", err)
	}

	return nil
}

// GetUserSeen retrieves the last seen information for a user
func GetUserSeen(nickname string) (time.Time, string, string, error) {
	if seenDB == nil {
		return time.Time{}, "", "", fmt.Errorf("base de données non initialisée")
	}

	query := `SELECT channel, last_seen_at, last_message FROM user_seen WHERE nickname = ? COLLATE NOCASE`
	
	var channel, lastSeenStr, lastMessage string
	err := seenDB.QueryRow(query, strings.ToLower(nickname)).Scan(&channel, &lastSeenStr, &lastMessage)
	if err != nil {
		if err == sql.ErrNoRows {
			return time.Time{}, "", "", fmt.Errorf("utilisateur jamais vu")
		}
		return time.Time{}, "", "", fmt.Errorf("erreur lors de la récupération: %w", err)
	}

	// Try multiple time formats that SQLite might return
	var lastSeen time.Time
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000000",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
	}
	
	for _, format := range formats {
		if lastSeen, err = time.Parse(format, lastSeenStr); err == nil {
			break
		}
	}
	
	if err != nil {
		return time.Time{}, "", "", fmt.Errorf("erreur de parsing de la date: %w", err)
	}

	return lastSeen, channel, lastMessage, nil
}

// SearchUsersSeen retrieves users matching a wildcard pattern
func SearchUsersSeen(pattern string) ([]struct{
	Nickname string
	Channel string
	LastSeen time.Time
	LastMessage string
}, error) {
	if seenDB == nil {
		return nil, fmt.Errorf("base de données non initialisée")
	}

	// Convert shell-style wildcards to SQL LIKE patterns
	sqlPattern := strings.ReplaceAll(pattern, "*", "%")
	sqlPattern = strings.ReplaceAll(sqlPattern, "?", "_")
	
	query := `SELECT nickname, channel, last_seen_at, last_message FROM user_seen WHERE nickname LIKE ? COLLATE NOCASE ORDER BY last_seen_at DESC LIMIT 10`
	
	rows, err := seenDB.Query(query, strings.ToLower(sqlPattern))
	if err != nil {
		return nil, fmt.Errorf("erreur lors de la recherche: %w", err)
	}
	defer rows.Close()

	var results []struct{
		Nickname string
		Channel string
		LastSeen time.Time
		LastMessage string
	}

	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000000",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
	}

	for rows.Next() {
		var nickname, channel, lastSeenStr, lastMessage string
		err := rows.Scan(&nickname, &channel, &lastSeenStr, &lastMessage)
		if err != nil {
			continue
		}

		var lastSeen time.Time
		for _, format := range formats {
			if lastSeen, err = time.Parse(format, lastSeenStr); err == nil {
				break
			}
		}
		if err != nil {
			continue
		}

		results = append(results, struct{
			Nickname string
			Channel string
			LastSeen time.Time
			LastMessage string
		}{nickname, channel, lastSeen, lastMessage})
	}

	return results, nil
}

// GetRandomPresentResponse returns a random humorous response for when a user is currently present
func GetRandomPresentResponse(nickname string) string {
	responses := []string{
		fmt.Sprintf("Mais %s est là, regarde un peu mieux !", nickname),
		fmt.Sprintf("Tu as des problèmes de vue ? %s vient de parler !", nickname),
		fmt.Sprintf("%s est juste là, en train de se moquer de ta question probablement.", nickname),
		fmt.Sprintf("Sérieusement ? %s est présent, tu lui parles même pas ?", nickname),
		fmt.Sprintf("%s doit être en train de rigoler... il/elle est là maintenant !", nickname),
		fmt.Sprintf("Plot twist: %s lit ce message en ce moment même !", nickname),
		fmt.Sprintf("*regarde %s puis toi* Vraiment ?", nickname),
		fmt.Sprintf("%s est là, vivant et en pleine forme !", nickname),
	}
	return responses[rand.Intn(len(responses))]
}

// GetRandomNotSeenResponse returns a random humorous response for when a user hasn't been seen
func GetRandomNotSeenResponse(nickname string) string {
	responses := []string{
		fmt.Sprintf("Aucune idée où %s peut bien être, mystère et boule de gomme !", nickname),
		fmt.Sprintf("%s ? Jamais entendu parler. Tu es sûr que cette personne existe ?", nickname),
		fmt.Sprintf("%s s'est probablement évaporé dans la nature.", nickname),
		fmt.Sprintf("Introuvable, %s a dû partir à l'aventure !", nickname),
		fmt.Sprintf("%s ? Parti.e aux Bahamas peut-être ?", nickname),
		fmt.Sprintf("Aucune trace de %s. Parti.e en mission secrète ?", nickname),
		fmt.Sprintf("%s reste un mystère pour moi !", nickname),
	}
	return responses[rand.Intn(len(responses))]
}

// FormatTimeDifference formats the time difference in a human-readable French format
func FormatTimeDifference(then time.Time) string {
	now := time.Now()
	diff := now.Sub(then)

	days := int(diff.Hours() / 24)
	hours := int(diff.Hours()) % 24
	minutes := int(diff.Minutes()) % 60
	seconds := int(diff.Seconds()) % 60

	if days > 0 {
		if days == 1 {
			return fmt.Sprintf("il y a 1 jour")
		}
		return fmt.Sprintf("il y a %d jours", days)
	}

	if hours > 0 {
		if hours == 1 {
			return fmt.Sprintf("il y a 1 heure")
		}
		return fmt.Sprintf("il y a %d heures", hours)
	}

	if minutes > 0 {
		if minutes == 1 {
			return fmt.Sprintf("il y a 1 minute")
		}
		return fmt.Sprintf("il y a %d minutes", minutes)
	}

	if seconds < 10 {
		return "à l'instant"
	}

	return fmt.Sprintf("il y a %d secondes", seconds)
}

// Seen handles the !seen command
func (core Core) Seen(bot *hbot.Bot, m *hbot.Message, args []string) {
	if len(args) < 1 {
		bot.Reply(m, "Dis-moi qui tu cherches ! Usage: !seen <pseudo> (supports wildcards like dsp*)")
		return
	}

	targetNick := strings.TrimSpace(args[0])
	if targetNick == "" {
		bot.Reply(m, "Le pseudo ne peut pas être vide !")
		return
	}

	// Check if the user is asking about themselves
	if strings.EqualFold(targetNick, m.From) {
		bot.Reply(m, "Tu te cherches toi-même ? Regarde-toi dans un miroir !")
		return
	}

	// Check if this is a wildcard search
	if strings.Contains(targetNick, "*") || strings.Contains(targetNick, "?") {
		results, err := SearchUsersSeen(targetNick)
		if err != nil {
			bot.Reply(m, "Erreur lors de la recherche, désolé !")
			return
		}

		if len(results) == 0 {
			bot.Reply(m, fmt.Sprintf("Aucun utilisateur trouvé correspondant à '%s'", targetNick))
			return
		}

		if len(results) == 1 {
			// Single result, format like a normal !seen response
			result := results[0]
			timeDiff := time.Now().Sub(result.LastSeen)
			if timeDiff < 5*time.Minute {
				bot.Reply(m, GetRandomPresentResponse(result.Nickname))
				return
			}

			timeDiffStr := FormatTimeDifference(result.LastSeen)
			response := fmt.Sprintf("%s a été vu.e pour la dernière fois sur %s %s",
				result.Nickname, result.Channel, timeDiffStr)

			if len(result.LastMessage) > 0 && len(result.LastMessage) < 100 {
				response += fmt.Sprintf(" en disant: \"%s\"", result.LastMessage)
			}
			bot.Reply(m, response)
			return
		}

		// Multiple results, show a summary
		var responseLines []string
		responseLines = append(responseLines, fmt.Sprintf("Utilisateurs correspondant à '%s':", targetNick))

		for i, result := range results {
			if i >= 5 { // Limit to first 5 results
				responseLines = append(responseLines, fmt.Sprintf("... et %d autre(s)", len(results)-i))
				break
			}
			timeDiffStr := FormatTimeDifference(result.LastSeen)
			responseLines = append(responseLines, fmt.Sprintf("• %s sur %s %s", result.Nickname, result.Channel, timeDiffStr))
		}

		bot.Reply(m, strings.Join(responseLines, " "))
		return
	}

	// Standard exact search
	lastSeen, channel, lastMessage, err := GetUserSeen(targetNick)
	if err != nil {
		if strings.Contains(err.Error(), "jamais vu") {
			bot.Reply(m, GetRandomNotSeenResponse(targetNick))
		} else {
			bot.Reply(m, "Erreur lors de la recherche, désolé !")
		}
		return
	}

	// Check if the user spoke recently (within 5 minutes = very likely still present)
	timeDiff := time.Now().Sub(lastSeen)
	if timeDiff < 5*time.Minute {
		bot.Reply(m, GetRandomPresentResponse(targetNick))
		return
	}

	// Format the response with time and location
	timeDiffStr := FormatTimeDifference(lastSeen)
	response := fmt.Sprintf("%s a été vu.e pour la dernière fois sur %s %s",
		targetNick, channel, timeDiffStr)

	// Add a snippet of their last message if it's not too long
	if len(lastMessage) > 0 && len(lastMessage) < 100 {
		response += fmt.Sprintf(" en disant: \"%s\"", lastMessage)
	}

	bot.Reply(m, response)
}