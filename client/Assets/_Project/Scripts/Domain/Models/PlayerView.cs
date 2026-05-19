using System;

namespace BidKing.Client.Domain.Models
{
    [Serializable]
    public sealed class PlayerView
    {
        public string playerId;
        public string displayName;
        public int coins;
        public bool ready;
        public System.Collections.Generic.List<string> wonLotIds = new();
        public int collectionValue;

        public string PlayerId => playerId;
        public string DisplayName => displayName;
        public int Coins => coins;
        public bool Ready => ready;
        public int CollectionValue => collectionValue;
    }
}
