package chat

import (
	"context"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Client represents a single connected WebSocket client.  Each Client has a
// unique Username, a reference to the underlying WebSocket Conn, and a
// buffered Send channel through which ServerResponse messages are delivered
// asynchronously by the hub goroutine.
type Client struct {
	Username string
	Conn     *websocket.Conn     // underlying WebSocket; may be nil in tests
	Send     chan ServerResponse // buffered channel (capacity 32)
}

// NewClient creates a Client with the given username and WebSocket
// connection.  The Send channel is buffered so the hub can deliver
// responses without blocking on slow consumers.
func NewClient(username string, conn *websocket.Conn) *Client {
	return &Client{
		Username: username,
		Conn:     conn,
		Send:     make(chan ServerResponse, 32),
	}
}

// ReadPump runs in the calling goroutine, blocking on WebSocket reads.
// Each incoming ClientRequest has its Sender forced to the authenticated
// username and is forwarded to the hub's inbound channel for processing.
// The loop exits when the WebSocket connection is closed or encounters an
// error.
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

// WritePump runs in its own goroutine, consuming ServerResponse values from
// the Send channel and writing them over the WebSocket.  The goroutine exits
// when the Send channel is closed (signalling disconnection).
func (c *Client) WritePump(ctx context.Context) {
	for resp := range c.Send {
		err := wsjson.Write(ctx, c.Conn, resp)

		if err != nil {
			return
		}
	}
}
