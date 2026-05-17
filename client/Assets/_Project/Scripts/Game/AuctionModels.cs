using System;
using System.Collections.Generic;

namespace BidKing.Client.Game
{
    [Serializable]
    public sealed class PlayerView
    {
        public string PlayerId;
        public string DisplayName;
        public int Coins;
        public bool Ready;
    }

    [Serializable]
    public sealed class ItemView
    {
        public string ItemId;
        public string DisplayName;
        public string Rarity;
        public int EstimatedMinValue;
        public int EstimatedMaxValue;
    }

    [Serializable]
    public sealed class RoomSnapshot
    {
        public string RoomId;
        public string Phase;
        public List<PlayerView> Players = new();
        public ItemView CurrentItem;
        public int SecondsRemaining;
    }
}
