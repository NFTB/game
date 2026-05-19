using BidKing.Client.Application.Ports;
using BidKing.Client.Application.Sessions;
using BidKing.Client.Infrastructure.Networking;
using UnityEngine;

namespace BidKing.Client.App
{
    public sealed class AppBootstrap : MonoBehaviour
    {
        [SerializeField] private string serverUrl = "ws://localhost:8080/ws";

        private AuctionSession session;

        private void Awake()
        {
            DontDestroyOnLoad(gameObject);

            IRealtimeClient realtimeClient = new RealtimeClient(serverUrl);
            session = new AuctionSession(realtimeClient);

            Debug.Log($"BidKing client booted. Server: {serverUrl}");
        }

        public AuctionSession Session => session;

        private void OnDestroy()
        {
            session?.Dispose();
        }

        private void Update()
        {
            session?.Tick();
        }
    }
}
