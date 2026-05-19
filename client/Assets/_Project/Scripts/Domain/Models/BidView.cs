using System;

namespace BidKing.Client.Domain.Models
{
    [Serializable]
    public sealed class BidView
    {
        public string playerId;
        public bool hasBid;
    }
}
