package catalog

import (
	"fmt"
	"time"
)

// Rarity tiers
type Rarity string

const (
	Common    Rarity = "Common"
	Uncommon  Rarity = "Uncommon"
	Rare      Rarity = "Rare"
	Epic      Rarity = "Epic"
	Legendary Rarity = "Legendary"
)

// Feat defines a single achievement.
type Feat struct {
	ID          string
	Name        string
	Description string
	Rarity      Rarity
	Hidden      bool
	Countable   bool // can be unlocked multiple times
}

// Color returns ANSI color code for the rarity.
func (r Rarity) Color() string {
	switch r {
	case Common:
		return "\033[37m"
	case Uncommon:
		return "\033[32m"
	case Rare:
		return "\033[34m"
	case Epic:
		return "\033[35m"
	case Legendary:
		return "\033[33m"
	default:
		return "\033[0m"
	}
}

// Badge returns a short ASCII label for the rarity tier.
func (r Rarity) Badge() string {
	switch r {
	case Common:
		return "[ C ]"
	case Uncommon:
		return "[UCO]"
	case Rare:
		return "[RAR]"
	case Epic:
		return "[EPC]"
	case Legendary:
		return "[LEG]"
	default:
		return "[???]"
	}
}

// Order returns sort weight (highest rarity first).
func (r Rarity) Order() int {
	switch r {
	case Legendary:
		return 5
	case Epic:
		return 4
	case Rare:
		return 3
	case Uncommon:
		return 2
	case Common:
		return 1
	default:
		return 0
	}
}

// All is the complete feat catalog, visible and hidden.
var All = []Feat{
	// ── Behavioral ──────────────────────────────────────────────────────────
	{ID: "vimreflex", Name: ":wq", Description: "Typed :wq into the chat. Old habits die hard.", Rarity: Rare, Countable: true},
	{ID: "oneword", Name: "The Sovereign", Description: "Average user message under 10 characters. You speak little; you mean everything.", Rarity: Uncommon, Countable: true},
	{ID: "speedrun", Name: "Speedrun", Description: "Session wall time under 5 minutes. Blink and you'd miss it.", Rarity: Uncommon, Countable: true},
	{ID: "triage", Name: "Triage Master", Description: "Dispatched a numbered list of 5+ fix/skip decisions in a single message.", Rarity: Rare, Countable: true},
	{ID: "interrupt", Name: "Interrupt Strike", Description: "Pulled the cord mid-session. Decisive.", Rarity: Common, Countable: true},
	{ID: "oneshot", Name: "One and Done", Description: "A single message. A complete session. Pure signal.", Rarity: Epic, Countable: true},
	{ID: "longhaul", Name: "Marathon", Description: "Session ran longer than 4 hours. Endurance athlete.", Rarity: Rare, Countable: true},
	{ID: "perfectionist", Name: "Perfectionist", Description: "Edited the same file 10+ times in one session.", Rarity: Uncommon, Countable: true},

	// ── Temporal ────────────────────────────────────────────────────────────
	{ID: "earlybird", Name: "Early Bird", Description: "Started a session between 5:00am and 7:00am. The worm is yours.", Rarity: Uncommon, Countable: true},
	{ID: "nightowl", Name: "Night Owl", Description: "Session started between midnight and 5:00am. Sleep is optional.", Rarity: Rare, Countable: true},
	{ID: "weekendwarrior", Name: "Weekend Warrior", Description: "Shipped on a Saturday or Sunday. No days off.", Rarity: Uncommon, Countable: true},
	{ID: "holidayhacker", Name: "Holiday Hacker", Description: "Coded on a public holiday. Family can wait.", Rarity: Epic, Countable: true},

	// ── Milestone / Streak ──────────────────────────────────────────────────
	{ID: "firstblood", Name: "First Blood", Description: "Your very first session analyzed. The journey begins.", Rarity: Common, Countable: false},
	{ID: "luckyseven", Name: "Lucky Seven", Description: "Session 7, 77, or 777. Fortune favors the consistent.", Rarity: Rare, Countable: false},
	{ID: "century", Name: "The Century", Description: "100 sessions analyzed. A century of collaboration.", Rarity: Epic, Countable: false},
	{ID: "streak7", Name: "On a Roll", Description: "7 consecutive days with at least one session.", Rarity: Uncommon, Countable: true},
	{ID: "streak30", Name: "Relentless", Description: "30 consecutive days. A habit forged in fire.", Rarity: Epic, Countable: true},
	{ID: "streak365", Name: "Transcendent", Description: "365 consecutive days. You are the machine.", Rarity: Legendary, Countable: true},
	{ID: "anniversary", Name: "Anniversary", Description: "One full year since installing claude-feats.", Rarity: Legendary, Countable: false},
	{ID: "rarecollector", Name: "Rare Collector", Description: "Unlocked 10 or more Rare+ feats.", Rarity: Epic, Countable: false},

	// ── Git-based ───────────────────────────────────────────────────────────
	{ID: "timetraveler", Name: "Time Traveler", Description: "Rewound history with git revert, reset, or checkout HEAD~.", Rarity: Uncommon, Countable: true},
	{ID: "archaeologist", Name: "Archaeologist", Description: "Worked in a repo with files older than 5 years.", Rarity: Rare, Countable: true},
	{ID: "midnightship", Name: "Midnight Ship", Description: "Pushed a commit between midnight and 3am.", Rarity: Rare, Countable: true},
	{ID: "neveronsunday", Name: "Never on Sunday", Description: "30 sessions analyzed with zero on a Sunday. Principled.", Rarity: Epic, Countable: false},
	{ID: "worktreeclone", Name: "Shadow Clone", Description: "Created a git worktree during the session.", Rarity: Common, Countable: true},
	{ID: "sovereign", Name: "Platform Sovereign", Description: "Touched files with 5 or more different extensions in one session.", Rarity: Legendary, Countable: true},

	// ── Meta ────────────────────────────────────────────────────────────────
	{ID: "earlyinstall", Name: "Early Adopter", Description: "Installed claude-feats within 30 days of first public release.", Rarity: Epic, Countable: false},

	// ── Mana — monthly (Option A) ────────────────────────────────────────────
	{ID: "apprenticemage", Name: "Apprentice Mage", Description: "100K tokens in a single calendar month.", Rarity: Common, Countable: false},
	{ID: "archmage", Name: "Archmage", Description: "1M tokens in a single calendar month.", Rarity: Epic, Countable: false},
	{ID: "thevoid", Name: "The Void", Description: "10M tokens in a single calendar month. Unfathomable.", Rarity: Legendary, Countable: false},

	// ── Mana — efficiency (Option B) ────────────────────────────────────────
	{ID: "manaburn", Name: "Mana Burn", Description: "Single session over 100K tokens. You spent big.", Rarity: Rare, Countable: true},
	{ID: "precisioncast", Name: "Precision Cast", Description: "Under 2K tokens total and still shipped a commit.", Rarity: Uncommon, Countable: true},
	{ID: "frugalmage", Name: "The Frugal Mage", Description: "Monthly avg tokens/session lower than the previous month.", Rarity: Rare, Countable: true},

	// ── Mana — lifetime (Option C) ───────────────────────────────────────────
	{ID: "incantation", Name: "Incantation", Description: "1 million lifetime tokens cast.", Rarity: Uncommon, Countable: false},
	{ID: "grimoire", Name: "Grimoire", Description: "10 million lifetime tokens. The tome grows heavy.", Rarity: Epic, Countable: false},

	// ── Hidden feats ────────────────────────────────────────────────────────
	{ID: "ghostmode", Name: "Ghost Mode", Description: "A session with no user messages — pure tool-use.", Rarity: Epic, Hidden: true, Countable: true},
	{ID: "fibonacci", Name: "Golden Spiral", Description: "Session number fell on a Fibonacci number.", Rarity: Rare, Hidden: true, Countable: true},
	{ID: "conspiracy", Name: "3:33", Description: "Session ended at exactly 3:33am.", Rarity: Legendary, Hidden: true, Countable: true},
	{ID: "dejavu", Name: "Déjà Vu", Description: "You've started here before.", Rarity: Rare, Hidden: true, Countable: true},
	{ID: "silentnight", Name: "Silent Night", Description: "Some things are sacred.", Rarity: Legendary, Hidden: true, Countable: true},
	{ID: "oops", Name: "Oops", Description: "You git reset --hard or rm -rf'd something. Brave.", Rarity: Uncommon, Hidden: true, Countable: true},
	{ID: "yolo", Name: "YOLO", Description: "Used --force or -f. No safety net.", Rarity: Common, Hidden: true, Countable: true},
	{ID: "palindrome", Name: "Palindrome", Description: "Session number reads the same forwards and backwards.", Rarity: Uncommon, Hidden: true, Countable: true},
	{ID: "ouroboros", Name: "Ouroboros", Description: "The snake that eats its own tail.", Rarity: Legendary, Hidden: true, Countable: true},
	{ID: "inception", Name: "Inception", Description: "Dreams within dreams.", Rarity: Legendary, Hidden: true, Countable: true},
	{ID: "codexinfinitus", Name: "Codex Infinitus", Description: "100 million lifetime tokens. You are beyond measure.", Rarity: Legendary, Hidden: true, Countable: false},
}

// ByID returns a feat by ID, or nil.
func ByID(id string) *Feat {
	for i := range All {
		if All[i].ID == id {
			return &All[i]
		}
	}
	return nil
}

// Visible returns all non-hidden feats.
func Visible() []Feat {
	var out []Feat
	for _, f := range All {
		if !f.Hidden {
			out = append(out, f)
		}
	}
	return out
}

// IsUSHoliday returns whether a given date is a US federal public holiday.
func IsUSHoliday(t time.Time) bool {
	m, d := t.Month(), t.Day()
	wd := t.Weekday()

	switch {
	case m == 1 && d == 1: // New Year's Day
		return true
	case m == 7 && d == 4: // Independence Day
		return true
	case m == 11 && d == 11: // Veterans Day
		return true
	case m == 12 && d == 25: // Christmas
		return true
	case m == 1 && wd == time.Monday && d >= 15 && d <= 21: // MLK Day
		return true
	case m == 2 && wd == time.Monday && d >= 15 && d <= 21: // Presidents Day
		return true
	case m == 5 && wd == time.Monday && d >= 25: // Memorial Day
		return true
	case m == 9 && wd == time.Monday && d <= 7: // Labor Day
		return true
	case m == 10 && wd == time.Monday && d >= 8 && d <= 14: // Columbus Day
		return true
	case m == 11 && wd == time.Thursday && d >= 22 && d <= 28: // Thanksgiving
		return true
	}
	return false
}

// IsFibonacci returns true if n is a positive Fibonacci number.
func IsFibonacci(n int) bool {
	if n <= 0 {
		return false
	}
	a, b := 0, 1
	for b < n {
		a, b = b, a+b
	}
	return b == n
}

// IsPalindrome returns true if n is a palindrome number.
func IsPalindrome(n int) bool {
	if n < 0 {
		return false
	}
	s := fmt.Sprintf("%d", n)
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		if s[i] != s[j] {
			return false
		}
	}
	return true
}
