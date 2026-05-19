using System;
using System.Collections.Generic;

namespace BidKing.Client.Domain.Models
{
    [Serializable]
    public sealed class LotView
    {
        public string lotId;
        public string displayName;
        public int trueValue;
        public List<ItemView> items = new();
    }
}
