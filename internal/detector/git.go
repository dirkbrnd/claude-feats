package detector

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dirkbrand/claude-feats/internal/store"
	"github.com/dirkbrand/claude-feats/internal/transcript"
)

// Git detects feats based on git operations found in the session.
type Git struct {
	// WorkDir is the directory in which the session ran. If empty, the CWD
	// at worker start is used for archaeologist detection.
	WorkDir string
}

func (g Git) Check(s *transcript.Session, p *store.Progress) []string {
	var ids []string

	// timetraveler
	if s.HasGitRevert {
		ids = append(ids, "timetraveler")
	}

	// worktreeclone
	if s.HasWorktreeAdd {
		ids = append(ids, "worktreeclone")
	}

	// sovereign — 5+ extensions
	if len(s.Extensions) >= 5 {
		ids = append(ids, "sovereign")
	}

	// midnightship — git push between midnight and 3am
	if s.HasGitPush && !s.EndTime.IsZero() {
		local := s.EndTime.Local()
		h := local.Hour()
		if h >= 0 && h < 3 {
			ids = append(ids, "midnightship")
		}
	}

	// neveronsunday — 30+ sessions with 0 Sundays (index 0)
	if p.Sessions.Total >= 30 && p.Sessions.ByDayOfWeek[0] == 0 {
		if _, had := p.Feats["neveronsunday"]; !had {
			ids = append(ids, "neveronsunday")
		}
	}

	// archaeologist — repo has files older than 5 years
	if g.archaeologistCheck(s) {
		ids = append(ids, "archaeologist")
	}

	return ids
}

func (g Git) archaeologistCheck(s *transcript.Session) bool {
	dirs := collectDirs(s)
	if len(dirs) == 0 {
		if g.WorkDir != "" {
			dirs = append(dirs, g.WorkDir)
		} else {
			if cwd, err := os.Getwd(); err == nil {
				dirs = append(dirs, cwd)
			}
		}
	}

	cutoff := time.Now().AddDate(-5, 0, 0)
	for _, dir := range dirs {
		root := gitRoot(dir)
		if root == "" {
			continue
		}
		found := false
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if found || err != nil {
				return nil
			}
			if d.IsDir() && (d.Name() == ".git" || d.Name() == "vendor" || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			if info.ModTime().Before(cutoff) {
				found = true
			}
			return nil
		})
		if found {
			return true
		}
	}
	return false
}

func collectDirs(s *transcript.Session) []string {
	seen := make(map[string]struct{})
	var dirs []string
	for fp := range s.FilesEdited {
		dir := filepath.Dir(fp)
		if _, ok := seen[dir]; !ok {
			seen[dir] = struct{}{}
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

func gitRoot(dir string) string {
	d := dir
	for {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

// CacheDetector checks cache-efficiency feats from mana data.
// It receives the per-session token counts directly.
type CacheDetector struct {
	InputTokens              int
	CacheCreationInputTokens int
	CacheReadInputTokens     int
}

func (c CacheDetector) Check(_ *transcript.Session, _ *store.Progress) []string {
	total := c.InputTokens + c.CacheCreationInputTokens + c.CacheReadInputTokens
	if total == 0 {
		return nil
	}
	hitRate := float64(c.CacheReadInputTokens) / float64(total)

	var ids []string
	if hitRate >= 0.8 {
		ids = append(ids, "cachemaster")
	}
	if hitRate >= 0.9 {
		ids = append(ids, "cachehoarder")
	}
	return ids
}

// normalizeForMatch lowercases and strips excess whitespace.
func normalizeForMatch(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
