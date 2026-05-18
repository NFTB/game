package realtime

import (
	"encoding/json"
	"strings"
	"testing"

	"bidking/server/internal/game"
)

func TestRoomSnapshotJSONUsesProtocolFieldNames(t *testing.T) {
	payload := game.RoomSnapshot{
		RoomID:           "room_1",
		Phase:            game.RoomPhaseAuction,
		RoundNumber:      1,
		RoundTimeSeconds: 30,
		Players: []game.PlayerSnapshot{
			{ID: "player_1", DisplayName: "Alice", Coins: 1000, Ready: true, WonLotIDs: []string{"lot_1"}},
		},
		CurrentLot: &game.Lot{
			ID:          "lot_1",
			DisplayName: "测试仓库",
			Items: []game.Item{
				{ID: "item_1", DisplayName: "测试藏品", Rarity: "R", Type: "瓷器", EstimatedMinValue: 10, EstimatedMaxValue: 20, SellValue: 20},
			},
		},
		Bids: []game.BidSnapshot{
			{PlayerID: "player_1", HasBid: true},
		},
	}

	data, err := json.Marshal(outbound("req_1", "room.snapshot", payload))
	if err != nil {
		t.Fatalf("marshal snapshot envelope: %v", err)
	}
	jsonText := string(data)

	for _, want := range []string{
		`"type":"room.snapshot"`,
		`"requestId":"req_1"`,
		`"roomId":"room_1"`,
		`"roundNumber":1`,
		`"roundTimeSeconds":30`,
		`"playerId":"player_1"`,
		`"wonLotIds":["lot_1"]`,
		`"lotId":"lot_1"`,
		`"itemId":"item_1"`,
		`"estimatedMinValue":10`,
		`"hasBid":true`,
	} {
		if !strings.Contains(jsonText, want) {
			t.Fatalf("json %s does not contain %s", jsonText, want)
		}
	}

	for _, forbidden := range []string{`"RoomID"`, `"PlayerID"`, `"CurrentLot"`} {
		if strings.Contains(jsonText, forbidden) {
			t.Fatalf("json %s contains Go field name %s", jsonText, forbidden)
		}
	}
}
