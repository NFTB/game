using BidKing.Client.Networking;

namespace BidKing.Client.Game
{
    public sealed class AuctionSession
    {
        private readonly RealtimeClient realtimeClient;

        public AuctionSession(string serverUrl)
        {
            realtimeClient = new RealtimeClient(serverUrl);
        }

        public RealtimeClient RealtimeClient => realtimeClient;
    }
}
