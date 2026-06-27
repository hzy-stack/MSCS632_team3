package chat

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/labstack/echo/v4"
)

// Handler exposes the chat server's HTTP and WebSocket endpoints through
// Echo.  All business logic is delegated to the Hub.
type Handler struct {
	hub *Hub
}

// NewHandler creates a Handler wired to the given Hub instance.
func NewHandler(hub *Hub) *Handler {
	handler := &Handler{
		hub: hub,
	}

	return handler
}

// HandleWebSocket upgrades an HTTP GET request to a WebSocket connection.
//
// Flow:
//  1. Extract and validate the "username" query parameter
//  2. Accept the WebSocket upgrade
//  3. Register the client with the hub (may fail for duplicates)
//  4. On success: start WritePump in a goroutine, block on ReadPump
//  5. On read error or disconnect: close the Send channel, unregister
func (h *Handler) HandleWebSocket(c echo.Context) error {
	username := c.QueryParam("username")

	if username == "" {
		return c.String(http.StatusBadRequest, "username is required")
	}

	conn, err := websocket.Accept(c.Response(), c.Request(), nil)

	if err != nil {
		return err
	}

	client := NewClient(username, conn)

	result := make(chan error, 1)

	regCmd := registerCommand{
		client: client,
		result: result,
	}

	h.hub.register <- regCmd

	err = <-result

	if err != nil {
		errorResp := ServerResponse{
			Type:  MessageTypeError,
			Error: err.Error(),
		}

		writeErr := wsjson.Write(context.Background(), conn, errorResp)

		if writeErr != nil {
			conn.Close(websocket.StatusNormalClosure, err.Error())
			return nil
		}

		conn.Close(websocket.StatusNormalClosure, err.Error())
		return nil
	}

	ctx := context.Background()

	go client.WritePump(ctx)

	client.ReadPump(ctx, h.hub)

	close(client.Send)

	unregCmd := unregisterCommand{
		username: username,
	}

	h.hub.unregister <- unregCmd

	conn.Close(websocket.StatusNormalClosure, "disconnected")

	return nil
}

// HandleLogout handles POST /logout.  It accepts the username via query
// parameter (?username=alice) or JSON body ({"username":"alice"}).  On
// success the user is removed from active sessions and their WebSocket
// is closed, but all messages are preserved until the server restarts.
func (h *Handler) HandleLogout(c echo.Context) error {
	username := strings.TrimSpace(c.QueryParam("username"))

	if username == "" {
		var body struct {
			Username string `json:"username"`
		}
		if err := c.Bind(&body); err == nil && body.Username != "" {
			username = strings.TrimSpace(body.Username)
		}
	}

	if username == "" {
		log.Printf("logout attempt with empty username")
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "username is required",
		})
	}

	err := h.hub.Logout(username)
	if err != nil {
		log.Printf("logout error for %s: %v", username, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	log.Printf("user %s logged out via HTTP", username)
	return c.JSON(http.StatusOK, map[string]string{
		"status":   "logged out",
		"username": username,
	})
}
