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

// ManaDetector checks token-based mana feats.
type ManaDetector struct{}

func (ManaDetector) Check(_ *transcript.Session, p *store.Progress) []string {
	var ids []string
	total := p.Mana.Lifetime.Total()

	if total >= 10_000_000 {
		if _, had := p.Feats["spellbook10m"]; !had {
			ids = append(ids, "spellbook10m")
		}
	}
	if total >= 100_000_000 {
		if _, had := p.Feats["spellbook100m"]; !had {
			ids = append(ids, "spellbook100m")
		}
	}
	if total >= 1_000_000_000 {
		if _, had := p.Feats["spellbook1b"]; !had {
			ids = append(ids, "spellbook1b")
		}
	}

	// Monthly mana feats
	now := time.Now()
	monthKey := now.Format("2006-01")
	if m, ok := p.Mana.Monthly[monthKey]; ok {
		monthTotal := m.InputTokens + m.OutputTokens
		if monthTotal >= 1_000_000 {
			ids = append(ids, "manamonth1m")
		}
		if monthTotal >= 10_000_000 {
			ids = append(ids, "manamonth10m")
		}
	}

	return ids
}
