package main

import (
	"flag"
	"fmt"
	"log"
	"marmithon/command"
	"marmithon/config"
	"marmithon/identd"
	"marmithon/metrics"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		if m := metrics.Get(); m != nil {
			m.IncMessagesReceived()
		}
		cmdList.Process(bot, m)
		return false
	},
}

// MetricsTrigger tracks all messages for metrics
var MetricsTrigger = hbot.Trigger{
	Condition: func(bot *hbot.Bot, m *hbot.Message) bool {
		return true
	},
	Action: func(bot *hbot.Bot, m *hbot.Message) bool {
		met := metrics.Get()
		if met != nil {
			// Track channel joins
			if m.Command == "JOIN" && m.Name == bot.Nick {
				met.AddChannel(m.To)
			}
			// Track channel parts
			if m.Command == "PART" && m.Name == bot.Nick {
				met.RemoveChannel(m.To)
			}
		}
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

	// Initialize metrics if enabled
	var met *metrics.Metrics
	var metricsSrv *metrics.Server
	if conf.MetricsEnabled {
		met = metrics.Init()
		metricsSrv = metrics.NewServer(conf.MetricsPort)
		if err := metricsSrv.Start(); err != nil {
			return fmt.Errorf("failed to start metrics server: %w", err)
		}
		defer metricsSrv.Stop()
		log.Printf("Metrics enabled on port %s", conf.MetricsPort)
	}

	// Start identd server if enabled
	var identdSrv *identd.Server
	if conf.IdentdEnabled {
		identdSrv = identd.New(conf.IdentdPort, conf.IdentdUsername)
		if err := identdSrv.Start(); err != nil {
			return fmt.Errorf("failed to start identd server: %w", err)
		}
		defer identdSrv.Stop()
		log.Printf("Identd enabled on port %s with username %s", conf.IdentdPort, conf.IdentdUsername)
	}

	// Determine data directory with fallback
	dataDir := "/data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Printf("Cannot create %s directory, falling back to /tmp: %v", dataDir, err)
		dataDir = "/tmp"
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return fmt.Errorf("erreur lors de la création du répertoire de fallback %s: %w", dataDir, err)
		}
	}

	// Initialize the seen database
	dbPath := dataDir + "/seen.db"
	if err := command.InitSeenDB(dbPath); err != nil {
		return fmt.Errorf("erreur lors de l'initialisation de la base seen: %w", err)
	}

	// Initialize core with config (bot will be set later)
	core = &command.Core{Config: &conf}
	setupCommands()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run bot with reconnection logic
	if conf.ReconnectEnabled {
		return runWithReconnect(conf, sigChan, met)
	}

	// Run bot without reconnection
	bot, err := createAndStartBot(conf, met)
	if err != nil {
		return err
	}

	// Wait for shutdown signal
	<-sigChan
	log.Println("Arrêt du bot...")
	bot.Close()
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

func createAndStartBot(conf config.Config, met *metrics.Metrics) (*hbot.Bot, error) {
	bot, err := createBot(conf)
	if err != nil {
		return nil, err
	}

	// Update the bot reference in the existing core
	core.Bot = bot
	bot.AddTrigger(CommandTrigger)
	if met != nil {
		bot.AddTrigger(MetricsTrigger)
	}
	bot.Logger.SetHandler(log15.StdoutHandler)

	if met != nil {
		met.SetConnected(true)
	}

	return bot, nil
}

func runWithReconnect(conf config.Config, sigChan chan os.Signal, met *metrics.Metrics) error {
	attempts := 0
	maxAttempts := conf.ReconnectMaxAttempts

	for {
		attempts++
		if maxAttempts > 0 && attempts > maxAttempts {
			return fmt.Errorf("nombre maximum de tentatives de reconnexion atteint (%d)", maxAttempts)
		}

		if attempts > 1 {
			if met != nil {
				met.IncReconnects()
				met.SetConnected(false)
			}
			delay := time.Duration(conf.ReconnectDelaySeconds) * time.Second
			log.Printf("Tentative de reconnexion %d dans %v...", attempts, delay)
			time.Sleep(delay)
		}

		bot, err := createAndStartBot(conf, met)
		if err != nil {
			log.Printf("Erreur de création du bot: %v", err)
			continue
		}

		log.Printf("Démarrage du bot (tentative %d)...", attempts)

		// Run bot in a goroutine and track when it exits
		botDone := make(chan struct{})
		go func() {
			bot.Run()
			close(botDone)
			if met != nil {
				met.SetConnected(false)
			}
		}()

		// Wait for either bot exit or shutdown signal
		select {
		case <-botDone:
			log.Println("Connexion perdue, tentative de reconnexion...")
			continue
		case <-sigChan:
			log.Println("Signal d'arrêt reçu, fermeture...")
			bot.Close()
			// Wait a bit for clean shutdown
			select {
			case <-botDone:
			case <-time.After(5 * time.Second):
			}
			return nil
		}
	}
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

	cmdList.AddCommand(command.Command{
		Name:        "distance",
		Description: "Calcule la distance entre deux aéroports en codes ICAO",
		Usage:       "!distance LFLL EGLL",
		Run:         core.CalculateDistance,
	})
}
