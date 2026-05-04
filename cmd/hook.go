package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var hookInstallSkill bool

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Manage the Claude Code Stop hook",
}

var hookInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Wire claude-feats into Claude Code's Stop hook",
	RunE:  runHookInstall,
}

var hookUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove claude-feats from the Stop hook",
	RunE:  runHookUninstall,
}

func init() {
	hookInstallCmd.Flags().BoolVar(&hookInstallSkill, "skill", false, "Also install the SKILL.md agent skill")
	hookCmd.AddCommand(hookInstallCmd)
	hookCmd.AddCommand(hookUninstallCmd)
}

// settingsHook is the minimal struct we need for the Stop hook entry.
type settingsHook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type settingsHookEntry struct {
	Matcher string         `json:"matcher"`
	Hooks   []settingsHook `json:"hooks"`
}

func runHookInstall(_ *cobra.Command, _ []string) error {
	settingsPath, err := claudeSettingsPath()
	if err != nil {
		return err
	}

	settings, err := loadSettings(settingsPath)
	if err != nil {
		return err
	}

	// Check if already installed
	if isHookInstalled(settings) {
		fmt.Println("claude-feats hook is already installed.")
	} else {
		entry := settingsHookEntry{
			Matcher: "",
			Hooks:   []settingsHook{{Type: "command", Command: "claude-feats check"}},
		}
		stopHooks := toSlice(settings["Stop"])
		stopHooks = append(stopHooks, entry)
		settings["Stop"] = stopHooks

		if err := saveSettings(settingsPath, settings); err != nil {
			return err
		}
		fmt.Printf("✓ Hook installed in %s\n", settingsPath)
	}

	if hookInstallSkill {
		if err := installSkill(); err != nil {
			fmt.Printf("Warning: could not install SKILL.md: %v\n", err)
		} else {
			fmt.Println("✓ Agent skill installed in ~/.claude/skills/claude-feats/")
		}
	}

	return nil
}

func runHookUninstall(_ *cobra.Command, _ []string) error {
	settingsPath, err := claudeSettingsPath()
	if err != nil {
		return err
	}

	settings, err := loadSettings(settingsPath)
	if err != nil {
		return err
	}

	if !isHookInstalled(settings) {
		fmt.Println("claude-feats hook is not installed.")
		return nil
	}

	stopHooks := toSlice(settings["Stop"])
	var filtered []interface{}
	for _, h := range stopHooks {
		if !entryHasClaudeFeats(h) {
			filtered = append(filtered, h)
		}
	}
	if len(filtered) == 0 {
		delete(settings, "Stop")
	} else {
		settings["Stop"] = filtered
	}

	if err := saveSettings(settingsPath, settings); err != nil {
		return err
	}
	fmt.Printf("✓ Hook removed from %s\n", settingsPath)
	return nil
}

func claudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func loadSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(map[string]interface{}), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read settings: %w", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}
	return m, nil
}

func saveSettings(path string, settings map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func isHookInstalled(settings map[string]interface{}) bool {
	for _, h := range toSlice(settings["Stop"]) {
		if entryHasClaudeFeats(h) {
			return true
		}
	}
	return false
}

func entryHasClaudeFeats(v interface{}) bool {
	data, err := json.Marshal(v)
	if err != nil {
		return false
	}
	var entry settingsHookEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return false
	}
	for _, h := range entry.Hooks {
		if h.Command == "claude-feats check" {
			return true
		}
	}
	return false
}

func toSlice(v interface{}) []interface{} {
	if v == nil {
		return nil
	}
	s, ok := v.([]interface{})
	if !ok {
		return nil
	}
	return s
}

func installSkill() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	skillDir := filepath.Join(home, ".claude", "skills", "claude-feats")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		return err
	}

	// Try to find the bundled SKILL.md relative to the binary
	exe, _ := os.Executable()
	candidates := []string{
		filepath.Join(filepath.Dir(exe), "..", "skills", "claude-feats", "SKILL.md"),
		filepath.Join(filepath.Dir(exe), "skills", "claude-feats", "SKILL.md"),
		"skills/claude-feats/SKILL.md",
	}
	for _, src := range candidates {
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), data, 0o644)
	}

	// Fall back to writing the embedded content
	return os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(embeddedSkillMD), 0o644)
}

// embeddedSkillMD is a fallback copy of the skill content.
const embeddedSkillMD = `# claude-feats

claude-feats is a Go CLI tool that tracks RPG-style achievements across Claude Code sessions.
It hooks into Claude Code's Stop event and analyzes transcripts to unlock feats.

## Commands

- ` + "`claude-feats list`" + ` — show all visible feats (unlocked and locked)
- ` + "`claude-feats list --unlocked`" + ` — show only unlocked feats
- ` + "`claude-feats stats`" + ` — full display with category progress bars and streaks
- ` + "`claude-feats mana`" + ` — monthly token usage chart and lifetime total
- ` + "`claude-feats bio`" + ` — generate an RPG character bio based on your feat history
- ` + "`claude-feats export`" + ` — dump progress to markdown for sharing
- ` + "`claude-feats hook install`" + ` — wire into Claude Code's Stop hook

## Answering user questions

When the user asks "how many feats do I have?" or similar:
1. Run ` + "`claude-feats list --unlocked`" + ` and count the results.
2. Report the count and highlight any Legendary or Epic feats.

When the user asks about their coding style or wants a summary:
- Suggest running ` + "`claude-feats bio`" + ` for a personalized RPG character bio.

When the user asks about their token usage:
- Run ` + "`claude-feats mana`" + ` to show the monthly breakdown.

## Hidden feats

There are feats that cannot be seen in the list — they only appear when first unlocked.
You may hint that they exist ("there are feats you haven't discovered yet") but never
name or describe them. Let the user find them organically.
`
