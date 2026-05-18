using System;
using System.IO;
using System.Net.WebSockets;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

namespace BidKing.Client.Networking
{
    public sealed class RealtimeClient : IDisposable
    {
        private readonly string serverUrl;
        private readonly byte[] receiveBuffer = new byte[8192];
        private ClientWebSocket webSocket;
        private CancellationTokenSource receiveCancellation;
        private Task receiveTask;

        public RealtimeClient(string serverUrl)
        {
            this.serverUrl = serverUrl;
        }

        public event Action<RealtimeMessage> MessageReceived;
        public event Action<Exception> ConnectionError;

        public bool IsConnected { get; private set; }

        public async Task ConnectAsync(CancellationToken cancellationToken = default)
        {
            if (IsConnected)
            {
                return;
            }

            webSocket = new ClientWebSocket();
            receiveCancellation = CancellationTokenSource.CreateLinkedTokenSource(cancellationToken);
            await webSocket.ConnectAsync(new Uri(serverUrl), cancellationToken);
            IsConnected = true;
            receiveTask = ReceiveLoopAsync(receiveCancellation.Token);
        }

        public Task SendAsync(string messageType, string payloadJson, CancellationToken cancellationToken = default)
        {
            return SendAsync(messageType, payloadJson, Guid.NewGuid().ToString("N"), cancellationToken);
        }

        public async Task SendAsync(string messageType, string payloadJson, string requestId, CancellationToken cancellationToken = default)
        {
            if (!IsConnected)
            {
                throw new InvalidOperationException("Realtime client is not connected.");
            }

            if (string.IsNullOrWhiteSpace(payloadJson))
            {
                payloadJson = "{}";
            }

            string envelopeJson = BuildEnvelopeJson(messageType, payloadJson, requestId);
            byte[] bytes = Encoding.UTF8.GetBytes(envelopeJson);
            await webSocket.SendAsync(new ArraySegment<byte>(bytes), WebSocketMessageType.Text, true, cancellationToken);
        }

        public async Task DisconnectAsync(CancellationToken cancellationToken = default)
        {
            if (!IsConnected || webSocket == null)
            {
                return;
            }

            receiveCancellation?.Cancel();
            if (webSocket.State == WebSocketState.Open)
            {
                await webSocket.CloseAsync(WebSocketCloseStatus.NormalClosure, "client disconnect", cancellationToken);
            }

            IsConnected = false;
        }

        public void Dispose()
        {
            receiveCancellation?.Cancel();
            webSocket?.Dispose();
            receiveCancellation?.Dispose();
        }

        private async Task ReceiveLoopAsync(CancellationToken cancellationToken)
        {
            try
            {
                while (!cancellationToken.IsCancellationRequested && webSocket.State == WebSocketState.Open)
                {
                    using (MemoryStream messageStream = new MemoryStream())
                    {
                        WebSocketReceiveResult result;
                        do
                        {
                            result = await webSocket.ReceiveAsync(new ArraySegment<byte>(receiveBuffer), cancellationToken);
                            if (result.MessageType == WebSocketMessageType.Close)
                            {
                                IsConnected = false;
                                return;
                            }

                            messageStream.Write(receiveBuffer, 0, result.Count);
                        }
                        while (!result.EndOfMessage);

                        string json = Encoding.UTF8.GetString(messageStream.ToArray());
                        if (RealtimeMessageParser.TryParse(json, out RealtimeMessage message))
                        {
                            MessageReceived?.Invoke(message);
                        }
                    }
                }
            }
            catch (OperationCanceledException)
            {
            }
            catch (Exception ex)
            {
                IsConnected = false;
                ConnectionError?.Invoke(ex);
            }
        }

        private static string BuildEnvelopeJson(string messageType, string payloadJson, string requestId)
        {
            return "{\"type\":\"" + EscapeJsonString(messageType) + "\",\"requestId\":\"" + EscapeJsonString(requestId) + "\",\"payload\":" + payloadJson + "}";
        }

        private static string EscapeJsonString(string value)
        {
            if (string.IsNullOrEmpty(value))
            {
                return string.Empty;
            }

            return value.Replace("\\", "\\\\").Replace("\"", "\\\"");
        }

        public override string ToString()
        {
            return serverUrl;
        }
    }
}
