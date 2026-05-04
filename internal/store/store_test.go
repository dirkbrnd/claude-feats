package store_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dirkbrnd/claude-feats/internal/store"
)

func withTempDir(t *testing.T) func() {
	t.Helper()
	orig, _ := os.UserHomeDir()
	dir := t.TempDir()
	// Override home by pointing the store dir directly
	_ = orig
	t.Setenv("HOME", dir)
	return func() {}
}

func TestFreshProgress(t *testing.T) {
	withTempDir(t)()
	p, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if p.Version != store.Version {
		t.Errorf("version = %q, want %q", p.Version, store.Version)
	}
	if p.Feats == nil {
		t.Error("feats map is nil")
	}
	if p.Mana.Monthly == nil {
		t.Error("mana.monthly map is nil")
	}
}

func TestSaveLoad(t *testing.T) {
	withTempDir(t)()
	p, _ := store.Load()
	p.Sessions.Total = 42
	p.UnlockFeat("vimreflex", nil)

	if err := store.Save(p); err != nil {
		t.Fatal(err)
	}

	p2, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if p2.Sessions.Total != 42 {
		t.Errorf("sessions.total = %d, want 42", p2.Sessions.Total)
	}
	fp, ok := p2.Feats["vimreflex"]
	if !ok {
		t.Fatal("vimreflex feat not saved")
	}
	if fp.Count != 1 {
		t.Errorf("count = %d, want 1", fp.Count)
	}
}

func TestUnlockFeat_FirstTime(t *testing.T) {
	withTempDir(t)()
	p, _ := store.Load()
	isNew := p.UnlockFeat("vimreflex", nil)
	if !isNew {
		t.Error("first unlock should return true")
	}
	if p.Feats["vimreflex"].Count != 1 {
		t.Error("count should be 1")
	}
}

func TestUnlockFeat_Repeat(t *testing.T) {
	withTempDir(t)()
	p, _ := store.Load()
	p.UnlockFeat("interrupt", nil)
	isNew := p.UnlockFeat("interrupt", nil)
	if isNew {
		t.Error("repeat unlock should return false")
	}
	if p.Feats["interrupt"].Count != 2 {
		t.Errorf("count = %d, want 2", p.Feats["interrupt"].Count)
	}
}

func TestUnlockFeat_PersonalRecord(t *testing.T) {
	withTempDir(t)()
	p, _ := store.Load()
	pr1 := 100
	p.UnlockFeat("speedrun", &pr1)
	pr2 := 50
	p.UnlockFeat("speedrun", &pr2) // worse — should not replace
	if *p.Feats["speedrun"].PersonalRecord != 100 {
		t.Errorf("PR = %d, want 100", *p.Feats["speedrun"].PersonalRecord)
	}
	pr3 := 200
	p.UnlockFeat("speedrun", &pr3) // better
	if *p.Feats["speedrun"].PersonalRecord != 200 {
		t.Errorf("PR = %d, want 200", *p.Feats["speedrun"].PersonalRecord)
	}
}

func TestIsProcessed(t *testing.T) {
	withTempDir(t)()
	p, _ := store.Load()
	if p.IsProcessed("abc") {
		t.Error("should not be processed yet")
	}
	p.MarkProcessed("abc")
	if !p.IsProcessed("abc") {
		t.Error("should be processed after marking")
	}
	if p.IsProcessed("xyz") {
		t.Error("xyz should not be processed")
	}
}

func TestUpdateMana(t *testing.T) {
	withTempDir(t)()
	p, _ := store.Load()
	p.UpdateMana(1000, 500, 200, 300, "2026-05")
	p.UpdateMana(2000, 1000, 0, 0, "2026-05")

	m := p.Mana.Monthly["2026-05"]
	if m.InputTokens != 3000 {
		t.Errorf("input = %d, want 3000", m.InputTokens)
	}
	if m.Sessions != 2 {
		t.Errorf("sessions = %d, want 2", m.Sessions)
	}
	if p.Mana.LifetimeTotal() != 4500 {
		t.Errorf("lifetime = %d, want 4500", p.Mana.LifetimeTotal())
	}
}

func TestLifetimeTotal_SumsMonthly(t *testing.T) {
	withTempDir(t)()
	p, _ := store.Load()
	p.UpdateMana(1000, 500, 0, 0, "2026-04")
	p.UpdateMana(2000, 1000, 0, 0, "2026-05")
	// 1000+500 + 2000+1000 = 4500
	if got := p.Mana.LifetimeTotal(); got != 4500 {
		t.Errorf("LifetimeTotal = %d, want 4500", got)
	}
}

func TestUpdateStreak_Consecutive(t *testing.T) {
	withTempDir(t)()
	p, _ := store.Load()
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	p.Streak.LastActiveDate = yesterday
	p.Streak.Current = 3
	p.UpdateStreak()
	if p.Streak.Current != 4 {
		t.Errorf("current = %d, want 4", p.Streak.Current)
	}
}

func TestUpdateStreak_Broken(t *testing.T) {
	withTempDir(t)()
	p, _ := store.Load()
	p.Streak.LastActiveDate = "2026-01-01" // long ago
	p.Streak.Current = 10
	p.Streak.Longest = 10
	p.UpdateStreak()
	if p.Streak.Current != 1 {
		t.Errorf("current = %d, want 1 (streak broken)", p.Streak.Current)
	}
	if p.Streak.Longest != 10 {
		t.Errorf("longest should remain 10, got %d", p.Streak.Longest)
	}
}

func TestUpdateStreak_Idempotent(t *testing.T) {
	withTempDir(t)()
	p, _ := store.Load()
	p.UpdateStreak()
	before := p.Streak.Current
	p.UpdateStreak()
	if p.Streak.Current != before {
		t.Error("calling UpdateStreak twice on same day should be idempotent")
	}
}

func TestAtomicSave(t *testing.T) {
	withTempDir(t)()
	// Verify temp file doesn't linger after save
	p, _ := store.Load()
	if err := store.Save(p); err != nil {
		t.Fatal(err)
	}
	tmp := store.ProgressPath() + ".tmp"
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Error("temp file should not exist after successful save")
	}
}

func TestLoadLocked(t *testing.T) {
	withTempDir(t)()
	called := false
	err := store.LoadLocked(func(p *store.Progress) error {
		called = true
		p.Sessions.Total = 99
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("fn was not called")
	}
	p, _ := store.Load()
	if p.Sessions.Total != 99 {
		t.Errorf("total = %d, want 99", p.Sessions.Total)
	}
}

func TestPrevMonthKey(t *testing.T) {
	cases := []struct{ in, want string }{
		{"2026-05", "2026-04"},
		{"2026-01", "2025-12"},
		{"2025-12", "2025-11"},
	}
	for _, tc := range cases {
		got := store.PrevMonthKey(tc.in)
		if got != tc.want {
			t.Errorf("PrevMonthKey(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestManaMonth_AvgPerSession(t *testing.T) {
	m := store.ManaMonth{InputTokens: 900, OutputTokens: 100, Sessions: 5}
	if got := m.AvgPerSession(); got != 200 {
		t.Errorf("AvgPerSession = %d, want 200", got)
	}
	zero := store.ManaMonth{}
	if got := zero.AvgPerSession(); got != 0 {
		t.Errorf("zero sessions avg = %d, want 0", got)
	}
}

// Make ProgressPath accessible for test cleanup check.
func init() {
	_ = filepath.Join // ensure filepath imported
}
