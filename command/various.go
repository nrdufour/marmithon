package command

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/earthboundkid/versioninfo/v2"
	hbot "github.com/whyrusleeping/hellabot"
)

// // Kudos sends a kudos to the target nick
// func (core Core) Kudos(m *hbot.Message, args []string) {
// 	if len(args) < 1 {
// 		core.Bot.Reply(m, "Dites moi qui je dois remercier !")
// 		return
// 	}
// 	teammate := args[0]
// 	core.Bot.Reply(m, fmt.Sprintf("Hey %s, merci d'être si génial !", teammate))
// }

type cveResponse struct {
	Data struct {
		Modified     string      `json:"Modified"`
		Published    string      `json:"Published"`
		Cvss         interface{} `json:"cvss"`
		Cwe          string      `json:"cwe"`
		ID           string      `json:"id"`
		LastModified string      `json:"last-modified"`
		Redhat       struct {
			Advisories []struct {
				Bugzilla struct {
					ID    string `json:"id"`
					Title string `json:"title"`
				} `json:"bugzilla"`
				Rhsa struct {
					ID       string `json:"id"`
					Released string `json:"released"`
					Severity string `json:"severity"`
					Title    string `json:"title"`
				} `json:"rhsa"`
			} `json:"advisories"`
			Rpms []string `json:"rpms"`
		} `json:"redhat"`
		References []string `json:"references"`
		Refmap     struct {
			Confirm []string `json:"confirm"`
		} `json:"refmap"`
		Summary                      string        `json:"summary"`
		VulnerableConfiguration      []interface{} `json:"vulnerable_configuration"`
		VulnerableConfigurationCpe22 []interface{} `json:"vulnerable_configuration_cpe_2_2"`
	} `json:"data"`
	Status string `json:"status"`
}

// GetCVE gets info about a CVE
func (core Core) GetCVE(m *hbot.Message, args []string) {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	if len(args) < 1 {
		core.Bot.Reply(m, "Veuillez me donner la CVE à retrouver")
		return
	}
	cve := args[0]

	cve = strings.ToUpper(cve)
	matched, err := regexp.MatchString("CVE-\\d{4}-\\d{4,}", cve)
	if err != nil {
		core.Bot.Reply(m, fmt.Sprintf("regexp error: %v", err))
		return
	}
	if !matched {
		core.Bot.Reply(m, fmt.Sprintf("Err! %v n'est pas dans un bon format pour une CVE. Exemple valide: CVE-2017-7494", cve))
		return
	}
	url := fmt.Sprintf("http://cve.circl.lu/api/cve/%s", cve)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		core.Bot.Reply(m, fmt.Sprintf("error creating new request: %v", err))
		return
	}
	req.Header.Add("Version", "1.1")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", "NorCERT likes you :)")
	resp, err := client.Do(req)
	if err != nil {
		core.Bot.Reply(m, fmt.Sprintf("client request error: %v", err))
		return
	}
	if resp.StatusCode == 404 {
		core.Bot.Reply(m, fmt.Sprintf("%v not found", cve))
		return
	}
	if resp.StatusCode != 200 {
		core.Bot.Reply(m, fmt.Sprintf("response status code not 200: %v", resp))
		return
	}

	var r cveResponse
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		core.Bot.Reply(m, fmt.Sprintf("json decode error: %v", err))
		return
	}
	core.Bot.Reply(m, fmt.Sprintf("%s: %s", cve, r.Data.Summary))
	if len(r.Data.Refmap.Confirm) > 0 {
		core.Bot.Reply(m, fmt.Sprintf("%v", r.Data.Refmap.Confirm[0]))
	}
}

func (core Core) ShowVersion(m *hbot.Message, args []string) {
	core.Bot.Reply(m, fmt.Sprintf("Version: %s -- Dernier commit: %s",
		versioninfo.Short(),
		versioninfo.LastCommit))
}
