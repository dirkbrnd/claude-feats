package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/dirkbrnd/claude-feats/internal/store"
	"github.com/dirkbrnd/claude-feats/internal/transcript"
)

var (
	seedDryRun      bool
	seedConcurrency int
)

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "One-time backfill of all historical Claude Code sessions",
	Long: `Scans ~/.claude/projects/ for all past session transcripts and runs the
full detector pipeline on each. Skips sessions already processed.

Safe to run multiple times — idempotent via processedSessions tracking.`,
	RunE: runSeed,
}

func init() {
	seedCmd.Flags().BoolVar(&seedDryRun, "dry-run", false, "Preview unlocks without writing progress")
	seedCmd.Flags().IntVar(&seedConcurrency, "concurrency", 8, "Number of parallel workers")
	rootCmd.AddCommand(seedCmd)
}

type seedEntry struct {
	path      string
	sessionID string
	mtime     time.Time
}

func runSeed(_ *cobra.Command, _ []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	// Discover all transcript files
	fmt.Printf("Scanning ~/.claude/projects/ ... ")
	entries, err := discoverTranscripts(filepath.Join(home, ".claude", "projects"))
	if err != nil {
		return fmt.Errorf("discover: %w", err)
	}
	fmt.Printf("found %d sessions\n", len(entries))

	if len(entries) == 0 {
		fmt.Println("No sessions found. Nothing to do.")
		return nil
	}

	// Load current progress to skip already-processed sessions
	current, err := store.Load()
	if err != nil {
		return fmt.Errorf("load progress: %w", err)
	}

	// Filter to unprocessed
	var toProcess []seedEntry
	for _, e := range entries {
		if !current.IsProcessed(e.sessionID) {
			toProcess = append(toProcess, e)
		}
	}
	skipped := len(entries) - len(toProcess)
	if skipped > 0 {
		fmt.Printf("Skipping %d already-processed sessions.\n", skipped)
	}
	if len(toProcess) == 0 {
		fmt.Println("All sessions already processed. Nothing to do.")
		return nil
	}

	fmt.Printf("Processing %d sessions (concurrency=%d)%s\n\n",
		len(toProcess), seedConcurrency,
		map[bool]string{true: "  [DRY RUN]", false: ""}[seedDryRun])

	type result struct {
		entry    seedEntry
		unlocked []string
		newFeats []string
		err      error
	}

	jobs := make(chan seedEntry, len(toProcess))
	results := make(chan result, len(toProcess))

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < seedConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range jobs {
				session, err := transcript.Parse(e.path)
				if err != nil {
					results <- result{entry: e, err: err}
					continue
				}
				month := e.mtime.Format("2006-01")
				if seedDryRun {
					// Don't write — just collect what would unlock
					results <- result{entry: e}
				} else {
					unlocked, newFeats := analyzeSession(session, e.sessionID, month, 0)
					results <- result{entry: e, unlocked: unlocked, newFeats: newFeats}
				}
			}
		}()
	}

	// Feed jobs
	go func() {
		for _, e := range toProcess {
			jobs <- e
		}
		close(jobs)
	}()

	// Close results when all workers done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results with progress display
	done := 0
	total := len(toProcess)
	start := time.Now()
	var allNewFeats []string

	for r := range results {
		done++
		printProgress(done, total, start)

		for _, id := range r.newFeats {
			fmt.Printf("\n  🏆 %s (session #%s — %s)\n",
				featName(id), r.entry.sessionID[:min(8, len(r.entry.sessionID))],
				r.entry.mtime.Format("2006-01-02"))
			allNewFeats = append(allNewFeats, id)
		}
		for _, id := range r.unlocked {
			if !contains(r.newFeats, id) {
				// countable re-unlock — print quietly
				_ = id
			}
		}
	}

	fmt.Printf("\n\nDone. %d sessions processed. %d feats unlocked.\n", total, len(allNewFeats))

	if !seedDryRun {
		// Mark seededAt
		_ = store.LoadLocked(func(p *store.Progress) error {
			p.SeededAt = time.Now().UTC()
			return nil
		})
		fmt.Println("Run `claude-feats stats` to see your collection.")
	}

	return nil
}

// discoverTranscripts walks ~/.claude/projects/*/conversations/*.jsonl
// and returns entries sorted by mtime ascending (oldest first).
func discoverTranscripts(projectsDir string) ([]seedEntry, error) {
	pattern := filepath.Join(projectsDir, "*", "conversations", "*.jsonl")
	// Also check flat layout: ~/.claude/projects/*/*.jsonl
	pattern2 := filepath.Join(projectsDir, "*", "*.jsonl")

	var entries []seedEntry
	seen := make(map[string]struct{})

	for _, pat := range []string{pattern, pattern2} {
		matches, err := filepath.Glob(pat)
		if err != nil {
			continue
		}
		for _, path := range matches {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}

			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			// Derive session ID from filename (strip .jsonl)
			base := filepath.Base(path)
			sessionID := strings.TrimSuffix(base, filepath.Ext(base))

			entries = append(entries, seedEntry{
				path:      path,
				sessionID: sessionID,
				mtime:     info.ModTime(),
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].mtime.Before(entries[j].mtime)
	})
	return entries, nil
}

func printProgress(done, total int, start time.Time) {
	width := 20
	filled := done * width / total
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	pct := done * 100 / total

	elapsed := time.Since(start)
	var eta string
	if done > 0 {
		remaining := time.Duration(float64(elapsed) / float64(done) * float64(total-done))
		eta = fmt.Sprintf("  ~%s remaining", remaining.Round(time.Second))
	}

	fmt.Printf("\rProcessing %s %d%%  (%d/%d)%s   ",
		bar, pct, done, total, eta)
}

func featName(id string) string {
	from := []string{
		"github.com/dirkbrnd/claude-feats/internal/catalog",
	}
	_ = from
	// inline lookup to avoid import cycle concern (catalog is already imported via worker)
	names := map[string]string{
		"vimreflex": ":wq", "oneword": "The Sovereign", "speedrun": "Speedrun",
		"triage": "Triage Master", "interrupt": "Interrupt Strike", "oneshot": "One and Done",
		"longhaul": "Marathon", "perfectionist": "Perfectionist",
		"earlybird": "Early Bird", "nightowl": "Night Owl",
		"weekendwarrior": "Weekend Warrior", "holidayhacker": "Holiday Hacker",
		"firstblood": "First Blood", "luckyseven": "Lucky Seven", "century": "The Century",
		"streak7": "On a Roll", "streak30": "Relentless", "streak365": "Transcendent",
		"anniversary": "Anniversary", "rarecollector": "Rare Collector",
		"timetraveler": "Time Traveler", "archaeologist": "Archaeologist",
		"midnightship": "Midnight Ship", "neveronsunday": "Never on Sunday",
		"worktreeclone": "Shadow Clone", "sovereign": "Platform Sovereign",
		"earlyinstall": "Early Adopter",
		"apprenticemage": "Apprentice Mage", "archmage": "Archmage", "thevoid": "The Void",
		"manaburn": "Mana Burn", "precisioncast": "Precision Cast", "frugalmage": "The Frugal Mage",
		"incantation": "Incantation", "grimoire": "Grimoire",
	}
	if n, ok := names[id]; ok {
		return n
	}
	return id
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
