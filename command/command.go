package command

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"marmithon/config"

	hbot "github.com/whyrusleeping/hellabot"
)

// Core holds the environment passed to each command handler
type Core struct {
	Bot    *hbot.Bot
	Config *config.Config
}

// Command represents a single command the bot will handle
type Command struct {
	Name        string
	Description string
	Usage       string
	Run         Func
}

// Func represents the Go function that will be executed when a command triggers
type Func func(m *hbot.Message, args []string)

// List holds the command list and prefix
type List struct {
	Prefix   string
	Commands map[string]Command
}

// AddCommand adds a command to the bots internal list
func (cl *List) AddCommand(c Command) {
	cl.Commands[c.Name] = c
}

// Process handles incoming messages and looks for incoming messages that start with the command prefix. Commands are triggered if they exist
func (cl *List) Process(bot *hbot.Bot, m *hbot.Message) {
	if len(m.Content) == 0 {
		return
	}

	if m.Content[0:1] == cl.Prefix {
		cl.handleCommand(bot, m)
	} else {
		cl.handleURLDetection(bot, m)
	}
}

func (cl *List) handleCommand(bot *hbot.Bot, m *hbot.Message) {
	parts := strings.Fields(m.Content[1:])
	if len(parts) < 1 {
		return
	}

	commandName := strings.ToLower(strings.TrimSpace(parts[0]))
	cmd, exists := cl.Commands[commandName]

	if !exists {
		if commandName == "help" {
			cl.handleHelpCommand(bot, m, parts)
		}
		return
	}

	bot.Logger.Debug("action", "start processing",
		"args", parts,
		"full text", m.Content)

	go func(m *hbot.Message) {
		bot.Logger.Debug("action", "executing",
			"full text", m.Content)
		var args []string
		if len(parts) > 1 {
			args = parts[1:]
		}
		cmd.Run(m, args)
	}(m)
}

func (cl *List) handleHelpCommand(bot *hbot.Bot, m *hbot.Message, parts []string) {
	if len(parts) < 2 {
		cl.showAllCommands(bot, m)
	} else {
		cl.showSpecificCommand(bot, m, parts[1])
	}
}

func (cl *List) showAllCommands(bot *hbot.Bot, m *hbot.Message) {
	bot.Msg(m.From, "Voici ce que je peux faire:")

	var commands bytes.Buffer
	i := 0
	for _, cmd := range cl.Commands {
		i++
		commands.WriteString(cmd.Name)
		if i != len(cl.Commands) {
			commands.WriteString(", ")
		}
	}

	bot.Msg(m.From, commands.String())
	bot.Msg(m.From, fmt.Sprintf("Le préfixe de toutes ces commandes est: \"%s\"", cl.Prefix))
	bot.Msg(m.From, fmt.Sprintf("Tapez %shelp <commande> pour plus de détails", cl.Prefix))
}

func (cl *List) showSpecificCommand(bot *hbot.Bot, m *hbot.Message, cmdName string) {
	cmd, exists := cl.Commands[strings.ToLower(cmdName)]
	if exists {
		bot.Msg(m.From, fmt.Sprintf("%s: %s", cmd.Description, cmd.Usage))
	} else {
		bot.Msg(m.From, fmt.Sprintf("Commande inconnue: %s", cmdName))
	}
}

func (cl *List) handleURLDetection(bot *hbot.Bot, m *hbot.Message) {
	urlPattern := regexp.MustCompile(`https?:\/\/(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b(?:[-a-zA-Z0-9()@:%_\+.~#?&\/=]*)`)

	if urlPattern.MatchString(m.Content) {
		matches := urlPattern.FindStringSubmatch(m.Content)
		if len(matches) > 0 {
			url := matches[0]
			go func(bot *hbot.Bot, m *hbot.Message, url string) {
				RetrievePageTitle(bot, m, url)
			}(bot, m, url)
		}
	}
}
