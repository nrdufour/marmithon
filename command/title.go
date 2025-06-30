package command

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	hbot "github.com/whyrusleeping/hellabot"
)

func GetHtmlTitle(r io.Reader) (string, bool) {
  // Load the HTML document
  doc, err := goquery.NewDocumentFromReader(r)
  if err != nil {
    return "", false
  }

  // Find the review items
  title := doc.Find("title").Text()
  return title, true
}

func GetYoutubeTitle(r io.Reader) (string, bool) {
	title := "- error with title -"

	buf := new(strings.Builder)
	_, err := io.Copy(buf, r)
	if err != nil {
		return title, false
	}

	// fmt.Println("CONNNNTENNNT ---->")
	// fmt.Println(buf.String())

	re := regexp.MustCompile(`"videoPrimaryInfoRenderer":{"title":{"runs":\[{"text":"([^"]+)"}`)
	results := re.FindStringSubmatch(buf.String())

	// fmt.Println("RESULTS ---->")
	// fmt.Println(results)

	if len(results) >= 2 {
		// We found the damn title
		title = results[1] + " - Fucking Youtube"
	}

	return title, true
}

func RetrievePageTitle(bot *hbot.Bot, m *hbot.Message, url string) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		bot.Reply(m, "error with the cookiejar")
	}

	client := http.Client{
		Jar: jar,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		bot.Reply(m, fmt.Sprintf("Http get error: %s", err))
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		bot.Reply(m, fmt.Sprintf("Http get error: %s", err))
	}
	defer resp.Body.Close()

	// Testing if it's a bloody youtube video
	isYoutube, err := regexp.Match(`www.youtube.com\/watch\?v=`, []byte(url))
	if err != nil {
		bot.Reply(m, fmt.Sprintf("Error matching url: %s", err))
	}
	if isYoutube {
		if title, ok := GetYoutubeTitle(resp.Body); ok {
			bot.Reply(m, fmt.Sprintf("\x02%s", title))
		}
	} else {
		if title, ok := GetHtmlTitle(resp.Body); ok {
			bot.Reply(m, fmt.Sprintf("\x02%s", title))
		}
	}
}
