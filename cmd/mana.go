package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/dirkbrnd/claude-feats/internal/store"
)

var manaCmd = &cobra.Command{
	Use:   "mana",
	Short: "Monthly mana (token) usage bar chart and lifetime spellbook",
	RunE:  runMana,
}

func runMana(_ *cobra.Command, _ []string) error {
	p, err := store.Load()
	if err != nil {
		return fmt.Errorf("load progress: %w", err)
	}

	useColor := supportsColor()
	c := ""
	r := ""
	if useColor {
		c = "\033[34m" // blue for mana
		r = reset
	}

	fmt.Printf("\n%s  ╔══════════════════════════════════════╗\n", c)
	fmt.Printf("  ║         🔮  Mana Spellbook  🔮         ║\n")
	fmt.Printf("  ╚══════════════════════════════════════╝%s\n\n", r)

	// Lifetime
	lt := p.Mana.Lifetime
	fmt.Printf("  Lifetime tokens cast: %s\n", formatTokens(lt.Total()))
	fmt.Printf("    Input:         %s\n", formatTokens(lt.InputTokens))
	fmt.Printf("    Output:        %s\n", formatTokens(lt.OutputTokens))
	fmt.Printf("    Cache created: %s\n", formatTokens(lt.CacheCreationInputTokens))
	fmt.Printf("    Cache read:    %s\n\n", formatTokens(lt.CacheReadInputTokens))

	if len(p.Mana.Monthly) == 0 {
		fmt.Println("  No monthly data yet.")
		fmt.Println()
		return nil
	}

	// Sort months
	months := make([]string, 0, len(p.Mana.Monthly))
	for k := range p.Mana.Monthly {
		months = append(months, k)
	}
	sort.Strings(months)

	// Find max for bar scaling
	maxTotal := 0
	for _, k := range months {
		m := p.Mana.Monthly[k]
		t := m.InputTokens + m.OutputTokens
		if t > maxTotal {
			maxTotal = t
		}
	}

	fmt.Printf("  Monthly usage (last 12 months):\n\n")
	start := len(months) - 12
	if start < 0 {
		start = 0
	}
	for _, k := range months[start:] {
		m := p.Mana.Monthly[k]
		total := m.InputTokens + m.OutputTokens
		barWidth := 20
		filled := 0
		if maxTotal > 0 {
			filled = total * barWidth / maxTotal
		}
		label := monthLabel(k)
		bar := repeat("█", filled) + repeat("░", barWidth-filled)
		fmt.Printf("  %s  %s%s%s  %s\n", label, c, bar, r, formatTokens(total))
	}
	fmt.Println()

	return nil
}

func formatTokens(n int) string {
	switch {
	case n >= 1_000_000_000:
		return fmt.Sprintf("%.2fB", float64(n)/1_000_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.2fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func monthLabel(key string) string {
	t, err := time.Parse("2006-01", key)
	if err != nil {
		return key
	}
	return t.Format("Jan 2006")
}
