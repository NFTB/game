using BidKing.Client.Game;
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
            session = new AuctionSession(serverUrl);
            Debug.Log($"BidKing client booted. Server: {serverUrl}");
        }

        private void OnDestroy()
        {
            session?.Dispose();
        }

        public AuctionSession Session => session;
    }
}
