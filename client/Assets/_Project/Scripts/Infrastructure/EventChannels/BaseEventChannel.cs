using System;
using System.Collections.Generic;
using UnityEngine;

namespace BidKing.Client.Infrastructure.EventChannels
{
    public abstract class BaseEventChannel<T> : ScriptableObject
    {
        private readonly List<Action<T>> listeners = new();

        public void Register(Action<T> handler)
        {
            if (!listeners.Contains(handler))
            {
                listeners.Add(handler);
            }
        }

        public void Unregister(Action<T> handler)
        {
            listeners.Remove(handler);
        }

        public void Raise(T payload)
        {
            for (int i = listeners.Count - 1; i >= 0; i--)
            {
                listeners[i]?.Invoke(payload);
            }
        }
    }
}
