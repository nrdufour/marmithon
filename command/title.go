package command

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"

	hbot "github.com/whyrusleeping/hellabot"
	"golang.org/x/net/html"
)

func isTitleElement(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "title"
}

func traverse(n *html.Node) (string, bool) {
	if isTitleElement(n) {
		return n.FirstChild.Data, true
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result, ok := traverse(c)
		if ok {
			return result, ok
		}
	}

	return "", false
}

func GetHtmlTitle(r io.Reader) (string, bool) {
	doc, err := html.Parse(r)
	if err != nil {
		panic("Fail to parse html")
	}

	return traverse(doc)
}

func DisplayHTMLTitle(bot *hbot.Bot, m *hbot.Message, url string) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		bot.Reply(m, fmt.Sprintf("error with the cookiejar"))
	}

	client := http.Client{
		Jar: jar,
	}

	resp, err := client.Get(url)
	if err != nil {
		bot.Reply(m, fmt.Sprintf("Http get error: %s", err))
	}
	defer resp.Body.Close()

	if title, ok := GetHtmlTitle(resp.Body); ok {
		bot.Reply(m, fmt.Sprintf("\x02%s", title))
	}
}
