package detector_test

import (
	"os"
	"testing"
	"time"

	"github.com/dirkbrnd/claude-feats/internal/detector"
	"github.com/dirkbrnd/claude-feats/internal/store"
	"github.com/dirkbrnd/claude-feats/internal/transcript"
)

func emptySession() *transcript.Session {
	return &transcript.Session{
		FilesEdited: make(map[string]int),
		Extensions:  make(map[string]struct{}),
		StartTime:   time.Now(),
		EndTime:     time.Now(),
	}
}

func TestMilestone_FirstBlood(t *testing.T) {
	p := emptyProgress()
	s := emptySession()
	ids := detector.Milestone{}.Check(s, p)
	assertContains(t, ids, "firstblood")
	if p.Sessions.Total != 1 {
		t.Errorf("total = %d, want 1", p.Sessions.Total)
	}
}

func TestMilestone_LuckySeven(t *testing.T) {
	for _, target := range []int{7, 77, 777} {
		p := emptyProgress()
		p.Sessions.Total = target - 1
		ids := detector.Milestone{}.Check(emptySession(), p)
		assertContains(t, ids, "luckyseven")
	}
}

func TestMilestone_Century(t *testing.T) {
	p := emptyProgress()
	p.Sessions.Total = 99
	ids := detector.Milestone{}.Check(emptySession(), p)
	assertContains(t, ids, "century")
}

func TestMilestone_EarlyInstall(t *testing.T) {
	// InstalledAt is set to now by freshProgress, which is within 30 days of release
	p := emptyProgress()
	p.InstalledAt = time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC)
	ids := detector.Milestone{}.Check(emptySession(), p)
	assertContains(t, ids, "earlyinstall")
}

func TestMilestone_StreakProgression(t *testing.T) {
	p := emptyProgress()
	// Simulate 7 consecutive days
	for i := 6; i >= 0; i-- {
		p.Streak.LastActiveDate = time.Now().AddDate(0, 0, -(i + 1)).Format("2006-01-02")
		p.Streak.Current = 6 - i
		detector.Milestone{}.Check(emptySession(), p)
	}
	assertContains(t, detector.Milestone{}.Check(emptySession(), p), "streak7")
}

func TestHiddenMilestone_Fibonacci(t *testing.T) {
	p := emptyProgress()
	p.Sessions.Total = 8 // already incremented by Milestone
	ids := detector.HiddenMilestone{}.Check(emptySession(), p)
	assertContains(t, ids, "fibonacci")
}

func TestHiddenMilestone_Palindrome(t *testing.T) {
	p := emptyProgress()
	p.Sessions.Total = 11
	ids := detector.HiddenMilestone{}.Check(emptySession(), p)
	assertContains(t, ids, "palindrome")
}

func TestManaDetector_Incantation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p := emptyProgress()
	p.UpdateMana(600_000, 400_001, 0, 0, "2026-05") // 1M+1 total
	ids := detector.ManaDetector{CurrentMonth: "2026-05"}.Check(emptySession(), p)
	assertContains(t, ids, "incantation")
}

func TestManaDetector_ApprenticeMonthly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p := emptyProgress()
	p.UpdateMana(60_000, 40_001, 0, 0, "2026-05") // 100K+1 for month
	ids := detector.ManaDetector{CurrentMonth: "2026-05"}.Check(emptySession(), p)
	assertContains(t, ids, "apprenticemage")
}

func TestManaDetector_ManaBurn(t *testing.T) {
	p := emptyProgress()
	ids := detector.ManaDetector{SessionTokens: 150_000, CurrentMonth: "2026-05"}.Check(emptySession(), p)
	assertContains(t, ids, "manaburn")
}

func TestManaDetector_PrecisionCast(t *testing.T) {
	p := emptyProgress()
	s := emptySession()
	s.HasGitPush = true
	ids := detector.ManaDetector{SessionTokens: 1500, SessionHasCommit: true, CurrentMonth: "2026-05"}.Check(s, p)
	assertContains(t, ids, "precisioncast")
}

func TestManaDetector_FrugalMage(t *testing.T) {
	p := emptyProgress()
	// Previous month: 10K avg
	p.Mana.Monthly["2026-04"] = store.ManaMonth{InputTokens: 10_000, OutputTokens: 0, Sessions: 1}
	// Current month: 5K avg
	p.Mana.Monthly["2026-05"] = store.ManaMonth{InputTokens: 5_000, OutputTokens: 0, Sessions: 1}
	ids := detector.ManaDetector{CurrentMonth: "2026-05"}.Check(emptySession(), p)
	assertContains(t, ids, "frugalmage")
}

func TestManaDetector_FrugalMage_NotTriggered(t *testing.T) {
	p := emptyProgress()
	// Previous month: 5K avg, current: 8K — worse, not triggered
	p.Mana.Monthly["2026-04"] = store.ManaMonth{InputTokens: 5_000, Sessions: 1}
	p.Mana.Monthly["2026-05"] = store.ManaMonth{InputTokens: 8_000, Sessions: 1}
	ids := detector.ManaDetector{CurrentMonth: "2026-05"}.Check(emptySession(), p)
	assertNotContains(t, ids, "frugalmage")
}

func init() {
	_ = os.Setenv // ensure os imported
}
