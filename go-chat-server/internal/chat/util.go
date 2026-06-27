package chat

// conversationKey returns a canonical, order-independent key for a
// two-party conversation.  It always places the lexicographically smaller
// username first, so "alice:bob" and "bob:alice" produce the same key.
// This allows the hub to look up conversation history without caring
// which participant is the sender vs. recipient.
func conversationKey(a string, b string) string {
	if a < b {
		return a + ":" + b
	}

	return b + ":" + a
}
