package command

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// Soukoscope profile injection for !hey: the #souk stats site
// (souk.nemoworld.info) publishes public per-regular pages with activity
// facts, signature words and social links. We fetch and parse them to give
// the model grounding about who is asking (and who is mentioned), then
// inject a compact profile block into the prompt.
//
// Only public aggregate stats are used - the site itself publishes no
// message content, so nothing new is exposed to the model.

const (
	soukFetchTimeout  = 5 * time.Second
	soukCacheTTL      = 6 * time.Hour  // stats change at most daily
	soukNegativeTTL   = 24 * time.Hour // remember nicks without a page
	soukIndexCacheTTL = 24 * time.Hour
	soukMaxExtraNicks = 2 // nicks mentioned in the question, beyond the asker
)

var soukHTTP = &http.Client{Timeout: soukFetchTimeout}

type soukEntry struct {
	profile string
	missing bool
	expires time.Time
}

type soukCache struct {
	sync.Mutex
	profiles map[string]soukEntry
	slugs    map[string]bool
	slugsExp time.Time
}

var souk = soukCache{profiles: make(map[string]soukEntry)}

// HTML patterns: the site's generated markup is regular but inserts newlines
// freely between tags, so all multi-tag patterns tolerate \s*.
// regexes are enough (no external HTML dependency).
var (
	soukFactsRe = regexp.MustCompile(`<tr><td>([^<]+)</td><td class="num">([^<]+)</td></tr>`)
	soukTrophRe = regexp.MustCompile(`<tr><td>([^<]+)</td><td class="num">([0-9]+)</td><td class="num">[^<]*</td></tr>`)
	soukPeerRe  = regexp.MustCompile(`<a href="/habitues/[^"]+/">([^<]+)</a> <span class="num">\(([0-9]+)\)</span>`)
	soukMotsRe  = regexp.MustCompile(`Mots signatures</h2>\s*<p>([^<]+)</p>`)
	soukRythRe  = regexp.MustCompile(`<th>matin</th><th>[^<]*</th><th>[^<]*</th><th>[^<]*</th></tr></thead>\s*<tbody><tr>\s*` +
		`<td class="num">([0-9]+) ?%</td>\s*<td class="num">([0-9]+) ?%</td>\s*<td class="num">([0-9]+) ?%</td>\s*<td class="num">([0-9]+) ?%</td>`)
	soukSlugRe = regexp.MustCompile(`/habitues/([a-z0-9_-]+)/`)
)

// SoukProfile returns a compact profile block for the asker and any channel
// regulars mentioned in the question, or "" when nothing useful was found.
// Failures are silent: !hey must work even when the site is unreachable.
func SoukProfile(baseURL, asker, question string) string {
	if baseURL == "" {
		return ""
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	var block strings.Builder
	seen := map[string]bool{}
	addProfile := func(nick string) {
		slug := soukSlug(nick)
		if slug == "" || seen[slug] {
			return
		}
		seen[slug] = true
		if prof := soukProfileFor(baseURL, nick, slug); prof != "" {
			if block.Len() > 0 {
				block.WriteString("\n")
			}
			block.WriteString(prof)
		}
	}

	addProfile(asker)

	// Nicks explicitly mentioned in the question (word must match a known slug)
	for _, nick := range soukMentionedNicks(baseURL, question) {
		if block.Len() > 0 && len(seen) > soukMaxExtraNicks {
			break
		}
		addProfile(nick)
	}

	out := block.String()
	if out == "" {
		return ""
	}
	return "Profils soukoscope (stats publiques du salon sur " + baseURL +
		"; informatif, ne les cite pas mot pour mot):\n" + out
}

// soukProfileFor returns "nick: facts..." for one habitué, cached.
func soukProfileFor(baseURL, nick, slug string) string {
	now := time.Now()

	souk.Lock()
	if e, ok := souk.profiles[slug]; ok && now.Before(e.expires) {
		souk.Unlock()
		if e.missing {
			return ""
		}
		return e.profile
	}
	souk.Unlock()

	page, err := soukFetch(baseURL + "/habitues/" + slug + "/")
	missing := err != nil || page == ""
	prof := ""
	if !missing {
		prof = soukParseProfile(nick, page)
		missing = prof == ""
	}

	souk.Lock()
	ttl := soukCacheTTL
	if missing {
		ttl = soukNegativeTTL
	}
	souk.profiles[slug] = soukEntry{profile: prof, missing: missing, expires: now.Add(ttl)}
	souk.Unlock()

	return prof
}

// soukMentionedNicks finds question words that match known habitué slugs.
func soukMentionedNicks(baseURL, question string) []string {
	souk.Lock()
	expired := souk.slugs == nil || time.Now().After(souk.slugsExp)
	souk.Unlock()

	if expired {
		index, err := soukFetch(baseURL + "/habitues/")
		if err != nil {
			return nil
		}
		slugs := map[string]bool{}
		for _, m := range soukSlugRe.FindAllStringSubmatch(index, -1) {
			slugs[m[1]] = true
		}
		souk.Lock()
		souk.slugs = slugs
		souk.slugsExp = time.Now().Add(soukIndexCacheTTL)
		souk.Unlock()
	}

	souk.Lock()
	slugs := souk.slugs
	souk.Unlock()
	if slugs == nil {
		return nil
	}

	var found []string
	for _, tok := range strings.FieldsFunc(question, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_')
	}) {
		slug := strings.ToLower(tok)
		if len(slug) >= 3 && slugs[slug] {
			found = append(found, tok)
		}
	}
	if len(found) > soukMaxExtraNicks {
		found = found[:soukMaxExtraNicks]
	}
	return found
}

// soukSlug normalizes an IRC nick to a soukoscope URL slug.
func soukSlug(nick string) string {
	nick = strings.TrimLeft(nick, "@+~&")
	slug := url.PathEscape(strings.ToLower(nick))
	if slug == "" || !utf8.ValidString(slug) {
		return ""
	}
	return slug
}

func soukFetch(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), soukFetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "marmithon/0.1 (+https://souk.nemoworld.info)")
	resp, err := soukHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("soukoscope: HTTP %d pour %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// soukParseProfile extracts the interesting bits of an habitué page into one
// compact line. Unknown layout -> "" (caller treats as missing).
func soukParseProfile(nick, page string) string {
	var parts []string

	// Activity facts
	facts := map[string]string{}
	for _, m := range soukFactsRe.FindAllStringSubmatch(page, -1) {
		facts[strings.TrimSpace(m[1])] = strings.TrimSpace(m[2])
	}
	var act strings.Builder
	if v, ok := facts["jours actifs"]; ok {
		act.WriteString("actif " + v + " jours")
	}
	if v, ok := facts["première ligne"]; ok {
		act.WriteString(" (première ligne " + v)
		if v2, ok2 := facts["dernière ligne"]; ok2 {
			act.WriteString(", dernière " + v2)
		}
		act.WriteString(")")
	}
	if act.Len() > 0 {
		parts = append(parts, act.String())
	}

	// Day-part rhythm: report the two strongest parts
	if m := soukRythRe.FindStringSubmatch(page); m != nil {
		labels := []string{"le matin", "l'après-midi", "le soir", "la nuit"}
		vals := []int{}
		for _, s := range m[1:] {
			vals = append(vals, atoiSafe(s))
		}
		idx := []int{0, 1, 2, 3}
		for i := 0; i < len(idx); i++ {
			for j := i + 1; j < len(idx); j++ {
				if vals[idx[j]] > vals[idx[i]] {
					idx[i], idx[j] = idx[j], idx[i]
				}
			}
		}
		parts = append(parts, fmt.Sprintf("plutôt %s (%d%%) et %s (%d%%)",
			labels[idx[0]], vals[idx[0]], labels[idx[1]], vals[idx[1]]))
	}

	// Signature words
	if m := soukMotsRe.FindStringSubmatch(page); m != nil {
		words := strings.Split(m[1], ", ")
		for i, w := range words {
			if j := strings.Index(w, " ("); j > 0 {
				words[i] = w[:j]
			}
		}
		if len(words) > 5 {
			words = words[:5]
		}
		parts = append(parts, "mots signatures: "+strings.Join(words, ", "))
	}

	// Social links: top "parle à"
	if m := soukPeerRe.FindAllStringSubmatch(page, -1); len(m) > 0 {
		var peers []string
		for i, p := range m {
			if i >= 3 {
				break
			}
			peers = append(peers, fmt.Sprintf("%s (%s)", p[1], p[2]))
		}
		parts = append(parts, "parle surtout à: "+strings.Join(peers, ", "))
	}

	// Trophies (rank n°1..n)
	if t := soukTrophRe.FindAllStringSubmatch(page, -1); len(t) > 0 {
		var trophies []string
		for i, tr := range t {
			if i >= 3 {
				break
			}
			trophies = append(trophies, fmt.Sprintf("%s (n°%s)", tr[1], tr[2]))
		}
		parts = append(parts, "trophées: "+strings.Join(trophies, ", "))
	}

	if len(parts) == 0 {
		return ""
	}
	return nick + ": " + strings.Join(parts, "; ")
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}
