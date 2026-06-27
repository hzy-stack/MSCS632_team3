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

type Handler struct {
	hub *Hub
}

func NewHandler(hub *Hub) *Handler {
	handler := &Handler{
		hub: hub,
	}

	return handler
}

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
