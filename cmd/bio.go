package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/spf13/cobra"

	"github.com/dirkbrnd/claude-feats/internal/catalog"
	"github.com/dirkbrnd/claude-feats/internal/store"
)

var bioCmd = &cobra.Command{
	Use:   "bio",
	Short: "Generate your RPG character bio via the Anthropic API",
	RunE:  runBio,
}

func runBio(_ *cobra.Command, _ []string) error {
	p, err := store.Load()
	if err != nil {
		return fmt.Errorf("load progress: %w", err)
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		// Try to read from ~/.claude/.credentials or claude config
		apiKey = readClaudeAPIKey()
	}
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY not set — export it or run 'claude' once to authenticate")
	}

	prompt := buildBioPrompt(p)

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	fmt.Print("\n  Channeling the arcane...\n\n")

	stream := client.Messages.NewStreaming(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeHaiku4_5,
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})

	// Typewriter effect via streaming
	for stream.Next() {
		event := stream.Current()
		if cbde, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
			if td := cbde.Delta.AsTextDelta(); td.Text != "" {
				fmt.Print(td.Text)
			}
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("stream error: %w", err)
	}

	fmt.Print("\n\n")
	return nil
}

func buildBioPrompt(p *store.Progress) string {
	var sb strings.Builder

	sb.WriteString("You are a dramatic RPG narrator. Generate a short, vivid character bio (3–4 paragraphs) for a developer based on their coding achievements. Be creative, use fantasy/RPG language, but ground it in the real data. No headers or bullet points — pure prose.\n\n")
	sb.WriteString("CHARACTER DATA:\n\n")

	sb.WriteString(fmt.Sprintf("Sessions analyzed: %d\n", p.Sessions.Total))
	sb.WriteString(fmt.Sprintf("Current streak: %d days (longest: %d)\n", p.Streak.Current, p.Streak.Longest))
	sb.WriteString(fmt.Sprintf("Lifetime tokens: %s\n", formatTokens(p.Mana.LifetimeTotal())))

	// Session timing tendencies
	dayNames := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	maxDay, maxCount := 0, 0
	for i, c := range p.Sessions.ByDayOfWeek {
		if c > maxCount {
			maxCount = c
			maxDay = i
		}
	}
	if maxCount > 0 {
		sb.WriteString(fmt.Sprintf("Most active day: %s\n", dayNames[maxDay]))
	}

	// Unlocked feats
	type unlockedFeat struct {
		feat catalog.Feat
		fp   store.FeatProgress
	}
	var unlocked []unlockedFeat
	for _, f := range catalog.All {
		if fp, ok := p.Feats[f.ID]; ok {
			unlocked = append(unlocked, unlockedFeat{f, fp})
		}
	}
	sort.Slice(unlocked, func(i, j int) bool {
		if unlocked[i].feat.Rarity.Order() != unlocked[j].feat.Rarity.Order() {
			return unlocked[i].feat.Rarity.Order() > unlocked[j].feat.Rarity.Order()
		}
		return unlocked[i].fp.Count > unlocked[j].fp.Count
	})

	sb.WriteString(fmt.Sprintf("\nUnlocked feats (%d total):\n", len(unlocked)))
	for _, u := range unlocked {
		line := fmt.Sprintf("  - [%s] %s", u.feat.Rarity, u.feat.Name)
		if u.feat.Countable && u.fp.Count > 1 {
			line += fmt.Sprintf(" (×%d)", u.fp.Count)
		}
		line += ": " + u.feat.Description
		sb.WriteString(line + "\n")
	}

	if len(unlocked) == 0 {
		sb.WriteString("  (no feats yet — just beginning the journey)\n")
	}

	sb.WriteString(fmt.Sprintf("\nInstalled: %s\n", p.InstalledAt.Format(time.DateOnly)))
	sb.WriteString("\nWrite the bio now:")

	return sb.String()
}

// readClaudeAPIKey attempts to read the API key from Claude's config.
func readClaudeAPIKey() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	// Claude stores credentials in ~/.claude/.credentials.json or similar
	paths := []string{
		home + "/.claude/.credentials",
		home + "/.claude/credentials",
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		// Simple extraction — look for "api_key": "sk-ant-..."
		s := string(data)
		if idx := strings.Index(s, "sk-ant-"); idx >= 0 {
			end := strings.IndexAny(s[idx:], `"' `)
			if end > 0 {
				return s[idx : idx+end]
			}
		}
	}
	return ""
}
