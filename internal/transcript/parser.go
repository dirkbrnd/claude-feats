package transcript

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Session holds all signals extracted from a single Claude Code transcript.
type Session struct {
	// Timing
	StartTime time.Time
	EndTime   time.Time

	// Messages
	UserMessages []string // raw text of each user turn
	Interrupted  bool     // transcript contains "Request interrupted by user"

	// File operations
	FilesEdited map[string]int // filepath → edit count
	Extensions  map[string]struct{}

	// Raw tool command text (from Bash tool calls)
	BashCommands []string

	// Flags parsed from bash commands
	HasForceFlag    bool // --force or -f in a bash command
	HasGitRevert    bool // git revert | reset | checkout HEAD~
	HasWorktreeAdd  bool
	HasGitPush      bool
	HasGitResetHard bool
	HasRmRf         bool
}

// Duration returns session wall time.
func (s *Session) Duration() time.Duration {
	if s.StartTime.IsZero() || s.EndTime.IsZero() {
		return 0
	}
	return s.EndTime.Sub(s.StartTime)
}

// AvgUserMessageLen returns average rune length of user messages.
func (s *Session) AvgUserMessageLen() float64 {
	if len(s.UserMessages) == 0 {
		return 0
	}
	total := 0
	for _, m := range s.UserMessages {
		total += len([]rune(strings.TrimSpace(m)))
	}
	return float64(total) / float64(len(s.UserMessages))
}

// ─── JSONL types ────────────────────────────────────────────────────────────

type rawEntry struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
	Content   json.RawMessage `json:"content"`
}

type rawMessage struct {
	Role    string           `json:"role"`
	Content []rawContentItem `json:"content"`
}

type rawContentItem struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`  // tool_use name
	Input json.RawMessage `json:"input"` // tool_use input
}

type rawToolInput struct {
	Command string `json:"command"`
	Path    string `json:"path"`
	// Edit / Write
	FilePath string `json:"file_path"`
}

// forceRe matches -f or --force as a standalone flag in a shell command.
var forceRe = regexp.MustCompile(`(?:^|\s)(?:--force|-f)(?:\s|$)`)
var triageRe = regexp.MustCompile(`(?m)^\d+\.\s+(?:fix|skip)`)

// Parse reads a Claude Code JSONL transcript and returns a Session.
func Parse(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := &Session{
		FilesEdited: make(map[string]int),
		Extensions:  make(map[string]struct{}),
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry rawEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		ts := parseTime(entry.Timestamp)
		if !ts.IsZero() {
			if s.StartTime.IsZero() || ts.Before(s.StartTime) {
				s.StartTime = ts
			}
			if ts.After(s.EndTime) {
				s.EndTime = ts
			}
		}

		switch entry.Type {
		case "user":
			parseUserEntry(s, entry)
		case "assistant":
			parseAssistantEntry(s, entry)
		}
	}

	return s, scanner.Err()
}

func parseUserEntry(s *Session, entry rawEntry) {
	var msg rawMessage
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return
	}
	for _, item := range msg.Content {
		switch item.Type {
		case "text":
			text := strings.TrimSpace(item.Text)
			if text != "" {
				s.UserMessages = append(s.UserMessages, text)
				if strings.Contains(text, "[Request interrupted by user]") {
					s.Interrupted = true
				}
			}
		case "tool_result":
			// user-side tool results; check for interruption text
			if strings.Contains(item.Text, "Request interrupted by user") {
				s.Interrupted = true
			}
		}
	}
}

func parseAssistantEntry(s *Session, entry rawEntry) {
	var msg rawMessage
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return
	}
	for _, item := range msg.Content {
		if item.Type != "tool_use" {
			continue
		}
		var inp rawToolInput
		_ = json.Unmarshal(item.Input, &inp)

		switch item.Name {
		case "Edit", "MultiEdit":
			fp := inp.FilePath
			if fp != "" {
				s.FilesEdited[fp]++
				addExt(s, fp)
			}
		case "Write":
			fp := inp.FilePath
			if fp != "" {
				s.FilesEdited[fp]++
				addExt(s, fp)
			}
		case "Read":
			if inp.FilePath != "" {
				addExt(s, inp.FilePath)
			}
		case "Bash":
			cmd := strings.TrimSpace(inp.Command)
			if cmd == "" {
				continue
			}
			s.BashCommands = append(s.BashCommands, cmd)
			lc := strings.ToLower(cmd)

			if forceRe.MatchString(cmd) {
				s.HasForceFlag = true
			}
			if strings.Contains(lc, "git revert") ||
				strings.Contains(lc, "git reset") ||
				regexp.MustCompile(`git\s+checkout\s+HEAD~`).MatchString(lc) {
				s.HasGitRevert = true
			}
			if strings.Contains(lc, "git reset --hard") {
				s.HasGitResetHard = true
			}
			if strings.Contains(lc, "git worktree add") {
				s.HasWorktreeAdd = true
			}
			if strings.Contains(lc, "git push") {
				s.HasGitPush = true
			}
			if strings.Contains(lc, "rm -rf") || strings.Contains(lc, "rm -fr") {
				s.HasRmRf = true
			}
		}
	}
}

func addExt(s *Session, fp string) {
	ext := strings.ToLower(filepath.Ext(fp))
	if ext != "" {
		s.Extensions[ext] = struct{}{}
	}
}

func parseTime(ts string) time.Time {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, ts); err == nil {
			return t
		}
	}
	return time.Time{}
}

// TriageCount returns the number of triage items in a user message, or 0.
func TriageCount(msg string) int {
	matches := triageRe.FindAllString(msg, -1)
	return len(matches)
}

// ContainsClaude checks if any bash command references claude-feats or the
// claude CLI (for ouroboros / inception detection).
func (s *Session) ContainsClaudeFeats() bool {
	for _, cmd := range s.BashCommands {
		if strings.Contains(cmd, "claude-feats") {
			return true
		}
	}
	for _, msg := range s.UserMessages {
		if strings.Contains(strings.ToLower(msg), "claude-feats") {
			return true
		}
	}
	return false
}

// InceptionSignal returns true if the session appears to be about building/modifying claude-feats.
func (s *Session) InceptionSignal() bool {
	keywords := []string{"claude-feats", "feat catalog", "feat detector", "unlock feat"}
	for _, msg := range s.UserMessages {
		lc := strings.ToLower(msg)
		for _, kw := range keywords {
			if strings.Contains(lc, kw) {
				return true
			}
		}
	}
	return false
}
