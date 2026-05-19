package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadGameplayData(t *testing.T) {
	dir := t.TempDir()
	writeValidGameplayConfig(t, dir)

	data, err := LoadGameplayData(dir)
	if err != nil {
		t.Fatalf("load gameplay data: %v", err)
	}
	if data.Rules.Players.InitialGold != 1000 {
		t.Fatalf("initial gold = %d, want 1000", data.Rules.Players.InitialGold)
	}
	if len(data.Collectibles) != 1 || data.Collectibles[0].ID != "c_1" {
		t.Fatalf("collectibles = %+v, want c_1", data.Collectibles)
	}
	if len(data.Venues) != 1 || data.Venues[0].RoundEntryFee != 10 {
		t.Fatalf("venues = %+v, want entry fee 10", data.Venues)
	}
}

func TestLoadGameplayDataRejectsInvalidVenueRange(t *testing.T) {
	dir := t.TempDir()
	writeValidGameplayRules(t, dir)
	writeFile(t, filepath.Join(dir, "venues.json"), `[
		{"id": "v_1", "name": "场所", "round_entry_fee": 10, "collectible_count_range": {"min": 6, "max": 5}}
	]`)
	writeValidCollectibles(t, dir)

	_, err := LoadGameplayData(dir)
	if err == nil || !strings.Contains(err.Error(), "invalid venue collectible count range") {
		t.Fatalf("load error = %v, want invalid venue collectible count range", err)
	}
}

func TestLoadGameplayDataRejectsDuplicateCollectibleIDs(t *testing.T) {
	dir := t.TempDir()
	writeValidGameplayRules(t, dir)
	writeValidVenues(t, dir)
	writeFile(t, filepath.Join(dir, "collectibles.json"), `[
		{"id": "c_1", "name": "藏品一", "type": "瓷器", "tier": "R", "outline": "1x1", "true_value": 100, "sell_value": 100, "size": [1, 1]},
		{"id": "c_1", "name": "藏品二", "type": "玉器", "tier": "N", "outline": "1x1", "true_value": 50, "sell_value": 50, "size": [1, 1]}
	]`)

	_, err := LoadGameplayData(dir)
	if err == nil || !strings.Contains(err.Error(), "duplicate collectible id") {
		t.Fatalf("load error = %v, want duplicate collectible id", err)
	}
}

func writeValidGameplayConfig(t *testing.T, dir string) {
	t.Helper()

	writeValidGameplayRules(t, dir)
	writeValidVenues(t, dir)
	writeValidCollectibles(t, dir)
}

func writeValidGameplayRules(t *testing.T, dir string) {
	t.Helper()

	writeFile(t, filepath.Join(dir, "game_rules.json"), `{
		"players": {"min": 2, "max": 4, "initial_gold": 1000},
		"rounds": {"count": 5, "time_limit_seconds": 30},
		"bidding": {"min_bid": 1},
		"tiebreaker": {"max_rounds": 10}
	}`)
}

func writeValidVenues(t *testing.T, dir string) {
	t.Helper()

	writeFile(t, filepath.Join(dir, "venues.json"), `[
		{"id": "v_1", "name": "场所", "round_entry_fee": 10, "collectible_count_range": {"min": 3, "max": 5}}
	]`)
}

func writeValidCollectibles(t *testing.T, dir string) {
	t.Helper()

	writeFile(t, filepath.Join(dir, "collectibles.json"), `[
		{"id": "c_1", "name": "藏品", "type": "瓷器", "tier": "R", "outline": "1x1", "true_value": 100, "sell_value": 100, "size": [1, 1]}
	]`)
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
