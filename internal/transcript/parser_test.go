package transcript_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dirkbrnd/claude-feats/internal/transcript"
)

// writeTranscript writes a slice of raw JSON lines to a temp file.
func writeTranscript(t *testing.T, lines []string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "transcript-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range lines {
		f.WriteString(l + "\n")
	}
	f.Close()
	return f.Name()
}

func userEntry(text, ts string) string {
	msg := map[string]interface{}{
		"role": "user",
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
	}
	raw, _ := json.Marshal(msg)
	return `{"type":"user","timestamp":"` + ts + `","message":` + string(raw) + `}`
}

func bashEntry(cmd, ts string) string {
	input := map[string]interface{}{"command": cmd}
	rawInput, _ := json.Marshal(input)
	msg := map[string]interface{}{
		"role": "assistant",
		"content": []map[string]interface{}{
			{"type": "tool_use", "name": "Bash", "input": json.RawMessage(rawInput)},
		},
	}
	raw, _ := json.Marshal(msg)
	return `{"type":"assistant","timestamp":"` + ts + `","message":` + string(raw) + `}`
}

func editEntry(fp, ts string) string {
	input := map[string]interface{}{"file_path": fp}
	rawInput, _ := json.Marshal(input)
	msg := map[string]interface{}{
		"role": "assistant",
		"content": []map[string]interface{}{
			{"type": "tool_use", "name": "Edit", "input": json.RawMessage(rawInput)},
		},
	}
	raw, _ := json.Marshal(msg)
	return `{"type":"assistant","timestamp":"` + ts + `","message":` + string(raw) + `}`
}

func TestParse_UserMessages(t *testing.T) {
	path := writeTranscript(t, []string{
		userEntry("hello", "2026-05-04T10:00:00Z"),
		userEntry("world", "2026-05-04T10:01:00Z"),
	})
	s, err := transcript.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.UserMessages) != 2 {
		t.Errorf("user messages = %d, want 2", len(s.UserMessages))
	}
	if s.UserMessages[0] != "hello" {
		t.Errorf("first message = %q, want \"hello\"", s.UserMessages[0])
	}
}

func TestParse_Timestamps(t *testing.T) {
	path := writeTranscript(t, []string{
		userEntry("first", "2026-05-04T10:00:00Z"),
		userEntry("last", "2026-05-04T11:00:00Z"),
	})
	s, err := transcript.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.StartTime.IsZero() {
		t.Error("start time should not be zero")
	}
	if s.EndTime.IsZero() {
		t.Error("end time should not be zero")
	}
	if d := s.Duration(); d != time.Hour {
		t.Errorf("duration = %v, want 1h", d)
	}
}

func TestParse_BashCommands(t *testing.T) {
	path := writeTranscript(t, []string{
		bashEntry("git reset --hard HEAD~1", "2026-05-04T10:00:00Z"),
		bashEntry("git push --force origin main", "2026-05-04T10:01:00Z"),
	})
	s, err := transcript.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s.HasGitResetHard {
		t.Error("HasGitResetHard should be true")
	}
	if !s.HasGitPush {
		t.Error("HasGitPush should be true")
	}
	if !s.HasForceFlag {
		t.Error("HasForceFlag should be true (--force)")
	}
}

func TestParse_FilesEdited(t *testing.T) {
	path := writeTranscript(t, []string{
		editEntry("/repo/main.go", "2026-05-04T10:00:00Z"),
		editEntry("/repo/main.go", "2026-05-04T10:01:00Z"),
		editEntry("/repo/README.md", "2026-05-04T10:02:00Z"),
	})
	s, err := transcript.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.FilesEdited["/repo/main.go"] != 2 {
		t.Errorf("main.go edits = %d, want 2", s.FilesEdited["/repo/main.go"])
	}
	if _, ok := s.Extensions[".go"]; !ok {
		t.Error("extension .go should be tracked")
	}
	if _, ok := s.Extensions[".md"]; !ok {
		t.Error("extension .md should be tracked")
	}
}

func TestParse_WorktreeAdd(t *testing.T) {
	path := writeTranscript(t, []string{
		bashEntry("git worktree add ../worktrees/feat-x", "2026-05-04T10:00:00Z"),
	})
	s, err := transcript.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s.HasWorktreeAdd {
		t.Error("HasWorktreeAdd should be true")
	}
}

func TestParse_Interrupted(t *testing.T) {
	path := writeTranscript(t, []string{
		userEntry("[Request interrupted by user]", "2026-05-04T10:00:00Z"),
	})
	s, err := transcript.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Interrupted {
		t.Error("Interrupted should be true")
	}
}

func TestParse_RmRf(t *testing.T) {
	path := writeTranscript(t, []string{
		bashEntry("rm -rf /tmp/cache", "2026-05-04T10:00:00Z"),
	})
	s, err := transcript.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s.HasRmRf {
		t.Error("HasRmRf should be true")
	}
}

func TestParse_EmptyFile(t *testing.T) {
	path := writeTranscript(t, []string{})
	s, err := transcript.Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.UserMessages) != 0 {
		t.Error("empty transcript should have no user messages")
	}
}

func TestParse_MissingFile(t *testing.T) {
	_, err := transcript.Parse(filepath.Join(t.TempDir(), "nonexistent.jsonl"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestAvgUserMessageLen(t *testing.T) {
	path := writeTranscript(t, []string{
		userEntry("hi", "2026-05-04T10:00:00Z"),    // 2 chars
		userEntry("hello", "2026-05-04T10:01:00Z"), // 5 chars
	})
	s, _ := transcript.Parse(path)
	// avg = (2+5)/2 = 3.5
	if s.AvgUserMessageLen() != 3.5 {
		t.Errorf("avg = %f, want 3.5", s.AvgUserMessageLen())
	}
}

func TestTriageCount(t *testing.T) {
	msg := "1. fix\n2. skip\n3. fix\n4. fix\n5. skip"
	if n := transcript.TriageCount(msg); n != 5 {
		t.Errorf("triage count = %d, want 5", n)
	}
	if n := transcript.TriageCount("no triage here"); n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}
