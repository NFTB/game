using System;
using System.Collections.Generic;
using UnityEngine;

namespace BidKing.Client.Config
{
    [CreateAssetMenu(menuName = "bidking/Item Catalog", fileName = "ItemCatalog")]
    public sealed class ItemCatalog : ScriptableObject
    {
        [SerializeField] private List<ItemDefinition> items = new();

        public IReadOnlyList<ItemDefinition> Items => items;
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
}
