using System;
using System.Collections.Generic;

namespace BidKing.Client.Game
{
    [Serializable]
    public sealed class PlayerView
    {
        public string playerId;
        public string displayName;
        public int coins;
        public bool ready;
        public List<string> wonLotIds = new();
        public int collectionValue;

        public string PlayerId => playerId;
        public string DisplayName => displayName;
        public int Coins => coins;
        public bool Ready => ready;
    }

    [Serializable]
    public sealed class ItemView
    {
        public string itemId;
        public string displayName;
        public string rarity;
        public string type;
        public int estimatedMinValue;
        public int estimatedMaxValue;
        public int trueValue;
        public int sellValue;

        public string ItemId => itemId;
        public string DisplayName => displayName;
        public string Rarity => rarity;
    }

    [Serializable]
    public sealed class LotView
    {
        public string lotId;
        public string displayName;
        public int trueValue;
        public List<ItemView> items = new();
    }

    [Serializable]
    public sealed class BidView
    {
        public string playerId;
        public bool hasBid;
    }

    [Serializable]
    public sealed class RoomSnapshot
    {
        public string roomId;
        public string phase;
        public int roundNumber;
        public int roundTimeSeconds;
        public List<PlayerView> players = new();
        public LotView currentLot;
        public List<BidView> bids = new();
        public List<string> rebidPlayerIds = new();

        public string RoomId => roomId;
        public string Phase => phase;
        public IReadOnlyList<PlayerView> Players => players;
        public int SecondsRemaining => roundTimeSeconds;
    }
}
