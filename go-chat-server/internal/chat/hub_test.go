package chat

import (
	"testing"
	"time"
)

// --- helpers ---

// newTestClient creates a Client usable in hub tests.  Conn is nil because
// the hub actor only touches Conn during logout (and we guard with a nil
// check there).
func newTestClient(username string) *Client {
	return &Client{
		Username: username,
		Send:     make(chan ServerResponse, 32),
	}
}

// mustRegister registers a client with the hub and fails the test on error.
func mustRegister(t *testing.T, h *Hub, username string) *Client {
	t.Helper()
	c := newTestClient(username)
	result := make(chan error, 1)
	h.register <- registerCommand{client: c, result: result}
	if err := <-result; err != nil {
		t.Fatalf("register(%q): %v", username, err)
	}
	return c
}

// mustFailRegister asserts that registering a client with the given username
// produces a non-nil error.
func mustFailRegister(t *testing.T, h *Hub, username string) {
	t.Helper()
	c := newTestClient(username)
	result := make(chan error, 1)
	h.register <- registerCommand{client: c, result: result}
	if err := <-result; err == nil {
		t.Fatalf("register(%q): expected error, got nil", username)
	}
}

// chatCmd builds an inboundCommand that simulates what ReadPump does:
// it sets Sender from the client's Username.
func chatCmd(c *Client, recipient, text string) inboundCommand {
	return inboundCommand{
		request: ClientRequest{Type: MessageTypeChat, Sender: c.Username, Recipient: recipient, Text: text},
		client:  c,
	}
}

// historyCmd builds a history request with Sender populated.
func historyCmd(c *Client, recipient string) inboundCommand {
	return inboundCommand{
		request: ClientRequest{Type: MessageTypeHistory, Sender: c.Username, Recipient: recipient},
		client:  c,
	}
}

// searchCmd builds a search request with Sender populated so that the
// hub only searches conversations the user participates in.
func searchCmd(c *Client, query string) inboundCommand {
	return inboundCommand{
		request: ClientRequest{Type: MessageTypeSearch, Sender: c.Username, Query: query},
		client:  c,
	}
}

// readResp reads a single ServerResponse from the client's Send channel,
// failing the test if no response arrives within 1 second.
func readResp(t *testing.T, c *Client) ServerResponse {
	t.Helper()
	select {
	case r := <-c.Send:
		return r
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for ServerResponse")
		return ServerResponse{}
	}
}

// assertEmpty asserts that no response is pending on the client's Send
// channel.
func assertEmpty(t *testing.T, c *Client) {
	t.Helper()
	select {
	case r := <-c.Send:
		t.Fatalf("unexpected response: %+v", r)
	default:
	}
}

// drainResp consumes and ignores one response from the client's Send channel.
func drainResp(t *testing.T, c *Client) {
	t.Helper()
	select {
	case <-c.Send:
	case <-time.After(time.Second):
		t.Fatal("timeout draining response")
	}
}

// ========================================================================
// Tests
// ========================================================================

func TestConversationKey(t *testing.T) {
	tests := []struct {
		a, b, want string
	}{
		{"alice", "bob", "alice:bob"},
		{"bob", "alice", "alice:bob"},
		{"alice", "alice", "alice:alice"},
		{"A", "B", "A:B"},
		{"z", "a", "a:z"},
	}
	for _, tc := range tests {
		got := conversationKey(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("conversationKey(%q, %q) = %q, want %q", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestRegister(t *testing.T) {
	h := NewHub()

	c := mustRegister(t, h, "alice")

	if _, ok := h.clients["alice"]; !ok {
		t.Error("alice not found in clients map")
	}
	sess, ok := h.users["alice"]
	if !ok {
		t.Fatal("alice not found in users map")
	}
	if !sess.Connected {
		t.Error("alice should be connected")
	}
	if time.Since(sess.ConnectedAt) > time.Second {
		t.Error("ConnectedAt should be recent")
	}
	assertEmpty(t, c)
}

func TestRegisterDuplicate(t *testing.T) {
	h := NewHub()
	mustRegister(t, h, "alice")
	mustFailRegister(t, h, "alice")
}

func TestRegisterEmptyUsername(t *testing.T) {
	h := NewHub()
	mustFailRegister(t, h, "")
}

// -----------------------------------------------------------------------
// Unregister
// -----------------------------------------------------------------------

func TestUnregister(t *testing.T) {
	h := NewHub()
	mustRegister(t, h, "alice")

	h.unregister <- unregisterCommand{username: "alice"}

	// Re-registering should succeed because unregister removed the client.
	_ = mustRegister(t, h, "alice")
}

func TestUnregisterMarksDisconnected(t *testing.T) {
	h := NewHub()
	mustRegister(t, h, "alice")

	h.unregister <- unregisterCommand{username: "alice"}

	// Give the hub goroutine time to process. (The channel send returns
	// before handleUnregister finishes because the channel is unbuffered.)
	time.Sleep(10 * time.Millisecond)

	sess, ok := h.users["alice"]
	if !ok {
		t.Fatal("alice should still be in users map after unregister")
	}
	if sess.Connected {
		t.Error("alice should be marked disconnected")
	}
}

func TestUnregisterNonexistent(t *testing.T) {
	h := NewHub()
	// should not panic
	h.unregister <- unregisterCommand{username: "ghost"}
}

// -----------------------------------------------------------------------
// Logout
// -----------------------------------------------------------------------

func TestLogout(t *testing.T) {
	h := NewHub()
	mustRegister(t, h, "alice")

	err := h.Logout("alice")
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}

	if _, ok := h.clients["alice"]; ok {
		t.Error("alice should be removed from clients")
	}
	if _, ok := h.users["alice"]; ok {
		t.Error("alice should be removed from users")
	}
}

func TestLogoutNonexistent(t *testing.T) {
	h := NewHub()
	err := h.Logout("ghost")
	if err != nil {
		t.Fatalf("Logout of nonexistent user should succeed: %v", err)
	}
}

// -----------------------------------------------------------------------
// Chat
// -----------------------------------------------------------------------

func TestChat(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")
	bob := mustRegister(t, h, "bob")

	h.inbound <- chatCmd(alice, "bob", "hello bob")

	// alice gets echo
	r1 := readResp(t, alice)
	if r1.Type != MessageTypeChat {
		t.Errorf("alice response type = %q, want %q", r1.Type, MessageTypeChat)
	}
	if r1.Message == nil || r1.Message.Text != "hello bob" {
		t.Errorf("alice message = %+v, want 'hello bob'", r1.Message)
	}

	// bob gets the message
	r2 := readResp(t, bob)
	if r2.Type != MessageTypeChat {
		t.Errorf("bob response type = %q, want %q", r2.Type, MessageTypeChat)
	}
	if r2.Message == nil || r2.Message.Text != "hello bob" {
		t.Errorf("bob message = %+v, want 'hello bob'", r2.Message)
	}

	// verify message stored under correct canonical key
	key := conversationKey("alice", "bob")
	msgs := h.messagesByConversation[key]
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Text != "hello bob" {
		t.Errorf("stored message = %q, want 'hello bob'", msgs[0].Text)
	}
}

func TestChatToOfflineRecipient(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")

	h.inbound <- chatCmd(alice, "bob", "hi")

	// alice still gets echo
	r := readResp(t, alice)
	if r.Type != MessageTypeChat {
		t.Errorf("alice response type = %q, want %q", r.Type, MessageTypeChat)
	}

	// message is stored even though bob is offline
	key := conversationKey("alice", "bob")
	msgs := h.messagesByConversation[key]
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
}

func TestChatMissingRecipient(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")

	h.inbound <- chatCmd(alice, "", "hi")

	r := readResp(t, alice)
	if r.Type != MessageTypeError || r.Error != "recipient is required" {
		t.Errorf("expected recipient error, got type=%q error=%q", r.Type, r.Error)
	}
}

func TestChatMissingText(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")

	h.inbound <- chatCmd(alice, "bob", "")

	r := readResp(t, alice)
	if r.Type != MessageTypeError || r.Error != "message text is required" {
		t.Errorf("expected text error, got type=%q error=%q", r.Type, r.Error)
	}
}

// -----------------------------------------------------------------------
// History
// -----------------------------------------------------------------------

func TestHistory(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")
	bob := mustRegister(t, h, "bob")

	// send two messages; consume alice echo + bob receipt for each
	h.inbound <- chatCmd(alice, "bob", "m1")
	drainResp(t, alice) // echo
	drainResp(t, bob)   // bob receives

	h.inbound <- chatCmd(alice, "bob", "m2")
	drainResp(t, alice)
	drainResp(t, bob)

	// request history
	h.inbound <- historyCmd(alice, "bob")

	r := readResp(t, alice)
	if r.Type != MessageTypeHistory {
		t.Fatalf("expected history type, got %q", r.Type)
	}
	if len(r.Messages) != 2 {
		t.Fatalf("expected 2 history messages, got %d", len(r.Messages))
	}
	if r.Messages[0].Text != "m1" || r.Messages[1].Text != "m2" {
		t.Errorf("messages = %q, %q; want m1, m2", r.Messages[0].Text, r.Messages[1].Text)
	}
}

func TestHistoryEmpty(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")

	h.inbound <- historyCmd(alice, "bob")

	r := readResp(t, alice)
	if r.Type != MessageTypeHistory {
		t.Fatalf("expected history type, got %q", r.Type)
	}
	if len(r.Messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(r.Messages))
	}
}

func TestHistoryMissingRecipient(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")

	h.inbound <- inboundCommand{
		request: ClientRequest{Type: MessageTypeHistory, Sender: "alice"},
		client:  alice,
	}

	r := readResp(t, alice)
	if r.Type != MessageTypeError || r.Error != "recipient is required for history request" {
		t.Errorf("expected error, got type=%q error=%q", r.Type, r.Error)
	}
}

// -----------------------------------------------------------------------
// Search
// -----------------------------------------------------------------------

func TestSearch(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")
	mustRegister(t, h, "bob")

	h.inbound <- chatCmd(alice, "bob", "hello world")
	drainResp(t, alice)

	h.inbound <- chatCmd(alice, "bob", "goodbye")
	drainResp(t, alice)

	h.inbound <- searchCmd(alice, "hello")

	r := readResp(t, alice)
	if r.Type != MessageTypeSearch {
		t.Fatalf("expected search type, got %q", r.Type)
	}
	if len(r.Messages) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(r.Messages))
	}
	if r.Messages[0].Text != "hello world" {
		t.Errorf("search result = %q", r.Messages[0].Text)
	}
}

func TestSearchBySender(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")
	mustRegister(t, h, "bob")

	h.inbound <- chatCmd(alice, "bob", "hi")
	drainResp(t, alice)

	h.inbound <- searchCmd(alice, "alice")

	r := readResp(t, alice)
	if r.Type != MessageTypeSearch {
		t.Fatalf("expected search type, got %q", r.Type)
	}
	if len(r.Messages) != 1 {
		t.Fatalf("expected 1 result, got %d", len(r.Messages))
	}
}

func TestSearchByRecipient(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")
	mustRegister(t, h, "bob")

	h.inbound <- chatCmd(alice, "bob", "hi")
	drainResp(t, alice)

	h.inbound <- searchCmd(alice, "bob")

	r := readResp(t, alice)
	if r.Type != MessageTypeSearch {
		t.Fatalf("expected search type, got %q", r.Type)
	}
	if len(r.Messages) != 1 {
		t.Fatalf("expected 1 result, got %d", len(r.Messages))
	}
}

func TestSearchNoResults(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")

	h.inbound <- searchCmd(alice, "nothing")

	r := readResp(t, alice)
	if r.Type != MessageTypeSearch {
		t.Fatalf("expected search type, got %q", r.Type)
	}
	if len(r.Messages) != 0 {
		t.Fatalf("expected 0 results, got %d", len(r.Messages))
	}
}

func TestSearchMissingQuery(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")

	h.inbound <- inboundCommand{
		request: ClientRequest{Type: MessageTypeSearch, Sender: "alice"},
		client:  alice,
	}

	r := readResp(t, alice)
	if r.Type != MessageTypeError || r.Error != "query is required for search" {
		t.Errorf("expected error, got type=%q error=%q", r.Type, r.Error)
	}
}

// -----------------------------------------------------------------------
// Unknown message type
// -----------------------------------------------------------------------

func TestUnknownMessageType(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")

	h.inbound <- inboundCommand{
		request: ClientRequest{Type: "invalid"},
		client:  alice,
	}

	r := readResp(t, alice)
	if r.Type != MessageTypeError {
		t.Fatalf("expected error type, got %q", r.Type)
	}
	if r.Error == "" {
		t.Error("expected non-empty error message")
	}
}

// -----------------------------------------------------------------------
// Messages survive logout
// -----------------------------------------------------------------------

func TestMessagesSurviveLogout(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")
	mustRegister(t, h, "bob")

	// send a message
	h.inbound <- chatCmd(alice, "bob", "before logout")
	drainResp(t, alice) // echo

	// logout alice
	if err := h.Logout("alice"); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	// re-register alice
	alice2 := mustRegister(t, h, "alice")

	// request history — should still see the message
	h.inbound <- historyCmd(alice2, "bob")

	r := readResp(t, alice2)
	if r.Type != MessageTypeHistory {
		t.Fatalf("expected history type, got %q", r.Type)
	}
	if len(r.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(r.Messages))
	}
	if r.Messages[0].Text != "before logout" {
		t.Errorf("message = %q, want 'before logout'", r.Messages[0].Text)
	}
}

// -----------------------------------------------------------------------
// Message ordering
// -----------------------------------------------------------------------

func TestMessageOrdering(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")
	mustRegister(t, h, "bob")

	texts := []string{"first", "second", "third"}
	for _, txt := range texts {
		h.inbound <- chatCmd(alice, "bob", txt)
		drainResp(t, alice) // consume echo
	}

	h.inbound <- historyCmd(alice, "bob")

	r := readResp(t, alice)
	if len(r.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(r.Messages))
	}
	for i, want := range texts {
		if r.Messages[i].Text != want {
			t.Errorf("message[%d] = %q, want %q", i, r.Messages[i].Text, want)
		}
	}
}

// -----------------------------------------------------------------------
// Multiple conversations are isolated
// -----------------------------------------------------------------------

func TestConversationIsolation(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")
	bob := mustRegister(t, h, "bob")
	charlie := mustRegister(t, h, "charlie")

	// alice -> bob
	h.inbound <- chatCmd(alice, "bob", "ab")
	drainResp(t, alice) // echo
	drainResp(t, bob)   // bob receives

	// alice -> charlie
	h.inbound <- chatCmd(alice, "charlie", "ac")
	drainResp(t, alice)   // echo
	drainResp(t, charlie) // charlie receives

	// history alice:bob
	h.inbound <- historyCmd(alice, "bob")
	r1 := readResp(t, alice)
	if len(r1.Messages) != 1 || r1.Messages[0].Text != "ab" {
		t.Errorf("alice:bob history = %+v", r1.Messages)
	}

	// history alice:charlie
	h.inbound <- historyCmd(alice, "charlie")
	r2 := readResp(t, alice)
	if len(r2.Messages) != 1 || r2.Messages[0].Text != "ac" {
		t.Errorf("alice:charlie history = %+v", r2.Messages)
	}

	// bob can see his conversation with alice too (same canonical key)
	h.inbound <- historyCmd(bob, "alice")
	r3 := readResp(t, bob)
	if len(r3.Messages) != 1 || r3.Messages[0].Text != "ab" {
		t.Errorf("bob:alice history = %+v", r3.Messages)
	}
}

// -----------------------------------------------------------------------
// Case-insensitive search
// -----------------------------------------------------------------------

func TestSearchCaseInsensitive(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")
	mustRegister(t, h, "bob")

	h.inbound <- chatCmd(alice, "bob", "Hello World")
	drainResp(t, alice)

	h.inbound <- searchCmd(alice, "hello")

	r := readResp(t, alice)
	if len(r.Messages) != 1 {
		t.Fatalf("case-insensitive search failed: got %d results", len(r.Messages))
	}
}

// -----------------------------------------------------------------------
// ClientRequest Sender override (ReadPump simulation)
// -----------------------------------------------------------------------

func TestSenderOverride(t *testing.T) {
	h := NewHub()
	alice := mustRegister(t, h, "alice")
	mustRegister(t, h, "bob")

	// Simulate what ReadPump does: Sender is set from c.Username
	h.inbound <- chatCmd(alice, "bob", "msg")

	r := readResp(t, alice)
	if r.Type != MessageTypeChat {
		t.Fatalf("expected chat type, got %q", r.Type)
	}
	if r.Message.Sender != "alice" {
		t.Errorf("sender = %q, want alice", r.Message.Sender)
	}
}
