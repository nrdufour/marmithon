package command

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
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
const maxContentLength = 1024 * 1024 // 1MB limit

func GetHtmlTitle(r io.Reader, contentType string) (string, error) {
	limitedReader := io.LimitReader(r, maxContentLength)
	
	doc, err := goquery.NewDocumentFromReader(limitedReader)
	if err != nil {
		return "", fmt.Errorf("impossible d'analyser le HTML: %w", err)
	}

	title := strings.TrimSpace(doc.Find("title").First().Text())
	if title == "" {
		// Try meta property="og:title"
		title = strings.TrimSpace(doc.Find(`meta[property="og:title"]`).AttrOr("content", ""))
	}
	if title == "" {
		// Try meta name="title"
		title = strings.TrimSpace(doc.Find(`meta[name="title"]`).AttrOr("content", ""))
	}

	if title != "" {
		title = cleanTitle(title)
	}

	return title, nil
}

type PlatformExtractor struct {
	name     string
	pattern  *regexp.Regexp
	extract  func(io.Reader) (string, error)
}

var platformExtractors = []PlatformExtractor{
	{"YouTube", regexp.MustCompile(`(?i)(www\.|m\.|music\.)?youtu(\.be|be\.com)`), GetYoutubeTitle},
	{"Vimeo", regexp.MustCompile(`(?i)(www\.)?vimeo\.com`), GetVimeoTitle},
	{"Dailymotion", regexp.MustCompile(`(?i)(www\.)?dailymotion\.com`), GetDailymotionTitle},
	{"Twitch", regexp.MustCompile(`(?i)(www\.)?twitch\.tv`), GetTwitchTitle},
}

func GetYoutubeTitle(r io.Reader) (string, error) {
	limitedReader := io.LimitReader(r, maxContentLength)
	buf := new(strings.Builder)
	_, err := io.Copy(buf, limitedReader)
	if err != nil {
		return "", fmt.Errorf("impossible de lire la réponse: %w", err)
	}

	content := buf.String()

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`"videoDetails":{[^}]*"title":"([^"]+)"`),
		regexp.MustCompile(`"videoPrimaryInfoRenderer":{"title":{"runs":\[{"text":"([^"]+)"}`),
		regexp.MustCompile(`<meta property="og:title" content="([^"]+)"`),
		regexp.MustCompile(`<title>([^<]+) - YouTube</title>`),
		regexp.MustCompile(`"title":{"simpleText":"([^"]+)"}}`),
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(content); len(matches) >= 2 {
			title := strings.TrimSpace(matches[1])
			if title != "" {
				return cleanTitle(title) + " - YouTube", nil
			}
		}
	}

	return "", fmt.Errorf("titre YouTube introuvable")
}

func GetVimeoTitle(r io.Reader) (string, error) {
	limitedReader := io.LimitReader(r, maxContentLength)
	buf := new(strings.Builder)
	_, err := io.Copy(buf, limitedReader)
	if err != nil {
		return "", fmt.Errorf("impossible de lire la réponse: %w", err)
	}

	content := buf.String()
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`<meta property="og:title" content="([^"]+)"`),
		regexp.MustCompile(`<title>([^<]+) on Vimeo</title>`),
		regexp.MustCompile(`"title":"([^"]+)"`),
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(content); len(matches) >= 2 {
			title := strings.TrimSpace(matches[1])
			if title != "" {
				return cleanTitle(title) + " - Vimeo", nil
			}
		}
	}

	return "", fmt.Errorf("titre Vimeo introuvable")
}

func GetDailymotionTitle(r io.Reader) (string, error) {
	limitedReader := io.LimitReader(r, maxContentLength)
	buf := new(strings.Builder)
	_, err := io.Copy(buf, limitedReader)
	if err != nil {
		return "", fmt.Errorf("impossible de lire la réponse: %w", err)
	}

	content := buf.String()
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`<meta property="og:title" content="([^"]+)"`),
		regexp.MustCompile(`<title>([^<]+) - Dailymotion</title>`),
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(content); len(matches) >= 2 {
			title := strings.TrimSpace(matches[1])
			if title != "" {
				return cleanTitle(title) + " - Dailymotion", nil
			}
		}
	}

	return "", fmt.Errorf("titre Dailymotion introuvable")
}

func GetTwitchTitle(r io.Reader) (string, error) {
	limitedReader := io.LimitReader(r, maxContentLength)
	buf := new(strings.Builder)
	_, err := io.Copy(buf, limitedReader)
	if err != nil {
		return "", fmt.Errorf("impossible de lire la réponse: %w", err)
	}

	content := buf.String()
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`<meta property="og:title" content="([^"]+)"`),
		regexp.MustCompile(`<title>([^<]+) - Twitch</title>`),
	}

	for _, pattern := range patterns {
		if matches := pattern.FindStringSubmatch(content); len(matches) >= 2 {
			title := strings.TrimSpace(matches[1])
			if title != "" {
				return cleanTitle(title) + " - Twitch", nil
			}
		}
	}

	return "", fmt.Errorf("titre Twitch introuvable")
}


func RetrievePageTitle(bot *hbot.Bot, m *hbot.Message, url string) {
	if url == "" {
		bot.Reply(m, "URL invalide fournie")
		return
	}

	// Check cache first
	if cachedTitle := getCachedTitle(url); cachedTitle != "" {
		bot.Reply(m, fmt.Sprintf("\x02%s \x0F\x0314[cache]", cachedTitle))
		return
	}

	title, err := fetchPageTitle(url)
	if err != nil {
		fmt.Printf("Title extraction error for %s: %s\n", url, err)
		return
	}

	if title != "" {
		cacheTitle(url, title)
		bot.Reply(m, fmt.Sprintf("\x02%s", title))
	}
}

func fetchPageTitle(url string) (string, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", fmt.Errorf("erreur de création du cookie jar: %w", err)
	}

	client := http.Client{
		Jar:     jar,
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("erreur de création de la requête: %w", err)
	}

	// Use a simple, honest user agent for all requests
	req.Header.Set("User-Agent", "Marmithon-TitleBot/1.0 (IRC Title Extractor)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("erreur HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("code de statut HTTP %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "text/html") {
		return "", fmt.Errorf("type de contenu non supporté: %s", contentType)
	}

	// Try platform-specific extractors first
	for _, extractor := range platformExtractors {
		if extractor.pattern.MatchString(url) {
			return extractor.extract(resp.Body)
		}
	}

	// Fall back to generic HTML title extraction
	return GetHtmlTitle(resp.Body, contentType)
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
