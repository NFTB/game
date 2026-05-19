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

	return data, nil
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
