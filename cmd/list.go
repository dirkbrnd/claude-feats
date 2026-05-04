package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/dirkbrand/claude-feats/internal/catalog"
	"github.com/dirkbrand/claude-feats/internal/store"
)

var listUnlocked bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Show visible feats (unlocked and locked)",
	RunE:  runList,
}

func init() {
	listCmd.Flags().BoolVar(&listUnlocked, "unlocked", false, "Show only unlocked feats")
}

func runList(_ *cobra.Command, _ []string) error {
	p, err := store.Load()
	if err != nil {
		return fmt.Errorf("load progress: %w", err)
	}

	feats := catalog.Visible()

	// Sort: unlocked first, then by rarity desc, then name
	sort.Slice(feats, func(i, j int) bool {
		_, iUnlocked := p.Feats[feats[i].ID]
		_, jUnlocked := p.Feats[feats[j].ID]
		if iUnlocked != jUnlocked {
			return iUnlocked
		}
		if feats[i].Rarity.Order() != feats[j].Rarity.Order() {
			return feats[i].Rarity.Order() > feats[j].Rarity.Order()
		}
		return feats[i].Name < feats[j].Name
	})

	useColor := supportsColor()

	total := len(feats)
	unlocked := 0
	for _, f := range feats {
		if _, ok := p.Feats[f.ID]; ok {
			unlocked++
		}
	}

	fmt.Printf("\n  claude-feats  —  %d / %d unlocked\n\n", unlocked, total)

	for _, f := range feats {
		fp, isUnlocked := p.Feats[f.ID]
		if listUnlocked && !isUnlocked {
			continue
		}

		color := ""
		colorReset := ""
		if useColor {
			color = f.Rarity.Color()
			colorReset = reset
		}

		if isUnlocked {
			countStr := ""
			if f.Countable && fp.Count > 1 {
				countStr = fmt.Sprintf(" ×%d", fp.Count)
			}
			fmt.Printf("  %s%s%s  %-22s  %s%s\n",
				color, f.Rarity.Badge(), colorReset,
				f.Name+countStr,
				f.Description,
				"",
			)
		} else {
			fmt.Printf("  %s%s%s  %-22s  %s\n",
				color, f.Rarity.Badge(), colorReset,
				"░░░ [locked]",
				"???",
			)
		}
	}

	fmt.Println()
	if p.Sessions.Total > 0 {
		fmt.Printf("  Sessions analyzed: %d  |  Streak: %d days\n\n",
			p.Sessions.Total, p.Streak.Current)
	}

	return nil
}

func supportsColor() bool {
	term := os.Getenv("TERM")
	colorTerm := os.Getenv("COLORTERM")
	noColor := os.Getenv("NO_COLOR")
	return noColor == "" && (term != "" && term != "dumb") || colorTerm != ""
}
