package command

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"marmitton/config"

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
	// Is the first character our command prefix?
	if m.Content[:1] == cl.Prefix {
		parts := strings.Fields(m.Content[1:])
		if len(parts) < 1 {
			return
		}
		commandstring := strings.ToLower(parts[0])
		cmd, ok := cl.Commands[commandstring]
		if !ok {
			if commandstring == "help" {
				if len(parts) < 2 {
					bot.Msg(m.From, "Voici ce que je peux faire:")
					var commands bytes.Buffer
					i := 0
					for _, cmd := range cl.Commands {
						i = i + 1
						commands.WriteString(cmd.Name)
						if i != len(cl.Commands) {
							commands.WriteString(", ")
						}
					}
					bot.Msg(m.From, commands.String())
					bot.Msg(m.From, fmt.Sprintf("Le préfixe de toutes ces commandes est: \"%s\"", cl.Prefix))
					bot.Msg(m.From, fmt.Sprintf("Tapez %shelp <commande> pour plus de détails", cl.Prefix))
				} else {
					helpcmd, helpok := cl.Commands[parts[1]]
					if helpok {
						bot.Msg(m.From, fmt.Sprintf("%s: %s", helpcmd.Description, helpcmd.Usage))
					} else {
						bot.Msg(m.From, fmt.Sprintf("Commande inconnue: %s", parts[1]))
					}
				}
			}
			return
		}
		// looks good, get the quote and reply with the result
		bot.Logger.Debug("action", "start processing",
			"args", parts,
			"full text", m.Content)
		go func(m *hbot.Message) {
			bot.Logger.Debug("action", "executing",
				"full text", m.Content)
			if len(parts) > 1 {
				cmd.Run(m, parts[1:])
			} else {
				cmd.Run(m, []string{})
			}
		}(m)
	} else {
		// Not a command
		URLPattern := regexp.MustCompile(`^.*(https?:\/\/(?:www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b(?:[-a-zA-Z0-9()@:%_\+.~#?&\/=]*)).*$`)
		// HACK: only match yahoo for now
		if URLPattern.MatchString(m.Content) && strings.Contains(m.Content, "yahoo") {
			results := URLPattern.FindAllSubmatch([]byte(m.Content), -1)
			url := string(results[0][1])

			go func(bot *hbot.Bot, m *hbot.Message, url string) {
				DisplayHTMLTitle(bot, m, url)
			}(bot, m, url)
		}

	}
}
