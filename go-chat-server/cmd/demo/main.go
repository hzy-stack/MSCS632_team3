// Command demo seeds the chat server with realistic demo data.
//
// It connects users via WebSocket, sends conversations, then disconnects.
// After running, the server contains full message history searchable
// through the normal WebSocket history / search endpoints.
//
// Usage:
//
//	make demo          # requires the server to be running on :8080
//	go run ./cmd/demo/ # same
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"go-chat-server/internal/chat"
)

const serverURL = "ws://localhost:8080/ws"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	log.SetFlags(0)

	log.Println("=== Chat Server Demo Seed ===")
	log.Println()

	seedLive(ctx)
	seedOffline(ctx)
	seedMore(ctx)

	log.Println()
	log.Println("=== Demo data seeded successfully ===")
	log.Println("Connect via ws://localhost:8080/ws?username=alice")
	log.Println("  then send {\"type\":\"history\",\"recipient\":\"bob\"}")
}

// --- helpers ---

func dial(ctx context.Context, username string) (*websocket.Conn, error) {
	url := fmt.Sprintf("%s?username=%s", serverURL, username)
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", username, err)
	}
	return conn, nil
}

// writeChat sends a chat message and drains the echo.
// Returns the ServerResponse (echo or error).
func writeChat(ctx context.Context, conn *websocket.Conn, recipient, text string) {
	if err := wsjson.Write(ctx, conn, chat.ClientRequest{
		Type:      chat.MessageTypeChat,
		Recipient: recipient,
		Text:      text,
	}); err != nil {
		log.Printf("  write error: %v", err)
		return
	}
	var resp chat.ServerResponse
	if err := wsjson.Read(ctx, conn, &resp); err != nil {
		return
	}
	if resp.Type == chat.MessageTypeError {
		log.Printf("  server error: %s", resp.Error)
	}
}

// drainOne reads and discards one message from conn.
// Returns the message if it's a chat, or nil on error / non-chat.
func drainOne(ctx context.Context, conn *websocket.Conn) *chat.ServerResponse {
	var resp chat.ServerResponse
	if err := wsjson.Read(ctx, conn, &resp); err != nil {
		return nil
	}
	return &resp
}

// --- live conversation (both users online, messages delivered) ---

func seedLive(ctx context.Context) {
	log.Println("--- Live: Alice <-> Bob (both online) ---")
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	alice, err := dial(ctx, "alice")
	if err != nil {
		log.Fatalf("connect alice: %v", err)
	}
	log.Println("[alice connected]")

	bob, err := dial(ctx, "bob")
	if err != nil {
		log.Fatalf("connect bob: %v", err)
	}
	log.Println("[bob connected]")

	// Alice sends messages to Bob.  After each send:
	//   1. drain Alice's echo
	//   2. drain Bob's copy (the delivered message)
	aliceMsgs := []string{
		"Hey Bob! How's the Sprint 2 progress?",
		"I pushed the auth middleware - can you review?",
		"Also, the test coverage dropped to 78%, we need more tests.",
	}
	for _, msg := range aliceMsgs {
		log.Printf("  alice -> bob: %q", msg)
		writeChat(ctx, alice, "bob", msg)
		if r := drainOne(ctx, bob); r != nil && r.Message != nil {
			log.Printf("  bob received: %q", r.Message.Text)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Bob responds.  Same pattern: write, drain echo from sender, drain
	// copy from recipient.
	bobMsgs := []string{
		"Sprint 2 is almost done, just finishing the WebSocket handler!",
		"Sure, I'll review the auth PR right after lunch.",
	}
	for _, msg := range bobMsgs {
		log.Printf("  bob -> alice: %q", msg)
		writeChat(ctx, bob, "alice", msg)
		if r := drainOne(ctx, alice); r != nil && r.Message != nil {
			log.Printf("  alice received: %q", r.Message.Text)
		}
		time.Sleep(300 * time.Millisecond)
	}

	alice.Close(websocket.StatusNormalClosure, "demo done")
	bob.Close(websocket.StatusNormalClosure, "demo done")
	log.Println("[alice disconnected]")
	log.Println("[bob disconnected]")
	log.Println()
}

// --- offline messages (recipient not connected, stored server-side) ---

func seedOffline(ctx context.Context) {
	log.Println("--- Offline: Charlie -> Alice (Alice not connected) ---")
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	charlie, err := dial(ctx, "charlie")
	if err != nil {
		log.Fatalf("connect charlie: %v", err)
	}
	log.Println("[charlie connected]")

	messages := []string{
		"Alice, I found a bug in message search - it's case-sensitive for sender names.",
		"I opened issue #42 with a proposed fix.",
		"Also, the README needs updating with the new logout endpoint.",
	}
	for _, msg := range messages {
		log.Printf("  charlie -> alice: %q", msg)
		writeChat(ctx, charlie, "alice", msg)
		time.Sleep(200 * time.Millisecond)
	}

	charlie.Close(websocket.StatusNormalClosure, "demo done")
	log.Println("[charlie disconnected]")
	log.Println()
}

// --- additional messages for search/history depth ---

func seedMore(ctx context.Context) {
	log.Println("--- More: Alice -> Bob (Bob offline) ---")
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	alice, err := dial(ctx, "alice")
	if err != nil {
		log.Fatalf("connect alice: %v", err)
	}
	log.Println("[alice connected]")

	messages := []string{
		"The deployment pipeline is broken again.",
		"We should upgrade the Go version to 1.26.",
		"Meeting at 3pm to discuss the architecture review.",
	}
	for _, msg := range messages {
		log.Printf("  alice -> bob: %q", msg)
		writeChat(ctx, alice, "bob", msg)
		time.Sleep(200 * time.Millisecond)
	}

	alice.Close(websocket.StatusNormalClosure, "demo done")
	log.Println("[alice disconnected]")
	log.Println()
}
