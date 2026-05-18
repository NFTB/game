package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGameplayData(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "game_rules.json"), `{
		"players": {"min": 2, "max": 4, "initial_gold": 1000},
		"rounds": {"count": 5, "time_limit_seconds": 30},
		"bidding": {"min_bid": 1},
		"tiebreaker": {"max_rounds": 10}
	}`)
	writeFile(t, filepath.Join(dir, "collectibles.json"), `[
		{"id": "c_1", "name": "藏品", "type": "瓷器", "tier": "R", "outline": "1x1", "true_value": 100, "sell_value": 100, "size": [1, 1]}
	]`)

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
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
