package command

import (
	"strings"
	"sync"
	"time"
)

// Rolling memory of recent channel chatter, used to give !hey some context.
// Memory only: nothing here is persisted.

const (
	chatCtxMaxLines   = 30                // kept per channel
	chatCtxMaxLineLen = 200               // truncate individual lines
	chatCtxStale      = 15 * time.Minute  // drop lines older than this
	chatCtxGCInterval = 5 * time.Minute
)

type chatLine struct {
	nick string
	text string
	at   time.Time
}

type chatContext struct {
	sync.Mutex
	lines  map[string][]chatLine
	lastGC time.Time
}

var chatCtx = chatContext{lines: make(map[string][]chatLine)}

// RecordChatLine stores a channel message for the !hey context.
// Commands (prefix !) are skipped: they are noise, not conversation.
func RecordChatLine(channel, nick, text string) {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasPrefix(text, "!") {
		return
	}
	if len(text) > chatCtxMaxLineLen {
		text = text[:chatCtxMaxLineLen]
	}

	now := time.Now()

	chatCtx.Lock()
	defer chatCtx.Unlock()

	chatCtx.gcLocked(now)
	lines := append(chatCtx.lines[channel], chatLine{nick: nick, text: text, at: now})
	if len(lines) > chatCtxMaxLines {
		lines = lines[len(lines)-chatCtxMaxLines:]
	}
	chatCtx.lines[channel] = lines
}

// RecentChatLines returns up to n recent messages of a channel as
// "nick: text" strings, oldest first. Stale lines are not returned.
func RecentChatLines(channel string, n int) []string {
	cutoff := time.Now().Add(-chatCtxStale)

	chatCtx.Lock()
	defer chatCtx.Unlock()

	all := chatCtx.lines[channel]
	if len(all) > n {
		all = all[len(all)-n:]
	}
	out := make([]string, 0, len(all))
	for _, l := range all {
		if l.at.Before(cutoff) {
			continue
		}
		out = append(out, l.nick+": "+l.text)
	}
	return out
}

// gcLocked drops stale lines and channels that have become empty.
// Callers must hold chatCtx.Lock.
func (c *chatContext) gcLocked(now time.Time) {
	if now.Sub(c.lastGC) < chatCtxGCInterval {
		return
	}
	c.lastGC = now
	for ch, lines := range c.lines {
		kept := lines[:0]
		for _, l := range lines {
			if now.Sub(l.at) <= chatCtxStale {
				kept = append(kept, l)
			}
		}
		if len(kept) == 0 {
			delete(c.lines, ch)
		} else {
			c.lines[ch] = kept
		}
	}
}
