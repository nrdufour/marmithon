package main

import (
	"flag"
	"fmt"
	"log"
	"marmithon/command"
	"marmithon/config"
	"os"

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
	flag.Parse()

	if err := run(); err != nil {
		log.Fatalf("Erreur fatale: %v", err)
	}
}

func run() error {
	conf, err := config.FromFile(*configFile)
	if err != nil {
		return err
	}

	if err := config.ValidateConfig(conf); err != nil {
		return err
	}

	bot, err := createBot(conf)
	if err != nil {
		return err
	}

	core = &command.Core{Bot: bot, Config: &conf}
	bot.AddTrigger(CommandTrigger)
	bot.Logger.SetHandler(log15.StdoutHandler)

	// Ensure data directory exists
	if err := os.MkdirAll("/data", 0755); err != nil {
		return fmt.Errorf("erreur lors de la création du répertoire /data: %w", err)
	}

	// Initialize the seen database
	if err := command.InitSeenDB("/data/seen.db"); err != nil {
		return fmt.Errorf("erreur lors de l'initialisation de la base seen: %w", err)
	}

	setupCommands()

	log.Println("Démarrage du bot...")
	bot.Run()
	log.Println("Arrêt du bot.")
	return nil
}

func createBot(conf config.Config) (*hbot.Bot, error) {
	options := func(bot *hbot.Bot) {
		bot.SSL = conf.SSL
		if conf.ServerPassword != "" {
			bot.Password = conf.ServerPassword
		}
		bot.Channels = conf.Channels
	}

	bot, err := hbot.NewBot(conf.Server, conf.Nick, options)
	if err != nil {
		return nil, err
	}
	return bot, nil
}

func setupCommands() {
	cmdList = &command.List{
		Prefix:   "!",
		Commands: make(map[string]command.Command),
	}

	cmdList.AddCommand(command.Command{
		Name:        "cve",
		Description: "Récupère des informations sur une CVE à partir de http://cve.circl.lu/",
		Usage:       "!cve CVE-2017-7494",
		Run:         core.GetCVE,
	})

	cmdList.AddCommand(command.Command{
		Name:        "convert",
		Description: "Effectue une conversion d'une mesure d'une unité à une autre",
		Usage:       "!convert 400 ft m, !convert pour la liste des unités connues",
		Run:         core.ConvertUnits,
	})

	cmdList.AddCommand(command.Command{
		Name:        "version",
		Description: "Affiche la version du bot",
		Usage:       "!version",
		Run:         core.ShowVersion,
	})

	cmdList.AddCommand(command.Command{
		Name:        "seen",
		Description: "Indique quand un utilisateur a été vu pour la dernière fois",
		Usage:       "!seen <pseudo>",
		Run:         core.Seen,
	})

	cmdList.AddCommand(command.Command{
		Name:        "icao",
		Description: "Trouve un aéroport par nom avec code pays optionnel",
		Usage:       "!icao lille FR",
		Run:         core.SearchForOACI,
	})
}
