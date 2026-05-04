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

	// Read the pending job file
	data, err := os.ReadFile(workerJobPath)
	if err != nil {
		return nil
	}
	var job pendingJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil
	}

	// Parse the transcript
	session, err := transcript.Parse(job.TranscriptPath)
	if err != nil {
		return nil
	}

	// Read cache usage from job (already recorded by check.go, but we need it
	// to run the CacheDetector which only reads — no double-counting)
	var cacheDetector detector.CacheDetector
	// We can derive cache metrics from the transcript if needed; for now skip
	// since check.go already updated mana. The CacheDetector needs raw numbers.
	// We'll skip cache feats here since we don't re-read usage from the job.
	_ = cacheDetector

	var unlockedIDs []string
	var newFeats []string

	err = store.LoadLocked(func(p *store.Progress) error {
		p.LastCheck = time.Now().UTC()

		// Parallel detectors (read-only on progress)
		parallel := []detector.Detector{
			detector.Behavioral{},
			detector.HiddenBehavioral{},
			detector.Temporal{},
			detector.HiddenTemporal{},
			detector.Git{},
		}

		// Serial detectors (mutate progress counters)
		serial := []detector.Detector{
			detector.Milestone{},
			detector.HiddenMilestone{},
			detector.ManaDetector{},
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
	if err != nil {
		return nil
	}

	// Print unlock notifications to stdout (CC may surface these)
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

	// Clean up the pending job file
	_ = os.Remove(workerJobPath)

	return nil
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
