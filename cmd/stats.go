package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/dirkbrand/claude-feats/internal/catalog"
	"github.com/dirkbrand/claude-feats/internal/store"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Full feat display with category bars, streaks, and records",
	RunE:  runStats,
}

func runStats(_ *cobra.Command, _ []string) error {
	p, err := store.Load()
	if err != nil {
		return fmt.Errorf("load progress: %w", err)
	}

	useColor := supportsColor()

	printBanner(useColor)

	// Group visible feats by rarity (highest first)
	rarities := []catalog.Rarity{
		catalog.Legendary,
		catalog.Epic,
		catalog.Rare,
		catalog.Uncommon,
		catalog.Common,
	}

	for _, r := range rarities {
		var feats []catalog.Feat
		for _, f := range catalog.Visible() {
			if f.Rarity == r {
				feats = append(feats, f)
			}
		}
		sort.Slice(feats, func(i, j int) bool { return feats[i].Name < feats[j].Name })

		unlockedCount := 0
		for _, f := range feats {
			if _, ok := p.Feats[f.ID]; ok {
				unlockedCount++
			}
		}

		color := ""
		colorReset := ""
		if useColor {
			color = r.Color()
			colorReset = reset
		}

		bar := progressBar(unlockedCount, len(feats), 12)
		fmt.Printf("\n  %s%s %-12s%s  %s  %d/%d\n",
			color, r.Badge(), string(r), colorReset,
			bar, unlockedCount, len(feats),
		)

		for _, f := range feats {
			fp, isUnlocked := p.Feats[f.ID]
			if isUnlocked {
				countStr := ""
				if f.Countable && fp.Count > 1 {
					countStr = fmt.Sprintf(" ×%d", fp.Count)
				}
				prStr := ""
				if fp.PersonalRecord != nil {
					prStr = fmt.Sprintf(" [PR: %d]", *fp.PersonalRecord)
				}
				fmt.Printf("    %s✓%s  %-22s%s%s\n",
					color, colorReset, f.Name+countStr, prStr, "")
			} else {
				fmt.Printf("    %s░%s  %-22s\n", color, colorReset, "░░░ [locked]")
			}
		}
	}

	// Streak
	fmt.Printf("\n  ┌─ Streak ──────────────────────────────\n")
	fmt.Printf("  │  Current:  %d days\n", p.Streak.Current)
	fmt.Printf("  │  Longest:  %d days\n", p.Streak.Longest)
	if p.Streak.LastActiveDate != "" {
		fmt.Printf("  │  Last:     %s\n", p.Streak.LastActiveDate)
	}
	fmt.Printf("  └───────────────────────────────────────\n")

	// Sessions by day of week
	days := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	fmt.Printf("\n  ┌─ Sessions by Day ─────────────────────\n")
	for i, d := range days {
		bar := miniBar(p.Sessions.ByDayOfWeek[i], p.Sessions.Total)
		fmt.Printf("  │  %s  %s %d\n", d, bar, p.Sessions.ByDayOfWeek[i])
	}
	fmt.Printf("  └───────────────────────────────────────\n")
	fmt.Printf("\n  Total sessions: %d\n\n", p.Sessions.Total)

	return nil
}

func printBanner(color bool) {
	c := ""
	r := ""
	if color {
		c = "\033[33m"
		r = reset
	}
	fmt.Printf("\n%s", c)
	fmt.Println("  ╔══════════════════════════════════════╗")
	fmt.Println("  ║         ⚔  claude-feats  ⚔           ║")
	fmt.Println("  ╚══════════════════════════════════════╝")
	fmt.Printf("%s", r)
}

func progressBar(done, total, width int) string {
	if total == 0 {
		return "[" + repeat("░", width) + "]"
	}
	filled := done * width / total
	return "[" + repeat("█", filled) + repeat("░", width-filled) + "]"
}

func miniBar(n, max int) string {
	if max == 0 {
		return "          "
	}
	filled := n * 10 / max
	return repeat("▓", filled) + repeat("░", 10-filled)
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
