using BidKing.Client.Networking;
using System;
using UnityEngine;

namespace BidKing.Client.Game
{
    public sealed class AuctionSession : IDisposable
    {
        private readonly RealtimeClient realtimeClient;

        public AuctionSession(string serverUrl)
        {
            realtimeClient = new RealtimeClient(serverUrl);
            realtimeClient.MessageReceived += HandleMessageReceived;
        }

        public event Action<RealtimeMessage> MessageReceived;
        public event Action<RoomSnapshot> SnapshotUpdated;

        public RealtimeClient RealtimeClient => realtimeClient;
        public RoomSnapshot CurrentSnapshot { get; private set; }

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
            realtimeClient.MessageReceived -= HandleMessageReceived;
            realtimeClient.Dispose();
        }

        private void HandleMessageReceived(RealtimeMessage message)
        {
            MessageReceived?.Invoke(message);
            if (message.Type != "room.snapshot")
            {
                return;
            }

            RoomSnapshot snapshot = JsonUtility.FromJson<RoomSnapshot>(message.PayloadJson);
            CurrentSnapshot = snapshot;
            SnapshotUpdated?.Invoke(snapshot);
        }

        private static string EscapeJson(string value)
        {
            return string.IsNullOrEmpty(value) ? string.Empty : value.Replace("\\", "\\\\").Replace("\"", "\\\"");
        }
    }
}
