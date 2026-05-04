package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/dirkbrnd/claude-feats/internal/store"
)

// hookPayload is the JSON that Claude Code's Stop hook sends on stdin.
type hookPayload struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Usage          *usage `json:"usage"`
}

type usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// pendingJob is what check.go writes for the worker to consume.
type pendingJob struct {
	SessionID      string    `json:"session_id"`
	TranscriptPath string    `json:"transcript_path"`
	RecordedAt     time.Time `json:"recorded_at"`
}

var checkCmd = &cobra.Command{
	Use:          "check",
	Short:        "Stop hook entrypoint — records mana, queues session analysis",
	SilenceUsage: true,
	RunE:         runCheck,
}

func runCheck(_ *cobra.Command, _ []string) error {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil // never block the hook
	}

	var payload hookPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}

	// Record mana inline (fast, no transcript parse needed)
	if payload.Usage != nil {
		u := payload.Usage
		month := time.Now().Format("2006-01")
		_ = store.LoadLocked(func(p *store.Progress) error {
			p.UpdateMana(
				u.InputTokens,
				u.OutputTokens,
				u.CacheCreationInputTokens,
				u.CacheReadInputTokens,
				month,
			)
			return nil
		})
	}

	// Write pending job for the worker to pick up
	if payload.TranscriptPath != "" && payload.SessionID != "" {
		pendingDir := store.PendingDir()
		if err := os.MkdirAll(pendingDir, 0o755); err == nil {
			job := pendingJob{
				SessionID:      payload.SessionID,
				TranscriptPath: payload.TranscriptPath,
				RecordedAt:     time.Now().UTC(),
			}
			data, _ := json.Marshal(job)
			jobPath := filepath.Join(pendingDir, payload.SessionID+".json")
			if err := os.WriteFile(jobPath, data, 0o644); err == nil {
				spawnWorker(jobPath)
			}
		}
	}

	return nil // always exit 0
}

// spawnWorker launches a detached worker process and returns immediately.
func spawnWorker(jobPath string) {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	attr := &os.ProcAttr{
		Files: []*os.File{nil, nil, nil},
		Sys:   &syscall.SysProcAttr{Setsid: true},
	}
	_, _ = os.StartProcess(exe, []string{exe, "worker", "--job", jobPath}, attr)
}
