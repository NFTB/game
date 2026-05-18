using BidKing.Client.Networking;

namespace BidKing.Client.Game
{
    public sealed class AuctionSession : System.IDisposable
    {
        private readonly RealtimeClient realtimeClient;

        public AuctionSession(string serverUrl)
        {
            realtimeClient = new RealtimeClient(serverUrl);
        }

        public RealtimeClient RealtimeClient => realtimeClient;

        public System.Threading.Tasks.Task ConnectAsync(System.Threading.CancellationToken cancellationToken = default)
        {
            return realtimeClient.ConnectAsync(cancellationToken);
        }

        public System.Threading.Tasks.Task AuthenticateGuestAsync(string displayName, System.Threading.CancellationToken cancellationToken = default)
        {
            return realtimeClient.SendAsync("auth.guest", "{\"displayName\":\"" + EscapeJson(displayName) + "\"}", cancellationToken);
        }

        public System.Threading.Tasks.Task CreateRoomAsync(System.Threading.CancellationToken cancellationToken = default)
        {
            return realtimeClient.SendAsync("room.create", "{}", cancellationToken);
        }

        public System.Threading.Tasks.Task JoinRoomAsync(string roomId, System.Threading.CancellationToken cancellationToken = default)
        {
            return realtimeClient.SendAsync("room.join", "{\"roomId\":\"" + EscapeJson(roomId) + "\"}", cancellationToken);
        }

        public System.Threading.Tasks.Task SetReadyAsync(bool ready, System.Threading.CancellationToken cancellationToken = default)
        {
            return realtimeClient.SendAsync("room.ready", "{\"ready\":" + (ready ? "true" : "false") + "}", cancellationToken);
        }

        public System.Threading.Tasks.Task PlaceBidAsync(int amount, System.Threading.CancellationToken cancellationToken = default)
        {
            return realtimeClient.SendAsync("auction.bid", "{\"amount\":" + amount + "}", cancellationToken);
        }

        public System.Threading.Tasks.Task PassAsync(System.Threading.CancellationToken cancellationToken = default)
        {
            return realtimeClient.SendAsync("auction.pass", "{}", cancellationToken);
        }

        public System.Threading.Tasks.Task NextRoundAsync(System.Threading.CancellationToken cancellationToken = default)
        {
            return realtimeClient.SendAsync("auction.next_round", "{}", cancellationToken);
        }

        public void Dispose()
        {
            realtimeClient.Dispose();
        }

        private static string EscapeJson(string value)
        {
            return string.IsNullOrEmpty(value) ? string.Empty : value.Replace("\\", "\\\\").Replace("\"", "\\\"");
        }
    }
}
