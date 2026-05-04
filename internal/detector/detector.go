package detector

import (
	"sync"

	"github.com/dirkbrand/claude-feats/internal/store"
	"github.com/dirkbrand/claude-feats/internal/transcript"
)

// Detector checks a session against a progress snapshot and returns feat IDs
// to unlock. A Detector may mutate progress (e.g., increment session counters).
type Detector interface {
	// Check returns feat IDs that should be unlocked for this session.
	// Detectors that need progress mutation must be safe to call concurrently
	// (behavioral/temporal ones only read; milestone ones should be run serially).
	Check(session *transcript.Session, progress *store.Progress) []string
}

// RunAll runs all detectors in parallel and returns the unique set of feat IDs
// that should be unlocked. Milestone detectors are run serially after to allow
// safe progress mutation.
func RunAll(
	session *transcript.Session,
	progress *store.Progress,
	parallel []Detector,
	serial []Detector,
) []string {
	type result struct {
		ids []string
	}
	results := make([]result, len(parallel))

	var wg sync.WaitGroup
	for i, d := range parallel {
		wg.Add(1)
		go func(idx int, det Detector) {
			defer wg.Done()
			results[idx] = result{ids: det.Check(session, progress)}
		}(i, d)
	}
	wg.Wait()

	seen := make(map[string]struct{})
	var all []string
	for _, r := range results {
		for _, id := range r.ids {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				all = append(all, id)
			}
		}
	}

	// Serial detectors (milestone/streak) run after to safely mutate progress
	for _, d := range serial {
		for _, id := range d.Check(session, progress) {
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				all = append(all, id)
			}
		}
	}

	return all
}
