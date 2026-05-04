package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/dirkbrnd/claude-feats/internal/catalog"
	"github.com/dirkbrnd/claude-feats/internal/detector"
	"github.com/dirkbrnd/claude-feats/internal/store"
	"github.com/dirkbrnd/claude-feats/internal/transcript"
)

var workerJobPath string

var workerCmd = &cobra.Command{
	Use:          "worker",
	Short:        "Detached worker: parse transcript, run detectors, update store",
	Hidden:       true,
	SilenceUsage: true,
	RunE:         runWorker,
}

func init() {
	workerCmd.Flags().StringVar(&workerJobPath, "job", "", "path to pending job JSON")
	rootCmd.AddCommand(workerCmd)
}

func runWorker(_ *cobra.Command, _ []string) error {
	if workerJobPath == "" {
		return nil
	}
	data, err := os.ReadFile(workerJobPath)
	if err != nil {
		return nil
	}
	var job pendingJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil
	}

	session, err := transcript.Parse(job.TranscriptPath)
	if err != nil {
		return nil
	}

	unlocked, newFeats := analyzeSession(session, job.SessionID, time.Now().Format("2006-01"), 0)

	printUnlocks(unlocked, newFeats)
	_ = os.Remove(workerJobPath)
	return nil
}

// analyzeSession runs all detectors for a parsed session and updates progress.json.
// month is "YYYY-MM", sessionTokens is the usage total (0 if unknown).
// Returns (allUnlockedIDs, firstTimeIDs).
func analyzeSession(session *transcript.Session, sessionID, month string, sessionTokens int) ([]string, []string) {
	var unlockedIDs []string
	var newFeats []string

	_ = store.LoadLocked(func(p *store.Progress) error {
		if sessionID != "" {
			if p.IsProcessed(sessionID) {
				return nil
			}
			p.MarkProcessed(sessionID)
		}

		p.LastCheck = time.Now().UTC()

		parallel := []detector.Detector{
			detector.Behavioral{},
			detector.HiddenBehavioral{},
			detector.Temporal{},
			detector.HiddenTemporal{},
			detector.Git{},
		}

		serial := []detector.Detector{
			detector.Milestone{},
			detector.HiddenMilestone{},
			detector.ManaDetector{
				SessionTokens:    sessionTokens,
				SessionHasCommit: session.HasGitPush,
				CurrentMonth:     month,
			},
		}

		ids := detector.RunAll(session, p, parallel, serial)

		for _, id := range ids {
			f := catalog.ByID(id)
			if f == nil {
				continue
			}
			isNew := p.UnlockFeat(id, nil)
			unlockedIDs = append(unlockedIDs, id)
			if isNew {
				newFeats = append(newFeats, id)
			}
		}
		return nil
	})

	return unlockedIDs, newFeats
}

// printUnlocks writes feat unlock notifications to stdout.
func printUnlocks(unlockedIDs, newFeats []string) {
	for _, id := range unlockedIDs {
		f := catalog.ByID(id)
		if f == nil {
			continue
		}
		isNew := contains(newFeats, id)
		if isNew {
			fmt.Printf("\n🏆 FEAT UNLOCKED: %s%s %s%s\n   %s\n",
				f.Rarity.Color(), f.Rarity.Badge(), f.Name, reset,
				f.Description,
			)
		} else if f.Countable {
			prog, _ := loadFeatProgress(id)
			fmt.Printf("  %s%s %s%s ×%d\n", f.Rarity.Color(), f.Rarity.Badge(), f.Name, reset, prog)
		}
	}
}

const reset = "\033[0m"

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func loadFeatProgress(id string) (int, bool) {
	p, err := store.Load()
	if err != nil {
		return 0, false
	}
	fp, ok := p.Feats[id]
	return fp.Count, ok
}
