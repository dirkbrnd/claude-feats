package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

const Version = "1.0.0"

// FeatProgress tracks per-feat unlock data.
type FeatProgress struct {
	FirstUnlockedAt time.Time  `json:"firstUnlockedAt"`
	LastUnlockedAt  time.Time  `json:"lastUnlockedAt"`
	Count           int        `json:"count"`
	PersonalRecord  *int       `json:"personalRecord,omitempty"` // session-seconds for speedrun/longhaul, triage count, etc.
}

// Streak tracks the active coding streak.
type Streak struct {
	Current        int    `json:"current"`
	Longest        int    `json:"longest"`
	LastActiveDate string `json:"lastActiveDate"` // YYYY-MM-DD
}

// Sessions tracks aggregate session metadata.
type Sessions struct {
	Total        int    `json:"total"`
	ByDayOfWeek  [7]int `json:"byDayOfWeek"`  // 0=Sunday ... 6=Saturday
	FirstOpeners []string `json:"firstOpeners"` // first message text of each session (last 100)
}

// ManaMonth tracks token usage for a calendar month.
type ManaMonth struct {
	InputTokens              int `json:"inputTokens"`
	OutputTokens             int `json:"outputTokens"`
	CacheCreationInputTokens int `json:"cacheCreationInputTokens"`
	CacheReadInputTokens     int `json:"cacheReadInputTokens"`
	Sessions                 int `json:"sessions"`
}

// ManaLifetime tracks all-time token totals.
type ManaLifetime struct {
	InputTokens              int `json:"inputTokens"`
	OutputTokens             int `json:"outputTokens"`
	CacheCreationInputTokens int `json:"cacheCreationInputTokens"`
	CacheReadInputTokens     int `json:"cacheReadInputTokens"`
}

// Mana holds all token tracking data.
type Mana struct {
	Lifetime ManaLifetime         `json:"lifetime"`
	Monthly  map[string]ManaMonth `json:"monthly"` // key: "2026-05"
}

// Progress is the root progress.json schema.
type Progress struct {
	Version     string                  `json:"version"`
	InstalledAt time.Time               `json:"installedAt"`
	LastCheck   time.Time               `json:"lastCheck"`
	Streak      Streak                  `json:"streak"`
	Sessions    Sessions                `json:"sessions"`
	Mana        Mana                    `json:"mana"`
	Feats       map[string]FeatProgress `json:"feats"`
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
	path := ProgressPath()
	data, err := os.ReadFile(path)
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
	if p.Feats == nil {
		p.Feats = make(map[string]FeatProgress)
	}
	if p.Mana.Monthly == nil {
		p.Mana.Monthly = make(map[string]ManaMonth)
	}
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

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal progress: %w", err)
	}

	tmp := ProgressPath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}
	if err := os.Rename(tmp, ProgressPath()); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
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
	if p.Feats == nil {
		p.Feats = make(map[string]FeatProgress)
	}
	if p.Mana.Monthly == nil {
		p.Mana.Monthly = make(map[string]ManaMonth)
	}
	return &p, nil
}

func saveNoLock(p *Progress) error {
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

func freshProgress() *Progress {
	return &Progress{
		Version:     Version,
		InstalledAt: time.Now().UTC(),
		LastCheck:   time.Time{},
		Streak:      Streak{},
		Sessions:    Sessions{},
		Mana: Mana{
			Monthly: make(map[string]ManaMonth),
		},
		Feats: make(map[string]FeatProgress),
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

// UpdateMana adds token counts from a session to monthly and lifetime totals.
func (p *Progress) UpdateMana(input, output, cacheCreate, cacheRead int, month string) {
	m := p.Mana.Monthly[month]
	m.InputTokens += input
	m.OutputTokens += output
	m.CacheCreationInputTokens += cacheCreate
	m.CacheReadInputTokens += cacheRead
	m.Sessions++
	p.Mana.Monthly[month] = m

	p.Mana.Lifetime.InputTokens += input
	p.Mana.Lifetime.OutputTokens += output
	p.Mana.Lifetime.CacheCreationInputTokens += cacheCreate
	p.Mana.Lifetime.CacheReadInputTokens += cacheRead
}

// UpdateStreak updates the streak counters based on today's date.
func (p *Progress) UpdateStreak() {
	today := time.Now().Format("2006-01-02")
	if p.Streak.LastActiveDate == today {
		return // already recorded today
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

// TotalMana returns total tokens cast (input + output).
func (m *ManaLifetime) Total() int {
	return m.InputTokens + m.OutputTokens
}
