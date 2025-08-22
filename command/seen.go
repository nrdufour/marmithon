package command

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	hbot "github.com/whyrusleeping/hellabot"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var seenDB *sql.DB

// InitSeenDB initializes the SQLite database for tracking user activity
func InitSeenDB(dbPath string) error {
	var err error
	seenDB, err = sql.Open("sqlite3", dbPath)
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

	lastSeen, err := time.Parse("2006-01-02 15:04:05", lastSeenStr)
	if err != nil {
		return time.Time{}, "", "", fmt.Errorf("erreur de parsing de la date: %w", err)
	}

	return lastSeen, channel, lastMessage, nil
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
func (core Core) Seen(m *hbot.Message, args []string) {
	if len(args) < 1 {
		core.Bot.Reply(m, "Dis-moi qui tu cherches ! Usage: !seen <pseudo>")
		return
	}

	targetNick := strings.TrimSpace(args[0])
	if targetNick == "" {
		core.Bot.Reply(m, "Le pseudo ne peut pas être vide !")
		return
	}

	// Check if the user is asking about themselves
	if strings.EqualFold(targetNick, m.From) {
		core.Bot.Reply(m, "Tu te cherches toi-même ? Regarde-toi dans un miroir !")
		return
	}

	// Check if the target user is currently present in the channel
	// We'll do a simple check by looking at recent activity (within last 5 minutes)
	lastSeen, channel, lastMessage, err := GetUserSeen(targetNick)
	if err != nil {
		if strings.Contains(err.Error(), "jamais vu") {
			core.Bot.Reply(m, GetRandomNotSeenResponse(targetNick))
		} else {
			core.Bot.Reply(m, "Erreur lors de la recherche, désolé !")
		}
		return
	}

	// Check if the user spoke recently (within 5 minutes = very likely still present)
	timeDiff := time.Now().Sub(lastSeen)
	if timeDiff < 5*time.Minute {
		core.Bot.Reply(m, GetRandomPresentResponse(targetNick))
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

	core.Bot.Reply(m, response)
}