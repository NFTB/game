using System;
using System.Collections.Generic;

namespace BidKing.Client.Domain.Models
{
    [Serializable]
    public sealed class CollectibleDefinition
    {
        public string Id;
        public string DisplayName;
        public string Type;
        public string Rarity;
        public int TrueValue;
        public int Size;
    }

    [Serializable]
    public sealed class ItemDefinition
    {
        public string Id;
        public string DisplayName;
        public string Rarity;
        public int BaseValue;
        public int ValueVariance;
    }

    [Serializable]
    public sealed class CollectibleCatalog
    {
        public List<CollectibleDefinition> Items = new();
    }
}
