package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type GameplayData struct {
	Rules        GameRulesData
	Venues       []VenueData
	Collectibles []CollectibleData
}

type GameRulesData struct {
	Players struct {
		Min         int `json:"min"`
		Max         int `json:"max"`
		InitialGold int `json:"initial_gold"`
	} `json:"players"`
	Rounds struct {
		Count            int `json:"count"`
		TimeLimitSeconds int `json:"time_limit_seconds"`
	} `json:"rounds"`
	Bidding struct {
		MinBid int `json:"min_bid"`
	} `json:"bidding"`
	Tiebreaker struct {
		MaxRounds int `json:"max_rounds"`
	} `json:"tiebreaker"`
}

type CollectibleData struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Tier      string `json:"tier"`
	Outline   string `json:"outline"`
	TrueValue int    `json:"true_value"`
	SellValue int    `json:"sell_value"`
	Size      []int  `json:"size"`
}

type VenueData struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	RoundEntryFee         int    `json:"round_entry_fee"`
	CollectibleCountRange struct {
		Min int `json:"min"`
		Max int `json:"max"`
	} `json:"collectible_count_range"`
}

func LoadGameplayData(configDir string) (GameplayData, error) {
	var data GameplayData
	if err := readJSON(filepath.Join(configDir, "game_rules.json"), &data.Rules); err != nil {
		return GameplayData{}, err
	}
	if err := readJSON(filepath.Join(configDir, "venues.json"), &data.Venues); err != nil {
		return GameplayData{}, err
	}
	if err := readJSON(filepath.Join(configDir, "collectibles.json"), &data.Collectibles); err != nil {
		return GameplayData{}, err
	}
	if len(data.Venues) == 0 {
		return GameplayData{}, fmt.Errorf("venues config is empty")
	}
	if len(data.Collectibles) == 0 {
		return GameplayData{}, fmt.Errorf("collectibles config is empty")
	}
	if err := validateGameplayData(data); err != nil {
		return GameplayData{}, err
	}

	return data, nil
}

func validateGameplayData(data GameplayData) error {
	if data.Rules.Players.Min < 1 || data.Rules.Players.Max < data.Rules.Players.Min || data.Rules.Players.InitialGold < 0 {
		return fmt.Errorf("invalid players config")
	}
	if data.Rules.Rounds.Count < 1 || data.Rules.Rounds.TimeLimitSeconds < 1 {
		return fmt.Errorf("invalid rounds config")
	}
	if data.Rules.Bidding.MinBid < 1 {
		return fmt.Errorf("invalid bidding config")
	}
	if data.Rules.Tiebreaker.MaxRounds < 0 {
		return fmt.Errorf("invalid tiebreaker config")
	}

	for _, venue := range data.Venues {
		if venue.ID == "" || venue.Name == "" || venue.RoundEntryFee < 0 {
			return fmt.Errorf("invalid venue config: %s", venue.ID)
		}
		if venue.CollectibleCountRange.Min < 1 || venue.CollectibleCountRange.Max < venue.CollectibleCountRange.Min {
			return fmt.Errorf("invalid venue collectible count range: %s", venue.ID)
		}
	}

	seenCollectibleIDs := make(map[string]struct{}, len(data.Collectibles))
	for _, collectible := range data.Collectibles {
		if collectible.ID == "" || collectible.Name == "" || collectible.Type == "" || collectible.Tier == "" {
			return fmt.Errorf("invalid collectible config: %s", collectible.ID)
		}
		if _, exists := seenCollectibleIDs[collectible.ID]; exists {
			return fmt.Errorf("duplicate collectible id: %s", collectible.ID)
		}
		seenCollectibleIDs[collectible.ID] = struct{}{}
		if collectible.TrueValue < 1 || collectible.SellValue < 0 {
			return fmt.Errorf("invalid collectible values: %s", collectible.ID)
		}
		if len(collectible.Size) != 2 || collectible.Size[0] < 1 || collectible.Size[1] < 1 {
			return fmt.Errorf("invalid collectible size: %s", collectible.ID)
		}
	}

	return nil
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	return nil
}
