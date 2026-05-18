namespace BidKing.Client.Networking
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
}
