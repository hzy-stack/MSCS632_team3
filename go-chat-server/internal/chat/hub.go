package chat

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type registerCommand struct {
	client *Client
	result chan error
}

type unregisterCommand struct {
	username string
}

type logoutCommand struct {
	username string
	result   chan error
}

type inboundCommand struct {
	request ClientRequest
	client  *Client
}

type Hub struct {
	users                  map[string]*UserSession
	clients                map[string]*Client
	messagesByConversation map[string][]Message
	register               chan registerCommand
	unregister             chan unregisterCommand
	logout                 chan logoutCommand
	inbound                chan inboundCommand
}

func NewHub() *Hub {
	hub := &Hub{
		users:                  make(map[string]*UserSession),
		clients:                make(map[string]*Client),
		messagesByConversation: make(map[string][]Message),
		register:               make(chan registerCommand),
		unregister:             make(chan unregisterCommand),
		logout:                 make(chan logoutCommand),
		inbound:                make(chan inboundCommand),
	}

	go hub.run()

	return hub
}

func (h *Hub) run() {
	for {
		select {
		case cmd := <-h.register:
			h.handleRegister(cmd)

		case cmd := <-h.unregister:
			h.handleUnregister(cmd)

		case cmd := <-h.logout:
			h.handleLogout(cmd)

		case cmd := <-h.inbound:
			h.handleInboundRequest(cmd)
		}
	}
}

func (h *Hub) handleRegister(cmd registerCommand) {
	username := cmd.client.Username

	if username == "" {
		cmd.result <- fmt.Errorf("username is required")
		return
	}

	_, exists := h.clients[username]

	if exists {
		cmd.result <- fmt.Errorf("username %q is already in use", username)
		return
	}

	h.clients[username] = cmd.client

	h.users[username] = &UserSession{
		Username:    username,
		Connected:   true,
		ConnectedAt: time.Now(),
	}

	log.Printf("user registered: %s", username)
	cmd.result <- nil
}

func (h *Hub) handleLogout(cmd logoutCommand) {
	username := cmd.username

	client, exists := h.clients[username]
	if exists {
		delete(h.clients, username)
		if client.Conn != nil {
			client.Conn.Close(websocket.StatusNormalClosure, "logged out")
		}
	}

	delete(h.users, username)

	log.Printf("user logged out: %s", username)
	cmd.result <- nil
}

func (h *Hub) Logout(username string) error {
	result := make(chan error, 1)
	h.logout <- logoutCommand{username: username, result: result}
	return <-result
}

func (h *Hub) handleUnregister(cmd unregisterCommand) {
	username := cmd.username

	_, exists := h.clients[username]

	if exists {
		delete(h.clients, username)
	}

	session, exists := h.users[username]

	if exists {
		session.Connected = false
	}

	log.Printf("user disconnected: %s", username)
}

func (h *Hub) handleInboundRequest(cmd inboundCommand) {
	requestType := cmd.request.Type

	if requestType == MessageTypeChat {
		h.handleChat(cmd)
	} else if requestType == MessageTypeHistory {
		h.handleHistory(cmd)
	} else if requestType == MessageTypeSearch {
		h.handleSearch(cmd)
	} else {
		errResp := ServerResponse{
			Type:  MessageTypeError,
			Error: "unknown message type: " + requestType,
		}
		sendToClient(cmd.client, errResp)
	}
}

func (h *Hub) handleChat(cmd inboundCommand) {
	req := cmd.request
	client := cmd.client

	if req.Recipient == "" {
		errResp := ServerResponse{
			Type:  MessageTypeError,
			Error: "recipient is required",
		}
		sendToClient(client, errResp)
		return
	}

	if req.Text == "" {
		errResp := ServerResponse{
			Type:  MessageTypeError,
			Error: "message text is required",
		}
		sendToClient(client, errResp)
		return
	}

	msg := Message{
		MessageID: uuid.New().String(),
		Sender:    req.Sender,
		Recipient: req.Recipient,
		Text:      req.Text,
		Timestamp: time.Now(),
	}

	key := conversationKey(req.Sender, req.Recipient)

	h.messagesByConversation[key] = append(h.messagesByConversation[key], msg)

	log.Printf("chat message: %s -> %s: %s", req.Sender, req.Recipient, req.Text)

	chatResp := ServerResponse{
		Type:    MessageTypeChat,
		Message: &msg,
	}

	sendToClient(client, chatResp)

	recipientClient, isOnline := h.clients[req.Recipient]

	if isOnline {
		sendToClient(recipientClient, chatResp)
	}
}

func (h *Hub) handleHistory(cmd inboundCommand) {
	req := cmd.request
	client := cmd.client

	if req.Recipient == "" {
		errResp := ServerResponse{
			Type:  MessageTypeError,
			Error: "recipient is required for history request",
		}
		sendToClient(client, errResp)
		return
	}

	key := conversationKey(req.Sender, req.Recipient)

	msgs, exists := h.messagesByConversation[key]

	var messages []Message

	if exists {
		messages = make([]Message, len(msgs))
		copy(messages, msgs)
	} else {
		messages = make([]Message, 0)
	}

	historyResp := ServerResponse{
		Type:     MessageTypeHistory,
		Messages: messages,
	}

	sendToClient(client, historyResp)
}

func (h *Hub) handleSearch(cmd inboundCommand) {
	req := cmd.request
	client := cmd.client

	if req.Query == "" {
		errResp := ServerResponse{
			Type:  MessageTypeError,
			Error: "query is required for search",
		}
		sendToClient(client, errResp)
		return
	}

	query := strings.ToLower(req.Query)
	results := make([]Message, 0)

	for key, msgs := range h.messagesByConversation {
		if !strings.Contains(key, req.Sender) {
			continue
		}

		for _, msg := range msgs {
			if strings.Contains(strings.ToLower(msg.Text), query) {
				results = append(results, msg)
				continue
			}

			if strings.Contains(strings.ToLower(msg.Sender), query) {
				results = append(results, msg)
				continue
			}

			if strings.Contains(strings.ToLower(msg.Recipient), query) {
				results = append(results, msg)
			}
		}
	}

	searchResp := ServerResponse{
		Type:     MessageTypeSearch,
		Messages: results,
	}

	sendToClient(client, searchResp)
}

func sendToClient(c *Client, resp ServerResponse) {
	select {
	case c.Send <- resp:
	default:
	}
}
