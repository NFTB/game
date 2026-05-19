using System.Collections.Generic;
using BidKing.Client.Domain.Models;
using UnityEngine;

namespace BidKing.Client.Infrastructure.Config
{
    [CreateAssetMenu(menuName = "bidking/Item Catalog", fileName = "ItemCatalog")]
    public sealed class ItemCatalog : ScriptableObject
    {
        [SerializeField] private List<ItemDefinition> items = new();

        public IReadOnlyList<ItemDefinition> Items => items;
    }
}
