using System;
using System.Threading;
using System.Threading.Tasks;

namespace BidKing.Client.Networking
{
    public sealed class RealtimeClient
    {
        private readonly string serverUrl;

        public RealtimeClient(string serverUrl)
        {
            this.serverUrl = serverUrl;
        }

        public bool IsConnected { get; private set; }

        public Task ConnectAsync(CancellationToken cancellationToken = default)
        {
            // WebSocket transport will be implemented after the server message contract is finalized.
            IsConnected = true;
            return Task.CompletedTask;
        }

        public Task SendAsync(string messageType, string payloadJson, CancellationToken cancellationToken = default)
        {
            if (!IsConnected)
            {
                throw new InvalidOperationException("Realtime client is not connected.");
            }

            return Task.CompletedTask;
        }

        public override string ToString()
        {
            return serverUrl;
        }
    }
}
