package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/gofrs/flock"
)

const Version = "1.0.0"

// FeatProgress tracks per-feat unlock data.
type FeatProgress struct {
	FirstUnlockedAt time.Time `json:"firstUnlockedAt"`
	LastUnlockedAt  time.Time `json:"lastUnlockedAt"`
	Count           int       `json:"count"`
	PersonalRecord  *int      `json:"personalRecord,omitempty"`
}

// Streak tracks the active coding streak.
type Streak struct {
	Current        int    `json:"current"`
	Longest        int    `json:"longest"`
	LastActiveDate string `json:"lastActiveDate"` // YYYY-MM-DD
}

// Sessions tracks aggregate session metadata.
type Sessions struct {
	Total       int      `json:"total"`
	ByDayOfWeek [7]int   `json:"byDayOfWeek"`  // 0=Sunday ... 6=Saturday
	FirstOpeners []string `json:"firstOpeners"` // last 100 session openers (for dejavu)
}

// ManaMonth tracks token usage for a calendar month.
type ManaMonth struct {
	InputTokens              int `json:"inputTokens"`
	OutputTokens             int `json:"outputTokens"`
	CacheCreationInputTokens int `json:"cacheCreationInputTokens"`
	CacheReadInputTokens     int `json:"cacheReadInputTokens"`
	Sessions                 int `json:"sessions"`
}

// Total returns input + output tokens for the month.
func (m ManaMonth) Total() int { return m.InputTokens + m.OutputTokens }

// AvgPerSession returns average tokens per session, or 0.
func (m ManaMonth) AvgPerSession() int {
	if m.Sessions == 0 {
		return 0
	}
	return m.Total() / m.Sessions
}

// Mana holds all token tracking data. Lifetime totals are computed on-the-fly
// from Monthly to avoid a separate running counter.
type Mana struct {
	Monthly map[string]ManaMonth `json:"monthly"` // key: "2026-05"
}

// LifetimeTotal returns the sum of input+output tokens across all months.
func (ma *Mana) LifetimeTotal() int {
	total := 0
	for _, m := range ma.Monthly {
		total += m.Total()
	}
	return total
}

// LifetimeInputOutput returns lifetime input and output totals separately.
func (ma *Mana) LifetimeInputOutput() (input, output int) {
	for _, m := range ma.Monthly {
		input += m.InputTokens
		output += m.OutputTokens
	}
	return
}

// LifetimeCacheTokens returns lifetime cache creation and read totals.
func (ma *Mana) LifetimeCacheTokens() (create, read int) {
	for _, m := range ma.Monthly {
		create += m.CacheCreationInputTokens
		read += m.CacheReadInputTokens
	}
	return
}

// SortedMonthKeys returns month keys sorted ascending.
func (ma *Mana) SortedMonthKeys() []string {
	keys := make([]string, 0, len(ma.Monthly))
	for k := range ma.Monthly {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// PrevMonthKey returns the month key immediately before the given one, or "".
func PrevMonthKey(key string) string {
	t, err := time.Parse("2006-01", key)
	if err != nil {
		return ""
	}
	return t.AddDate(0, -1, 0).Format("2006-01")
}

// Progress is the root progress.json schema.
type Progress struct {
	Version           string                  `json:"version"`
	InstalledAt       time.Time               `json:"installedAt"`
	LastCheck         time.Time               `json:"lastCheck"`
	SeededAt          time.Time               `json:"seededAt,omitempty"`
	ProcessedSessions []string                `json:"processedSessions,omitempty"`
	Streak            Streak                  `json:"streak"`
	Sessions          Sessions                `json:"sessions"`
	Mana              Mana                    `json:"mana"`
	Feats             map[string]FeatProgress `json:"feats"`
}

// IsProcessed returns true if sessionID has already been analyzed.
func (p *Progress) IsProcessed(sessionID string) bool {
	for _, id := range p.ProcessedSessions {
		if id == sessionID {
			return true
		}
	}
	return false
}

// MarkProcessed appends sessionID to ProcessedSessions (deduplication handled
// by the caller checking IsProcessed first).
func (p *Progress) MarkProcessed(sessionID string) {
	p.ProcessedSessions = append(p.ProcessedSessions, sessionID)
}

// Dir returns the ~/.claude-feats directory path.
func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude-feats")
}

// ProgressPath returns the full path to progress.json.
func ProgressPath() string {
	return filepath.Join(Dir(), "progress.json")
}

// PendingDir returns the path to the pending/ directory.
func PendingDir() string {
	return filepath.Join(Dir(), "pending")
}

// Load reads progress.json from disk. Returns a fresh Progress if the file
// doesn't exist yet.
func Load() (*Progress, error) {
	data, err := os.ReadFile(ProgressPath())
	if os.IsNotExist(err) {
		return freshProgress(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read progress: %w", err)
	}
	var p Progress
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse progress: %w", err)
	}
	initMaps(&p)
	return &p, nil
}

// Save writes progress atomically (temp file + rename) with a file lock.
func Save(p *Progress) error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	lockPath := filepath.Join(dir, ".progress.lock")
	fl := flock.New(lockPath)
	if err := fl.Lock(); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer fl.Unlock()
	return saveNoLock(p)
}

// LoadLocked reads, calls fn, then saves atomically — all under the file lock.
func LoadLocked(fn func(*Progress) error) error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	lockPath := filepath.Join(dir, ".progress.lock")
	fl := flock.New(lockPath)
	if err := fl.Lock(); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer fl.Unlock()

	p, err := loadNoLock()
	if err != nil {
		return err
	}
	if err := fn(p); err != nil {
		return err
	}
	return saveNoLock(p)
}

func loadNoLock() (*Progress, error) {
	data, err := os.ReadFile(ProgressPath())
	if os.IsNotExist(err) {
		return freshProgress(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read progress: %w", err)
	}
	var p Progress
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse progress: %w", err)
	}
	initMaps(&p)
	return &p, nil
}

func saveNoLock(p *Progress) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	tmp := ProgressPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	return os.Rename(tmp, ProgressPath())
}

func initMaps(p *Progress) {
	if p.Feats == nil {
		p.Feats = make(map[string]FeatProgress)
	}
	if p.Mana.Monthly == nil {
		p.Mana.Monthly = make(map[string]ManaMonth)
	}
}

func freshProgress() *Progress {
	return &Progress{
		Version:     Version,
		InstalledAt: time.Now().UTC(),
		Mana:        Mana{Monthly: make(map[string]ManaMonth)},
		Feats:       make(map[string]FeatProgress),
	}
}

// UnlockFeat records a feat unlock. Returns true if this is the first unlock.
func (p *Progress) UnlockFeat(id string, pr *int) bool {
	existing, had := p.Feats[id]
	now := time.Now().UTC()
	if !had {
		p.Feats[id] = FeatProgress{
			FirstUnlockedAt: now,
			LastUnlockedAt:  now,
			Count:           1,
			PersonalRecord:  pr,
		}
		return true
	}
	existing.LastUnlockedAt = now
	existing.Count++
	if pr != nil && (existing.PersonalRecord == nil || *pr > *existing.PersonalRecord) {
		existing.PersonalRecord = pr
	}
	p.Feats[id] = existing
	return false
}

// UpdateMana adds token counts to the given calendar month.
func (p *Progress) UpdateMana(input, output, cacheCreate, cacheRead int, month string) {
	m := p.Mana.Monthly[month]
	m.InputTokens += input
	m.OutputTokens += output
	m.CacheCreationInputTokens += cacheCreate
	m.CacheReadInputTokens += cacheRead
	m.Sessions++
	p.Mana.Monthly[month] = m
}

// UpdateStreak updates streak counters. Call once per session, after setting session date.
func (p *Progress) UpdateStreak() {
	today := time.Now().Format("2006-01-02")
	if p.Streak.LastActiveDate == today {
		return
	}
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if p.Streak.LastActiveDate == yesterday {
		p.Streak.Current++
	} else {
		p.Streak.Current = 1
	}
	if p.Streak.Current > p.Streak.Longest {
		p.Streak.Longest = p.Streak.Current
	}
	p.Streak.LastActiveDate = today
}

// UpdateStreakForDate updates streak counters for a specific date (used by seed).
func (p *Progress) UpdateStreakForDate(date string) {
	if p.Streak.LastActiveDate == date {
		return
	}
	// Parse both dates to check if they're consecutive
	last, err1 := time.Parse("2006-01-02", p.Streak.LastActiveDate)
	curr, err2 := time.Parse("2006-01-02", date)
	if err1 == nil && err2 == nil {
		diff := curr.Sub(last)
		if diff == 24*time.Hour {
			p.Streak.Current++
		} else if diff > 0 {
			p.Streak.Current = 1
		}
		// if diff <= 0 (out-of-order), don't update
		if curr.After(last) {
			if p.Streak.Current > p.Streak.Longest {
				p.Streak.Longest = p.Streak.Current
			}
			p.Streak.LastActiveDate = date
		}
	} else if p.Streak.LastActiveDate == "" {
		p.Streak.Current = 1
		p.Streak.Longest = 1
		p.Streak.LastActiveDate = date
	}
}
