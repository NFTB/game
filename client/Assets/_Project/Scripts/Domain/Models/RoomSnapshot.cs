using System;
using System.Collections.Generic;

namespace BidKing.Client.Domain.Models
{
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
