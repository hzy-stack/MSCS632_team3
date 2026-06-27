package chat

import (
	"context"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)


type Client struct {
	Username string
	Conn *websocket.Conn
	Send chan ServerResponse
}

func NewClient(username string, conn *websocket.Conn) *Client {
	return &Client{
		Username: username,
		Conn: conn,
		Send:     make(chan ServerResponse, 32),
	}
}

func (c *Client) ReadPump(ctx context.Context, hub *Hub) {
	for {
		var request ClientRequest

		err := wsjson.Read(ctx, c.Conn, &request)

		if err != nil {
			return
		}

		request.Sender = c.Username

		cmd := inboundCommand{
			request: request,
			client:  c,
		}

		hub.inbound <- cmd
	}
}

func (c *Client) WritePump(ctx context.Context) {
	for resp := range c.Send {
		err := wsjson.Write(ctx, c.Conn, resp)

		if err != nil {
			return
		}
	}
}
