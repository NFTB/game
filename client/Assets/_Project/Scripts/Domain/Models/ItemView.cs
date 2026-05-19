using System;

namespace BidKing.Client.Domain.Models
{
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
}
