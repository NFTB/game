using BidKing.Client.Application.Ports;
using BidKing.Client.Domain.Models;
using System;
using System.Collections.Generic;

namespace BidKing.Client.Application.Sessions
{
    public sealed class AuctionSession : IDisposable
    {
        private readonly IRealtimeClient realtimeClient;
        private readonly Queue<RealtimeMessage> pendingMessages = new();
        private readonly object messageLock = new();

        public AuctionSession(IRealtimeClient realtimeClient)
        {
            this.realtimeClient = realtimeClient;
            this.realtimeClient.MessageReceived += EnqueueMessage;
        }

        public event Action<RealtimeMessage> MessageReceived;
        public event Action<RoomSnapshot> SnapshotUpdated;

        public IRealtimeClient RealtimeClient => realtimeClient;
        public RoomSnapshot CurrentSnapshot { get; private set; }

        public void Tick()
        {
            while (TryDequeueMessage(out RealtimeMessage message))
            {
                HandleMessageReceived(message);
            }
        }

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

        public System.Threading.Tasks.Task LeaveRoomAsync(System.Threading.CancellationToken cancellationToken = default)
        {
            return realtimeClient.SendAsync("room.leave", "{}", cancellationToken);
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
            realtimeClient.MessageReceived -= EnqueueMessage;
            realtimeClient.Dispose();
        }

        private void EnqueueMessage(RealtimeMessage message)
        {
            lock (messageLock)
            {
                pendingMessages.Enqueue(message);
            }
        }

        private bool TryDequeueMessage(out RealtimeMessage message)
        {
            lock (messageLock)
            {
                if (pendingMessages.Count == 0)
                {
                    message = default;
                    return false;
                }

                message = pendingMessages.Dequeue();
                return true;
            }
        }

        private void HandleMessageReceived(RealtimeMessage message)
        {
            MessageReceived?.Invoke(message);
            if (message.Type != "room.snapshot")
            {
                return;
            }

            RoomSnapshot snapshot = UnityEngine.JsonUtility.FromJson<RoomSnapshot>(message.PayloadJson);
            CurrentSnapshot = snapshot;
            SnapshotUpdated?.Invoke(snapshot);
        }

        private static string EscapeJson(string value)
        {
            return string.IsNullOrEmpty(value) ? string.Empty : value.Replace("\\", "\\\\").Replace("\"", "\\\"");
        }
    }
}
