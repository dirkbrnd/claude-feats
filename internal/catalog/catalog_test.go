package catalog_test

import (
	"testing"

	"github.com/dirkbrnd/claude-feats/internal/catalog"
)

func TestByID(t *testing.T) {
	f := catalog.ByID("vimreflex")
	if f == nil {
		t.Fatal("vimreflex not found")
	}
	if f.Name != ":wq" {
		t.Errorf("name = %q, want \":wq\"", f.Name)
	}
	if catalog.ByID("nonexistent") != nil {
		t.Error("nonexistent ID should return nil")
	}
}

func TestVisible_ExcludesHidden(t *testing.T) {
	visible := catalog.Visible()
	for _, f := range visible {
		if f.Hidden {
			t.Errorf("feat %q is hidden but appeared in Visible()", f.ID)
		}
	}
}

func TestAllContainsHidden(t *testing.T) {
	hasHidden := false
	for _, f := range catalog.All {
		if f.Hidden {
			hasHidden = true
			break
		}
	}
	if !hasHidden {
		t.Error("All should contain at least one hidden feat")
	}
}

func TestIsFibonacci(t *testing.T) {
	fibs := []int{1, 2, 3, 5, 8, 13, 21, 34, 55, 89}
	nonFibs := []int{4, 6, 7, 9, 10, 100}

	for _, n := range fibs {
		if !catalog.IsFibonacci(n) {
			t.Errorf("IsFibonacci(%d) = false, want true", n)
		}
	}
	for _, n := range nonFibs {
		if catalog.IsFibonacci(n) {
			t.Errorf("IsFibonacci(%d) = true, want false", n)
		}
	}
	if catalog.IsFibonacci(0) {
		t.Error("IsFibonacci(0) should be false")
	}
}

func TestIsPalindrome(t *testing.T) {
	palindromes := []int{1, 11, 22, 33, 101, 111, 121, 1001}
	notPalindromes := []int{12, 23, 100, 123}

	for _, n := range palindromes {
		if !catalog.IsPalindrome(n) {
			t.Errorf("IsPalindrome(%d) = false, want true", n)
		}
	}
	for _, n := range notPalindromes {
		if catalog.IsPalindrome(n) {
			t.Errorf("IsPalindrome(%d) = true, want false", n)
		}
	}
}

func TestRarityOrder(t *testing.T) {
	if catalog.Legendary.Order() <= catalog.Epic.Order() {
		t.Error("Legendary should outrank Epic")
	}
	if catalog.Epic.Order() <= catalog.Rare.Order() {
		t.Error("Epic should outrank Rare")
	}
	if catalog.Rare.Order() <= catalog.Uncommon.Order() {
		t.Error("Rare should outrank Uncommon")
	}
	if catalog.Uncommon.Order() <= catalog.Common.Order() {
		t.Error("Uncommon should outrank Common")
	}
}

func TestAllFeatIDsUnique(t *testing.T) {
	seen := make(map[string]struct{})
	for _, f := range catalog.All {
		if _, dup := seen[f.ID]; dup {
			t.Errorf("duplicate feat ID: %q", f.ID)
		}
		seen[f.ID] = struct{}{}
	}
}

func TestManaFeatIDsPresent(t *testing.T) {
	required := []string{
		"apprenticemage", "archmage", "thevoid",
		"manaburn", "precisioncast", "frugalmage",
		"incantation", "grimoire",
		"codexinfinitus",
	}
	for _, id := range required {
		if catalog.ByID(id) == nil {
			t.Errorf("mana feat %q not found in catalog", id)
		}
	}
}

func TestCodexInfinitusIsHidden(t *testing.T) {
	f := catalog.ByID("codexinfinitus")
	if f == nil {
		t.Fatal("codexinfinitus not in catalog")
	}
	if !f.Hidden {
		t.Error("codexinfinitus should be hidden")
	}
}
