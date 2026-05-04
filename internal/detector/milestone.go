package detector

import (
	"strings"
	"time"

	"github.com/dirkbrnd/claude-feats/internal/catalog"
	"github.com/dirkbrnd/claude-feats/internal/store"
	"github.com/dirkbrnd/claude-feats/internal/transcript"
)

// Milestone detects count/streak-based feats and mutates progress counters.
// Must be run serially (not in the parallel goroutine pool).
type Milestone struct{}

func (Milestone) Check(s *transcript.Session, p *store.Progress) []string {
	var ids []string

	// Increment session counter and day-of-week tally
	p.Sessions.Total++
	if !s.StartTime.IsZero() {
		wd := int(s.StartTime.Local().Weekday()) // 0=Sunday
		p.Sessions.ByDayOfWeek[wd]++
	}

	// Track first opener (for dejavu hidden feat)
	if len(s.UserMessages) > 0 {
		first := strings.TrimSpace(s.UserMessages[0])
		if first != "" {
			p.Sessions.FirstOpeners = append(p.Sessions.FirstOpeners, first)
			if len(p.Sessions.FirstOpeners) > 100 {
				p.Sessions.FirstOpeners = p.Sessions.FirstOpeners[len(p.Sessions.FirstOpeners)-100:]
			}
		}
	}

	n := p.Sessions.Total

	// firstblood
	if n == 1 {
		ids = append(ids, "firstblood")
	}

	// luckyseven — 7th, 77th, or 777th
	if n == 7 || n == 77 || n == 777 {
		ids = append(ids, "luckyseven")
	}

	// century
	if n == 100 {
		ids = append(ids, "century")
	}

	// Update streak
	p.UpdateStreak()
	cur := p.Streak.Current

	if cur >= 7 {
		ids = append(ids, "streak7")
	}
	if cur >= 30 {
		ids = append(ids, "streak30")
	}
	if cur >= 365 {
		ids = append(ids, "streak365")
	}

	// anniversary
	if !p.InstalledAt.IsZero() {
		since := time.Since(p.InstalledAt)
		if since >= 365*24*time.Hour {
			if _, had := p.Feats["anniversary"]; !had {
				ids = append(ids, "anniversary")
			}
		}
	}

	// rarecollector — 10+ rare+ feats unlocked
	rareCount := 0
	for id := range p.Feats {
		f := catalog.ByID(id)
		if f != nil && f.Rarity.Order() >= catalog.Rare.Order() {
			rareCount++
		}
	}
	if rareCount >= 10 {
		if _, had := p.Feats["rarecollector"]; !had {
			ids = append(ids, "rarecollector")
		}
	}

	// earlyinstall — within 30 days of first public release (2026-05-04)
	firstRelease := time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	if !p.InstalledAt.IsZero() && p.InstalledAt.Before(firstRelease.Add(30*24*time.Hour)) {
		if _, had := p.Feats["earlyinstall"]; !had {
			ids = append(ids, "earlyinstall")
		}
	}

	return ids
}

// HiddenMilestone detects hidden milestone-based feats.
type HiddenMilestone struct{}

func (HiddenMilestone) Check(s *transcript.Session, p *store.Progress) []string {
	var ids []string
	n := p.Sessions.Total // already incremented by Milestone

	// fibonacci
	if catalog.IsFibonacci(n) {
		ids = append(ids, "fibonacci")
	}

	// palindrome
	if catalog.IsPalindrome(n) {
		ids = append(ids, "palindrome")
	}

	return ids
}

// ManaDetector checks token-based mana feats against the current progress state.
// Should be run serially after UpdateMana has been called for this session.
type ManaDetector struct {
	// SessionTokens is the total tokens for the current session (input+output).
	// Used for manaburn and precisioncast.
	SessionTokens int
	// SessionHasCommit indicates the session transcript contained a git push/commit.
	SessionHasCommit bool
	// CurrentMonth is "YYYY-MM" for the session being processed.
	CurrentMonth string
}

func (d ManaDetector) Check(s *transcript.Session, p *store.Progress) []string {
	var ids []string
	lifetime := p.Mana.LifetimeTotal()

	// ── Option C — lifetime milestones ──
	if lifetime >= 1_000_000 {
		if _, had := p.Feats["incantation"]; !had {
			ids = append(ids, "incantation")
		}
	}
	if lifetime >= 10_000_000 {
		if _, had := p.Feats["grimoire"]; !had {
			ids = append(ids, "grimoire")
		}
	}
	// hidden
	if lifetime >= 100_000_000 {
		if _, had := p.Feats["codexinfinitus"]; !had {
			ids = append(ids, "codexinfinitus")
		}
	}

	// ── Option A — monthly milestones ──
	month := d.CurrentMonth
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	if m, ok := p.Mana.Monthly[month]; ok {
		mt := m.Total()
		if mt >= 100_000 {
			if _, had := p.Feats["apprenticemage"]; !had {
				ids = append(ids, "apprenticemage")
			}
		}
		if mt >= 1_000_000 {
			if _, had := p.Feats["archmage"]; !had {
				ids = append(ids, "archmage")
			}
		}
		if mt >= 10_000_000 {
			if _, had := p.Feats["thevoid"]; !had {
				ids = append(ids, "thevoid")
			}
		}
	}

	// ── Option B — per-session efficiency ──
	if d.SessionTokens > 100_000 {
		ids = append(ids, "manaburn")
	}
	if d.SessionTokens > 0 && d.SessionTokens < 2_000 && d.SessionHasCommit {
		ids = append(ids, "precisioncast")
	}

	// frugalmage — current month avg < previous month avg
	if month != "" {
		prev := store.PrevMonthKey(month)
		if cur, ok1 := p.Mana.Monthly[month]; ok1 {
			if prv, ok2 := p.Mana.Monthly[prev]; ok2 && prv.AvgPerSession() > 0 {
				if cur.AvgPerSession() < prv.AvgPerSession() {
					ids = append(ids, "frugalmage")
				}
			}
		}
	}

	return ids
}
