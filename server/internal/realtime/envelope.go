package realtime

import "encoding/json"

type Envelope struct {
	Type      string          `json:"type"`
	RequestID string          `json:"requestId,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	SentAt    int64           `json:"sentAt,omitempty"`
}

type OutboundEnvelope struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId,omitempty"`
	Payload   any    `json:"payload"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decodePayload(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}

	return json.Unmarshal(raw, target)
}

func outbound(requestID string, messageType string, payload any) OutboundEnvelope {
	return OutboundEnvelope{
		Type:      messageType,
		RequestID: requestID,
		Payload:   payload,
	}
}

func outboundError(requestID string, code string, message string) OutboundEnvelope {
	return outbound(requestID, "error", ErrorPayload{
		Code:    code,
		Message: message,
	})
}
