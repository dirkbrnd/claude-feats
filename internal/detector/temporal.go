package detector

import (
	"time"

	"github.com/dirkbrnd/claude-feats/internal/catalog"
	"github.com/dirkbrnd/claude-feats/internal/store"
	"github.com/dirkbrnd/claude-feats/internal/transcript"
)

// Temporal detects feats based on wall-clock time of the session.
type Temporal struct{}

func (Temporal) Check(s *transcript.Session, _ *store.Progress) []string {
	if s.StartTime.IsZero() {
		return nil
	}
	local := s.StartTime.Local()
	hour := local.Hour()
	wd := local.Weekday()

	var ids []string

	// earlybird — 5:00–7:00am
	if hour >= 5 && hour < 7 {
		ids = append(ids, "earlybird")
	}

	// nightowl — midnight–5:00am
	if hour >= 0 && hour < 5 {
		ids = append(ids, "nightowl")
	}

	// weekendwarrior — Saturday or Sunday
	if wd == time.Saturday || wd == time.Sunday {
		ids = append(ids, "weekendwarrior")
	}

	// holidayhacker — US public holiday
	if catalog.IsUSHoliday(local) {
		ids = append(ids, "holidayhacker")
	}

	return ids
}

// HiddenTemporal detects hidden time-based feats.
type HiddenTemporal struct{}

func (HiddenTemporal) Check(s *transcript.Session, _ *store.Progress) []string {
	var ids []string

	if !s.EndTime.IsZero() {
		local := s.EndTime.Local()
		h, m, _ := local.Clock()
		// conspiracy — exactly 3:33am
		if h == 3 && m == 33 {
			ids = append(ids, "conspiracy")
		}
	}

	if !s.StartTime.IsZero() {
		local := s.StartTime.Local()
		mo, d := local.Month(), local.Day()
		h, _, _ := local.Clock()
		// silentnight — Dec 24 11pm–Dec 25 1am
		if (mo == time.December && d == 24 && h >= 23) ||
			(mo == time.December && d == 25 && h < 1) {
			ids = append(ids, "silentnight")
		}
	}

	return ids
}
