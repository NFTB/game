using System;

namespace BidKing.Client.Networking
{
    internal static class RealtimeMessageParser
    {
        public static bool TryParse(string json, out RealtimeMessage message)
        {
            message = default;
            if (string.IsNullOrWhiteSpace(json))
            {
                return false;
            }

            string type = ReadStringProperty(json, "type");
            if (string.IsNullOrEmpty(type))
            {
                return false;
            }

            string requestId = ReadStringProperty(json, "requestId");
            string payloadJson = ReadRawProperty(json, "payload");
            if (string.IsNullOrEmpty(payloadJson))
            {
                payloadJson = "{}";
            }

            message = new RealtimeMessage(type, requestId, payloadJson);
            return true;
        }

        private static string ReadStringProperty(string json, string propertyName)
        {
            int valueStart = FindPropertyValueStart(json, propertyName);
            if (valueStart < 0 || valueStart >= json.Length || json[valueStart] != '"')
            {
                return string.Empty;
            }

            int index = valueStart + 1;
            bool escaped = false;
            while (index < json.Length)
            {
                char c = json[index];
                if (escaped)
                {
                    escaped = false;
                }
                else if (c == '\\')
                {
                    escaped = true;
                }
                else if (c == '"')
                {
                    return json.Substring(valueStart + 1, index - valueStart - 1);
                }

                index++;
            }

            return string.Empty;
        }

        private static string ReadRawProperty(string json, string propertyName)
        {
            int valueStart = FindPropertyValueStart(json, propertyName);
            if (valueStart < 0 || valueStart >= json.Length)
            {
                return string.Empty;
            }

            char first = json[valueStart];
            if (first == '{' || first == '[')
            {
                int valueEnd = FindMatchingJsonEnd(json, valueStart);
                return valueEnd > valueStart ? json.Substring(valueStart, valueEnd - valueStart + 1) : string.Empty;
            }

            int end = valueStart;
            while (end < json.Length && json[end] != ',' && json[end] != '}')
            {
                end++;
            }

            return json.Substring(valueStart, end - valueStart).Trim();
        }

        private static int FindPropertyValueStart(string json, string propertyName)
        {
            string needle = "\"" + propertyName + "\"";
            int propertyIndex = json.IndexOf(needle, StringComparison.Ordinal);
            if (propertyIndex < 0)
            {
                return -1;
            }

            int colonIndex = json.IndexOf(':', propertyIndex + needle.Length);
            if (colonIndex < 0)
            {
                return -1;
            }

            int valueStart = colonIndex + 1;
            while (valueStart < json.Length && char.IsWhiteSpace(json[valueStart]))
            {
                valueStart++;
            }

            return valueStart;
        }

        private static int FindMatchingJsonEnd(string json, int start)
        {
            char open = json[start];
            char close = open == '{' ? '}' : ']';
            int depth = 0;
            bool inString = false;
            bool escaped = false;

            for (int i = start; i < json.Length; i++)
            {
                char c = json[i];
                if (escaped)
                {
                    escaped = false;
                    continue;
                }
                if (c == '\\')
                {
                    escaped = true;
                    continue;
                }
                if (c == '"')
                {
                    inString = !inString;
                    continue;
                }
                if (inString)
                {
                    continue;
                }
                if (c == open)
                {
                    depth++;
                }
                else if (c == close)
                {
                    depth--;
                    if (depth == 0)
                    {
                        return i;
                    }
                }
            }

            return -1;
        }
    }
}
