package chat

import "time"

const (
	MessageTypeChat    = "chat"
	MessageTypeHistory = "history"
	MessageTypeSearch  = "search"
	MessageTypeError   = "error"
)

type UserSession struct {
	Username    string    `json:"username"`
	Connected   bool      `json:"connected"`
	ConnectedAt time.Time `json:"connectedAt"`
}

type Message struct {
	MessageID string    `json:"messageId"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type ClientRequest struct {
	Type      string `json:"type"`
	Sender    string `json:"sender,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Text      string `json:"text,omitempty"`
	Query     string `json:"query,omitempty"`
}

type ServerResponse struct {
	Type     string    `json:"type"`
	Message  *Message  `json:"message,omitempty"`
	Messages []Message `json:"messages,omitempty"`
	Error    string    `json:"error,omitempty"`
}
