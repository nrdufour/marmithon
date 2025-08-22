package command

import (
	"context"
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
	if len(args) < 1 {
		core.Bot.Reply(m, "Veuillez me donner la CVE à retrouver")
		return
	}

	cve := strings.TrimSpace(strings.ToUpper(args[0]))
	if cve == "" {
		core.Bot.Reply(m, "La CVE ne peut pas être vide")
		return
	}

	if err := validateCVE(cve); err != nil {
		core.Bot.Reply(m, err.Error())
		return
	}

	cveInfo, err := fetchCVEInfo(cve)
	if err != nil {
		core.Bot.Reply(m, fmt.Sprintf("Erreur lors de la récupération de la CVE: %s", err.Error()))
		return
	}

	core.Bot.Reply(m, fmt.Sprintf("%s: %s", cve, cveInfo.Data.Summary))
	if len(cveInfo.Data.Refmap.Confirm) > 0 {
		core.Bot.Reply(m, fmt.Sprintf("Référence: %s", cveInfo.Data.Refmap.Confirm[0]))
	}
}

func (core Core) ShowVersion(m *hbot.Message, args []string) {
	core.Bot.Reply(m, fmt.Sprintf("Version: %s -- Dernier commit: %s",
		versioninfo.Short(),
		versioninfo.LastCommit))
}

func validateCVE(cve string) error {
	matched, err := regexp.MatchString(`^CVE-\d{4}-\d{4,}$`, cve)
	if err != nil {
		return fmt.Errorf("erreur de validation de la CVE: %w", err)
	}
	if !matched {
		return fmt.Errorf("%s n'est pas dans un format valide pour une CVE. Exemple: CVE-2017-7494", cve)
	}
	return nil
}

func fetchCVEInfo(cve string) (*cveResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := &http.Client{
		Timeout: time.Second * 10,
	}

	url := fmt.Sprintf("http://cve.circl.lu/api/cve/%s", cve)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("impossible de créer la requête: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Marmithon IRC Bot")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erreur de requête: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, fmt.Errorf("CVE %s introuvable", cve)
	case http.StatusOK:
		// Continue processing
	default:
		return nil, fmt.Errorf("code de réponse HTTP inattendu: %d", resp.StatusCode)
	}

	var cveResp cveResponse
	if err := json.NewDecoder(resp.Body).Decode(&cveResp); err != nil {
		return nil, fmt.Errorf("erreur de décodage JSON: %w", err)
	}

	return &cveResp, nil
}
