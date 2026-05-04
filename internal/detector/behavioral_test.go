package detector_test

import (
	"testing"
	"time"

	"github.com/dirkbrnd/claude-feats/internal/detector"
	"github.com/dirkbrnd/claude-feats/internal/store"
	"github.com/dirkbrnd/claude-feats/internal/transcript"
)

func emptyProgress() *store.Progress {
	return &store.Progress{
		Feats:   make(map[string]store.FeatProgress),
		Mana:    store.Mana{Monthly: make(map[string]store.ManaMonth)},
		Sessions: store.Sessions{},
	}
}

func sessionWithMessages(msgs ...string) *transcript.Session {
	s := &transcript.Session{
		FilesEdited: make(map[string]int),
		Extensions:  make(map[string]struct{}),
		StartTime:   time.Now().Add(-10 * time.Minute),
		EndTime:     time.Now(),
	}
	s.UserMessages = msgs
	return s
}

// ── Behavioral ───────────────────────────────────────────────────────────────

func TestBehavioral_VimReflex(t *testing.T) {
	s := sessionWithMessages(":wq")
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertContains(t, ids, "vimreflex")
}

func TestBehavioral_VimReflex_NotTriggeredByPartial(t *testing.T) {
	s := sessionWithMessages("run :wq please")
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertNotContains(t, ids, "vimreflex")
}

func TestBehavioral_OneWord(t *testing.T) {
	// avg length < 10
	s := sessionWithMessages("go", "fix", "ok")
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertContains(t, ids, "oneword")
}

func TestBehavioral_OneWord_NotTriggered(t *testing.T) {
	s := sessionWithMessages("please fix the authentication bug in the login service")
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertNotContains(t, ids, "oneword")
}

func TestBehavioral_Speedrun(t *testing.T) {
	s := sessionWithMessages("go")
	s.StartTime = time.Now().Add(-3 * time.Minute)
	s.EndTime = time.Now()
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertContains(t, ids, "speedrun")
}

func TestBehavioral_Speedrun_NotTriggered(t *testing.T) {
	s := sessionWithMessages("go")
	s.StartTime = time.Now().Add(-10 * time.Minute)
	s.EndTime = time.Now()
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertNotContains(t, ids, "speedrun")
}

func TestBehavioral_Triage(t *testing.T) {
	msg := "1. fix\n2. fix\n3. skip\n4. fix\n5. fix\n6. skip"
	s := sessionWithMessages(msg)
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertContains(t, ids, "triage")
}

func TestBehavioral_Triage_NotEnough(t *testing.T) {
	msg := "1. fix\n2. skip\n3. fix"
	s := sessionWithMessages(msg)
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertNotContains(t, ids, "triage")
}

func TestBehavioral_Interrupt(t *testing.T) {
	s := sessionWithMessages("do something")
	s.Interrupted = true
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertContains(t, ids, "interrupt")
}

func TestBehavioral_OneShot(t *testing.T) {
	s := sessionWithMessages("fix the bug")
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertContains(t, ids, "oneshot")
}

func TestBehavioral_OneShot_NotTriggered(t *testing.T) {
	s := sessionWithMessages("fix the bug", "looks good")
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertNotContains(t, ids, "oneshot")
}

func TestBehavioral_LongHaul(t *testing.T) {
	s := sessionWithMessages("working")
	s.StartTime = time.Now().Add(-5 * time.Hour)
	s.EndTime = time.Now()
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertContains(t, ids, "longhaul")
}

func TestBehavioral_Perfectionist(t *testing.T) {
	s := sessionWithMessages("refactor everything")
	s.FilesEdited = map[string]int{"main.go": 12}
	ids := detector.Behavioral{}.Check(s, emptyProgress())
	assertContains(t, ids, "perfectionist")
}

// ── HiddenBehavioral ─────────────────────────────────────────────────────────

func TestHiddenBehavioral_GhostMode(t *testing.T) {
	s := &transcript.Session{
		FilesEdited: make(map[string]int),
		Extensions:  make(map[string]struct{}),
	}
	ids := detector.HiddenBehavioral{}.Check(s, emptyProgress())
	assertContains(t, ids, "ghostmode")
}

func TestHiddenBehavioral_Oops(t *testing.T) {
	s := sessionWithMessages("fix it")
	s.HasGitResetHard = true
	ids := detector.HiddenBehavioral{}.Check(s, emptyProgress())
	assertContains(t, ids, "oops")
}

func TestHiddenBehavioral_Yolo(t *testing.T) {
	s := sessionWithMessages("deploy")
	s.HasForceFlag = true
	ids := detector.HiddenBehavioral{}.Check(s, emptyProgress())
	assertContains(t, ids, "yolo")
}

func TestHiddenBehavioral_DejaVu(t *testing.T) {
	p := emptyProgress()
	p.Sessions.FirstOpeners = []string{"fix the login bug"}
	s := sessionWithMessages("fix the login bug")
	ids := detector.HiddenBehavioral{}.Check(s, p)
	assertContains(t, ids, "dejavu")
}

// ── helpers ──────────────────────────────────────────────────────────────────

func assertContains(t *testing.T, ids []string, want string) {
	t.Helper()
	for _, id := range ids {
		if id == want {
			return
		}
	}
	t.Errorf("expected feat %q in %v", want, ids)
}

func assertNotContains(t *testing.T, ids []string, want string) {
	t.Helper()
	for _, id := range ids {
		if id == want {
			t.Errorf("feat %q should not be in %v", want, ids)
			return
		}
	}
}
