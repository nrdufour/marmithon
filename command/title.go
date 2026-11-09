package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	hbot "github.com/whyrusleeping/hellabot"
)

type TitleCache struct {
	mu    sync.RWMutex
	cache map[string]CacheEntry
}

type CacheEntry struct {
	title     string
	timestamp time.Time
}

var titleCache = &TitleCache{
	cache: make(map[string]CacheEntry),
}

const cacheExpiration = 1 * time.Hour

func RetrievePageTitle(bot *hbot.Bot, m *hbot.Message, url string, titlerURL string) {
	if url == "" {
		bot.Reply(m, "URL invalide fournie")
		return
	}

	// Check cache first
	if cachedTitle := getCachedTitle(url); cachedTitle != "" {
		bot.Reply(m, fmt.Sprintf("\x02%s \x0F\x0314[cache]", cachedTitle))
		return
	}

	// Only titlerr fetches pages, never the bot itself: the URL comes from
	// whoever posted it, and so does the server that sees the fetch. titlerr
	// fetches through passeur's web profile on mobilis; a direct fetch from
	// here would hand the poster the home IP. No titlerr, no title.
	if titlerURL == "" {
		return
	}
	title, err := fetchTitlerTitle(titlerURL, url)
	if err != nil {
		fmt.Printf("Titlerr error for %s: %s\n", url, err)
		return
	}
	title = cleanTitle(title)
	if title != "" {
		cacheTitle(url, title)
		bot.Reply(m, fmt.Sprintf("\x02%s", title))
	}
}

type titlerRequest struct {
	URL string `json:"url"`
}

type titlerResponse struct {
	URL    string `json:"url"`
	Title  string `json:"title"`
	Method string `json:"method"`
	Error  string `json:"error"`
}

func fetchTitlerTitle(titlerURL, url string) (string, error) {
	reqBody, err := json.Marshal(titlerRequest{URL: url})
	if err != nil {
		return "", fmt.Errorf("erreur d'encodage JSON: %w", err)
	}

	// Use a transport with no proxy to ensure internal cluster requests
	// don't get routed through any configured HTTP/SOCKS proxy
	client := http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	resp, err := client.Post(titlerURL+"/api/extract", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("erreur de requête titlerr: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("titlerr a retourné le code HTTP %d", resp.StatusCode)
	}

	var result titlerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("erreur de décodage de la réponse titlerr: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("erreur titlerr: %s", result.Error)
	}

	return result.Title, nil
}

func getCachedTitle(url string) string {
	titleCache.mu.RLock()
	defer titleCache.mu.RUnlock()

	entry, exists := titleCache.cache[url]
	if !exists || time.Since(entry.timestamp) > cacheExpiration {
		return ""
	}

	return entry.title
}

func cacheTitle(url, title string) {
	titleCache.mu.Lock()
	defer titleCache.mu.Unlock()

	titleCache.cache[url] = CacheEntry{
		title:     title,
		timestamp: time.Now(),
	}

	// Clean old entries periodically
	if len(titleCache.cache)%100 == 0 {
		go cleanExpiredCache()
	}
}

func cleanExpiredCache() {
	titleCache.mu.Lock()
	defer titleCache.mu.Unlock()

	now := time.Now()
	for url, entry := range titleCache.cache {
		if now.Sub(entry.timestamp) > cacheExpiration {
			delete(titleCache.cache, url)
		}
	}
}

func cleanTitle(title string) string {
	// Remove common HTML entities and clean up
	replacements := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": "\"",
		"&#39;":  "'",
		"&nbsp;": " ",
	}

	for entity, replacement := range replacements {
		title = strings.ReplaceAll(title, entity, replacement)
	}

	// Clean multiple spaces and trim
	spaceRegex := regexp.MustCompile(`\s+`)
	title = spaceRegex.ReplaceAllString(title, " ")
	title = strings.TrimSpace(title)

	// Limit length for IRC
	if len(title) > 300 {
		title = title[:297] + "..."
	}

	return title
}
