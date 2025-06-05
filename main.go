package main

import (
	"flag"
	"log"
	"marmithon/command"
	"marmithon/config"

	hbot "github.com/whyrusleeping/hellabot"
	log15 "gopkg.in/inconshreveable/log15.v2"
)

// Flags for passing arguments to the program
var configFile = flag.String("config", "production.toml", "path to config file")

// core holds the command environment (bot connection and db)
var core *command.Core

// cmdList holds our command list, which tells the bot what to respond to.
var cmdList *command.List

// CommandTrigger passes all incoming messages to the commandList parser.
var CommandTrigger = hbot.Trigger{
	Condition: func(bot *hbot.Bot, m *hbot.Message) bool {
		return m.Command == "PRIVMSG"
	},
	Action: func(bot *hbot.Bot, m *hbot.Message) bool {
		cmdList.Process(bot, m)
		return false
	},
}

// Main method
func main() {
	// Parse flags, this is needed for the flag package to work.
	// See https://godoc.org/flag
	flag.Parse()
	// Read the TOML Config
	conf := config.FromFile(*configFile)
	// Validate the config to see it's not missing anything vital.
	config.ValidateConfig(conf)

	// Setup our options anonymous function.. This gets called on the hbot.Bot object internally, applying the options inside.
	options := func(bot *hbot.Bot) {
		bot.SSL = conf.SSL
		if conf.ServerPassword != "" {
			bot.Password = conf.ServerPassword
		}
		bot.Channels = conf.Channels
	}
	// Create a new instance of hbot.Bot
	bot, err := hbot.NewBot(conf.Server, conf.Nick, options)
	if err != nil {
		log.Fatal(err)
	}
	// Setup the command environment
	core = &command.Core{Bot: bot, Config: &conf}
	// Add the command trigger (this is what triggers all command handling)
	bot.AddTrigger(CommandTrigger)
	// Set the default bot logger to stdout
	bot.Logger.SetHandler(log15.StdoutHandler)
	// Initialize the command list
	cmdList = &command.List{
		Prefix:   "!",
		Commands: make(map[string]command.Command),
	}
	// Add commands to handle
	// cmdList.AddCommand(command.Command{
	// 	Name:        "kudos",
	// 	Description: "Remercie une personne '!kudos <nickname>'",
	// 	Usage:       "!kudos CapNemo",
	// 	Run:         core.Kudos,
	// })
	cmdList.AddCommand(command.Command{
		Name:        "cve",                                                                    // Trigger word
		Description: "Récupere des informations sur une CVE à partir de http://cve.circl.lu/", // Description
		Usage:       "!cve CVE-2017-7494",                                                     // Usage example
		Run:         core.GetCVE,                                                              // Function or method to run when it triggers
	})
	//cmdList.AddCommand(command.Command{
	//	Name:        "oaci",
	//	Description: "Trouve un aéroport '!oaci <nom usuel> [<code pays ISO>]'",
	//	Usage:       "!oaci lille FR",
	//	Run:         core.SearchForOACI,
	//})
	cmdList.AddCommand(command.Command{
		Name:        "convert",
		Description: "Effectue une conversion d'une mesure d'une unité à une autre '!convert <valeur> <unité d'origine> <unité voulue>'",
		Usage:       "!convert 400 ft m, !convert pour la liste des unités connues",
		Run:         core.ConvertUnits,
	})
	cmdList.AddCommand(command.Command{
		Name:        "version",
		Description: "Affiche la version du bot",
		Usage:       "!version",
		Run:         core.ShowVersion,
	})

	// Start up bot (blocks until disconnect)
	bot.Run()
	log.Println("Bot shutting down.")
}
