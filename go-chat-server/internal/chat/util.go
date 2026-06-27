package chat

func conversationKey(a string, b string) string {
	if a < b {
		return a + ":" + b
	}

	return b + ":" + a
}
