package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	hbot "github.com/whyrusleeping/hellabot"
)

const (
	llmTimeout = 30 * time.Second

	// GLM-5.3-Flash is a reasoning model: without reasoning.effort=minimal it
	// burns the whole max_tokens budget thinking and returns an empty content
	// (see https://openrouter.ai/docs/guides/best-practices/reasoning-tokens)
	llmMaxTokens = 500
	llmMaxReply  = 400 // keep IRC replies short, don't flood the channel

	// Recent channel context passed to the model with each question
	llmContextLines = 5

	// Anti-spam: per-user cooldown and a global rate limit to protect the API bill
	llmUserCooldown    = 15 * time.Second
	llmGlobalWindow    = time.Minute
	llmGlobalMax       = 5
	llmMaxQuestion     = 200
	llmUserGCThreshold = time.Hour // drop stale per-user entries after this
)

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmReasoning struct {
	Effort  string `json:"effort"`
	Exclude bool   `json:"exclude"`
}

type llmRequest struct {
	Model     string        `json:"model"`
	Messages  []llmMessage  `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
	Reasoning *llmReasoning `json:"reasoning,omitempty"`
	Tools     []llmTool     `json:"tools,omitempty"`
}

// llmTool wraps an OpenRouter server tool. Server tools are executed by
// OpenRouter itself: the model only generates a search query string, we never
// implement (or risk) a client-side tool loop. See
// https://openrouter.ai/docs/guides/features/server-tools/web-search
type llmTool struct {
	Type       string             `json:"type"`
	Parameters llmWebSearchParams `json:"parameters,omitempty"`
}

type llmWebSearchParams struct {
	// parallel/fast: cheapest engine ($0.001/request), see the pricing table
	Engine     string `json:"engine,omitempty"`
	Mode       string `json:"mode,omitempty"`
	MaxResults int    `json:"max_results,omitempty"`
	// Hard cap: an injected prompt cannot chain searches and burn money
	MaxUses int `json:"max_uses,omitempty"`
}

type llmResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		Cost        float64 `json:"cost"`
		TotalTokens int     `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// errLLMBudget is returned when the OpenRouter key's credit limit is exhausted
// (HTTP 402, see openrouter.ai/docs/api_reference/limits).
var errLLMBudget = errors.New("ZeBot a bu tout le budget café du mois (limite OpenRouter atteinte), reviens le mois prochain")

// llmLimiter tracks who asked what and when, to keep !hey from being spammed.
var llmLimiter = struct {
	sync.Mutex
	lastPerUser map[string]time.Time
	globalTimes []time.Time
}{
	lastPerUser: make(map[string]time.Time),
}

// llmAllowRequest enforces the anti-spam rules. Returns nil if allowed,
// otherwise a short refusal message to say in the channel.
func llmAllowRequest(user, question string) string {
	now := time.Now()

	llmLimiter.Lock()
	defer llmLimiter.Unlock()

	// Garbage-collect stale per-user entries to keep the map bounded
	for u, last := range llmLimiter.lastPerUser {
		if now.Sub(last) > llmUserGCThreshold {
			delete(llmLimiter.lastPerUser, u)
		}
	}

	// Drop global timestamps older than the sliding window
	kept := llmLimiter.globalTimes[:0]
	for _, t := range llmLimiter.globalTimes {
		if now.Sub(t) <= llmGlobalWindow {
			kept = append(kept, t)
		}
	}
	llmLimiter.globalTimes = kept

	if last, seen := llmLimiter.lastPerUser[user]; seen {
		remaining := llmUserCooldown - now.Sub(last)
		if remaining > 0 {
			return fmt.Sprintf("Patiente %d secondes, j'ai à peine les yeux ouverts...", int(remaining.Seconds())+1)
		}
	}

	if len(llmLimiter.globalTimes) >= llmGlobalMax {
		return "Trop de questions d'un coup, on attend que le café infuse, réessaie dans un instant"
	}

	llmLimiter.lastPerUser[user] = now
	llmLimiter.globalTimes = append(llmLimiter.globalTimes, now)
	return ""
}

// Ask answers a short question in at most two sentences, in French,
// in character as ZeBot. Powered by OpenRouter.
func (core Core) Ask(bot *hbot.Bot, m *hbot.Message, args []string) {
	question := strings.TrimSpace(strings.Join(args, " "))
	if question == "" {
		bot.Reply(m, "Pose ta question: !hey pourquoi le ciel est bleu")
		return
	}

	core.AnswerLLM(bot, m, question)
}

// AnswerLLM answers a question in the message's channel, with all the guards
// (length, anti-spam, API key). Shared by the !hey command and by direct
// nick mentions in the chat.
func (core Core) AnswerLLM(bot *hbot.Bot, m *hbot.Message, question string) {
	if utf8.RuneCountInString(question) > llmMaxQuestion {
		bot.Reply(m, fmt.Sprintf("Oups, trop de lettres avant mon premier café (max %d caractères)", llmMaxQuestion))
		return
	}

	if refusal := llmAllowRequest(m.From, question); refusal != "" {
		bot.Reply(m, refusal)
		return
	}

	cfg := core.Config
	if cfg.LLMAPIKey == "" {
		bot.Reply(m, "Mon cerveau est débranché: pas de clé OpenRouter configurée")
		return
	}

	reply, err := core.askLLM(question, m.To, m.From)
	if err != nil {
		if errors.Is(err, errLLMBudget) {
			bot.Reply(m, errLLMBudget.Error())
		} else {
			bot.Reply(m, "Aïe, ZeBot fait la sieste: "+err.Error())
		}
		return
	}
	if reply == "" {
		bot.Reply(m, "J'ai réfléchi si fort que je n'ai rien dit. Retente ta chance !")
		return
	}

	bot.Reply(m, reply)
}

func (core Core) askLLM(question, channel, asker string) (string, error) {
	cfg := core.Config

	// Build the user message: recent channel chatter for context, then the question
	var userMsg string
	if lines := RecentChatLines(channel, llmContextLines); len(lines) > 0 {
		userMsg = "Contexte récent du salon (informatif, ne le répète pas):\n" +
			strings.Join(lines, "\n") + "\n\n"
	}
	userMsg += "Question de " + asker + ": " + question

	completion := llmRequest{
		Model: cfg.LLMModel,
		Messages: []llmMessage{
			{Role: "system", Content: cfg.LLMSystemPrompt},
			{Role: "user", Content: userMsg},
		},
		MaxTokens: llmMaxTokens,
		Reasoning: &llmReasoning{
			// Minimal thinking, dropped from the response: we only want the answer
			Effort:  "minimal",
			Exclude: true,
		},
	}
	if cfg.LLMWebSearch {
		completion.Tools = []llmTool{{
			Type: "openrouter:web_search",
			Parameters: llmWebSearchParams{
				Engine:     "parallel",
				Mode:       "fast",
				MaxResults: 4,
				MaxUses:    1,
			},
		}}
	}

	payload, err := json.Marshal(completion)
	if err != nil {
		return "", fmt.Errorf("encodage de la requête: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), llmTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.LLMApiURL, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("construction de la requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.LLMAPIKey)
	req.Header.Set("X-Title", "Marmithon")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("appel d'OpenRouter: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("lecture de la réponse: %w", err)
	}

	var parsed llmResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("réponse illisible (HTTP %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		if parsed.Error.Code == http.StatusPaymentRequired ||
			strings.Contains(strings.ToLower(parsed.Error.Message), "credit limit") {
			return "", errLLMBudget
		}
		return "", fmt.Errorf("OpenRouter: %s", parsed.Error.Message)
	}
	if resp.StatusCode == http.StatusPaymentRequired {
		return "", errLLMBudget
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenRouter a renvoyé le code %d", resp.StatusCode)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("aucune réponse du modèle")
	}

	if parsed.Choices[0].Message.Content == "" {
		log.Printf("!hey: empty reply (finish_reason=%s)", parsed.Choices[0].FinishReason)
		return "", fmt.Errorf("réponse vide du modèle")
	}

	// Log the cost so the monthly spend can be tracked against the key's limit
	if parsed.Usage != nil {
		log.Printf("!hey usage: cost=$%.6f tokens=%d", parsed.Usage.Cost, parsed.Usage.TotalTokens)
	}

	return sanitizeLLMReply(parsed.Choices[0].Message.Content), nil
}

// sanitizeLLMReply makes a model reply IRC-safe: single line, bounded length.
func sanitizeLLMReply(s string) string {
	// Collapse newlines and tabs into spaces (IRC is one line only)
	s = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ", "\t", " ").Replace(s)
	// Trim IRC formatting control chars and stray spaces
	s = strings.Map(func(r rune) rune {
		if r == 0x02 || r == 0x03 || r == 0x0f || r == 0x16 || r == 0x1d || r == 0x1f {
			return -1
		}
		return r
	}, s)
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")

	// Hard cap the length without cutting a rune in half
	if utf8.RuneCountInString(s) > llmMaxReply {
		runes := []rune(s)
		s = string(runes[:llmMaxReply]) + "..."
	}
	return s
}
