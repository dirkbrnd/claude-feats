package detector

import (
	"regexp"
	"strings"

	"github.com/dirkbrand/claude-feats/internal/store"
	"github.com/dirkbrand/claude-feats/internal/transcript"
)

var triageLineRe = regexp.MustCompile(`(?m)^\d+\.\s+(?i:fix|skip)`)

// Behavioral detects feats derived from session transcript content.
type Behavioral struct{}

func (Behavioral) Check(s *transcript.Session, _ *store.Progress) []string {
	var ids []string

	// :wq — user typed the vim quit command literally
	for _, msg := range s.UserMessages {
		if strings.TrimSpace(msg) == ":wq" {
			ids = append(ids, "vimreflex")
			break
		}
	}

	// oneword — avg user message < 10 chars
	if len(s.UserMessages) > 0 && s.AvgUserMessageLen() < 10 {
		ids = append(ids, "oneword")
	}

	// speedrun — session < 5 minutes
	if d := s.Duration(); d > 0 && d.Minutes() < 5 {
		ids = append(ids, "speedrun")
	}

	// triage — a message has 5+ numbered fix/skip lines
	for _, msg := range s.UserMessages {
		if len(triageLineRe.FindAllString(msg, -1)) >= 5 {
			ids = append(ids, "triage")
			break
		}
	}

	// interrupt — session was interrupted
	if s.Interrupted {
		ids = append(ids, "interrupt")
	}

	// oneshot — exactly one user message and session didn't end mid-sentence
	if len(s.UserMessages) == 1 {
		ids = append(ids, "oneshot")
	}

	// longhaul — session > 4 hours
	if d := s.Duration(); d.Hours() > 4 {
		ids = append(ids, "longhaul")
	}

	// perfectionist — same file edited 10+ times
	for _, count := range s.FilesEdited {
		if count >= 10 {
			ids = append(ids, "perfectionist")
			break
		}
	}

	return ids
}

// HiddenBehavioral detects hidden feats from session content.
type HiddenBehavioral struct{}

func (HiddenBehavioral) Check(s *transcript.Session, p *store.Progress) []string {
	var ids []string

	// ghostmode — 0 user messages
	if len(s.UserMessages) == 0 {
		ids = append(ids, "ghostmode")
	}

	// oops — git reset --hard or rm -rf
	if s.HasGitResetHard || s.HasRmRf {
		ids = append(ids, "oops")
	}

	// yolo — --force or -f flag used
	if s.HasForceFlag {
		ids = append(ids, "yolo")
	}

	// ouroboros — claude-feats invoked inside a CC session
	if s.ContainsClaudeFeats() {
		ids = append(ids, "ouroboros")
	}

	// inception — session is about building/modifying claude-feats
	if s.InceptionSignal() {
		ids = append(ids, "inception")
	}

	// dejavu — first user message matches a previous opener
	if len(s.UserMessages) > 0 && len(p.Sessions.FirstOpeners) > 0 {
		first := strings.TrimSpace(s.UserMessages[0])
		for _, prev := range p.Sessions.FirstOpeners {
			if strings.EqualFold(first, prev) {
				ids = append(ids, "dejavu")
				break
			}
		}
	}

	return ids
}
