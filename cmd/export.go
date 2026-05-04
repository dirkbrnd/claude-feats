package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/dirkbrnd/claude-feats/internal/catalog"
	"github.com/dirkbrnd/claude-feats/internal/store"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Dump feat progress to markdown (for Slack, Notion, etc.)",
	RunE:  runExport,
}

func runExport(_ *cobra.Command, _ []string) error {
	p, err := store.Load()
	if err != nil {
		return fmt.Errorf("load progress: %w", err)
	}

	fmt.Println(buildMarkdown(p))
	return nil
}

func buildMarkdown(p *store.Progress) string {
	var sb strings.Builder

	sb.WriteString("# ⚔️ claude-feats\n\n")
	sb.WriteString(fmt.Sprintf("*%d sessions analyzed · %d-day streak · %s tokens cast*\n\n",
		p.Sessions.Total, p.Streak.Current, formatTokens(p.Mana.LifetimeTotal())))

	rarities := []catalog.Rarity{
		catalog.Legendary, catalog.Epic, catalog.Rare, catalog.Uncommon, catalog.Common,
	}

	for _, r := range rarities {
		var feats []catalog.Feat
		for _, f := range catalog.Visible() {
			if f.Rarity == r {
				feats = append(feats, f)
			}
		}
		sort.Slice(feats, func(i, j int) bool { return feats[i].Name < feats[j].Name })

		var lines []string
		for _, f := range feats {
			fp, ok := p.Feats[f.ID]
			if !ok {
				continue
			}
			line := fmt.Sprintf("**%s**", f.Name)
			if f.Countable && fp.Count > 1 {
				line += fmt.Sprintf(" ×%d", fp.Count)
			}
			line += " — " + f.Description
			lines = append(lines, line)
		}
		if len(lines) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("## %s %s\n\n", r.Badge(), r))
		for _, l := range lines {
			sb.WriteString("- " + l + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("---\n*Generated %s with [claude-feats](https://github.com/dirkbrnd/claude-feats)*\n",
		time.Now().Format(time.DateOnly)))

	return sb.String()
}
