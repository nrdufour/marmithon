package command

import (
	"fmt"

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

var (
	GitCommit = "dev"
	BuildTime = "unknown"
)

func (core Core) ShowVersion(bot *hbot.Bot, m *hbot.Message, args []string) {
	bot.Reply(m, fmt.Sprintf("Version: %s -- Build: %s", GitCommit, BuildTime))
}

// ShowStats gives the link to the channel statistics
func (core Core) ShowStats(bot *hbot.Bot, m *hbot.Message, args []string) {
	bot.Reply(m, "Statistiques du canal: https://souk.nemoworld.info")
}
