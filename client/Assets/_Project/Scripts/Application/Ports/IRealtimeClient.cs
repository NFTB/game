using System.Threading;
using System.Threading.Tasks;

namespace BidKing.Client.Application.Ports
{
    public readonly struct RealtimeMessage
    {
        public RealtimeMessage(string type, string requestId, string payloadJson)
        {
            Type = type;
            RequestId = requestId;
            PayloadJson = payloadJson;
        }

        public string Type { get; }
        public string RequestId { get; }
        public string PayloadJson { get; }
    }

    public interface IRealtimeClient : System.IDisposable
    {
        event System.Action<RealtimeMessage> MessageReceived;
        event System.Action<System.Exception> ConnectionError;

        bool IsConnected { get; }
        Task ConnectAsync(CancellationToken cancellationToken = default);
        Task SendAsync(string messageType, string payloadJson, CancellationToken cancellationToken = default);
        Task SendAsync(string messageType, string payloadJson, string requestId, CancellationToken cancellationToken = default);
        Task DisconnectAsync(CancellationToken cancellationToken = default);
    }
}
