// Package chat implements a chat server hub that uses the actor pattern:
// a single background goroutine owns all mutable state (users, clients,
// and messages) and processes commands delivered through channels.
// This design eliminates the need for mutexes entirely.
package chat

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// --- Command types for the hub actor ---

// registerCommand is sent when a new WebSocket client attempts to register.
// The result channel carries nil on success or an error (e.g. duplicate
// username).
type registerCommand struct {
	client *Client
	result chan error
}

// unregisterCommand is sent when a WebSocket connection closes to mark
// the user as disconnected (but preserves their session and messages).
type unregisterCommand struct {
	username string
}

// logoutCommand is sent when a user explicitly logs out via HTTP.
// Unlike unregister, it fully removes the client and user session entries
// from the hub maps.  Messages are preserved until server restart.
type logoutCommand struct {
	username string
	result   chan error
}

// inboundCommand wraps a ClientRequest received from a WebSocket client.
// The hub routes it to the appropriate handler based on the request Type.
type inboundCommand struct {
	request ClientRequest
	client  *Client
}

// Hub is the central actor that owns all chat state and serialises all
// mutations through a single goroutine (run).  External callers interact
// with the hub exclusively via channels — there are no mutexes.
//
// State ownership:
//   users     — active and disconnected user sessions (keyed by username)
//   clients   — currently connected WebSocket clients (keyed by username)
//   messagesByConversation — message history keyed by canonical
//                              conversation key (see conversationKey)
type Hub struct {
	users                  map[string]*UserSession
	clients                map[string]*Client
	messagesByConversation map[string][]Message
	register               chan registerCommand
	unregister             chan unregisterCommand
	logout                 chan logoutCommand
	inbound                chan inboundCommand
}

// NewHub creates a Hub, initialises its internal maps and channels, and
// immediately starts the background actor goroutine.
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

// run is the single goroutine that owns all hub state.  It select-multiplexes
// across the command channels, guaranteeing that all state mutations are
// sequential and race-free.
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

// handleRegister validates the username, rejects duplicates, and adds the
// client to both the clients and users maps.  Errors are sent on
// cmd.result; nil means success.
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

// handleLogout actively logs out a user: it deletes the client entry,
// closes the underlying WebSocket connection (if present), and removes the
// user session entirely.  Messages in messagesByConversation are NOT
// touched so they survive until the next server restart.
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

// Logout is the public API for logging out a user.  It sends a logoutCommand
// to the hub actor and blocks until the operation completes.
func (h *Hub) Logout(username string) error {
	result := make(chan error, 1)
	h.logout <- logoutCommand{username: username, result: result}
	return <-result
}

// handleUnregister handles a passive WebSocket disconnect.  The client is
// removed from the clients map and the user session is marked disconnected
// but NOT deleted.  Messages are preserved.
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

// handleInboundRequest dispatches an incoming WebSocket message to the
// appropriate handler based on the request type field.
func (h *Hub) handleInboundRequest(cmd inboundCommand) {
	requestType := cmd.request.Type

	if requestType == MessageTypeChat {
		h.handleChat(cmd)
	} else if requestType == MessageTypeHistory {
		h.handleHistory(cmd)
	} else if requestType == MessageTypeSearch {
		h.handleSearch(cmd)
	} else if requestType == MessageTypeListUsers {
		h.handleListUsers(cmd)
	} else {
		errResp := ServerResponse{
			Type:  MessageTypeError,
			Error: "unknown message type: " + requestType,
		}
		sendToClient(cmd.client, errResp)
	}
}

// handleChat processes a chat message: validates recipient and text,
// creates a Message with a UUID, stores it under the canonical conversation
// key, and delivers it to both sender (echo) and recipient (if online).
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

// handleHistory returns all messages in the conversation between the
// requesting user and the specified recipient.  A copy of the slice is
// returned to avoid data races.
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

// handleSearch performs a case-insensitive full-text search across all
// conversations the requesting user participates in.  It matches against
// message text, sender, and recipient fields.
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

// handleListUsers returns a list of all registered users (both connected
// and disconnected) along with their connection status.
func (h *Hub) handleListUsers(cmd inboundCommand) {
	users := make([]UserSession, 0, len(h.users))

	for _, s := range h.users {
		users = append(users, *s)
	}

	resp := ServerResponse{
		Type:  MessageTypeListUsers,
		Users: users,
	}

	sendToClient(cmd.client, resp)
}

// sendToClient is a non-blocking send on the client's buffered Send channel.
// If the channel is full the response is silently dropped (the client is
// too slow).
func sendToClient(c *Client, resp ServerResponse) {
	select {
	case c.Send <- resp:
	default:
	}
}
