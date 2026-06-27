# Go Chat Server

A simple two-user WebSocket chat server written in Go.

## Overview

- In-memory storage only (all data is lost when the server restarts)
- No database, no authentication, no encryption
- One-on-one (two-user) chats only
- WebSocket-based real-time messaging
- Message history and keyword search

## Dependencies

- [Echo v4](https://github.com/labstack/echo) - HTTP server and routing
- [coder/websocket](https://github.com/coder/websocket) - WebSocket support
- [google/uuid](https://github.com/google/uuid) - Message IDs

## Quick Start

```bash
cd go-chat-server
make run          # build and start the server on port 8080
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Compile binary to `bin/server` |
| `make run` | Build and start the server |
| `make test` | Run all tests with verbose output |
| `make test-cover` | Run tests and display coverage |
| `make vet` | Run `go vet` static analysis |
| `make fmt` | Format all Go source files |
| `make tidy` | Tidy module dependencies |
| `make clean` | Remove build artifacts |
| `make all` | Build, vet, test (full CI check) |

Tests cover: registration, unregistration, logout, chat messaging, conversation
history, full-text search (case-insensitive), message ordering, conversation
isolation, and message survival across logout cycles.

## WebSocket URL

Connect using a query parameter for the username:

```
ws://localhost:8080/ws?username=alice
ws://localhost:8080/ws?username=bob
```

- Empty usernames are rejected.
- Duplicate active usernames are rejected.

## JSON Message Examples

### Send a chat message

```json
{
  "type": "chat",
  "recipient": "bob",
  "text": "Hello Bob"
}
```

The server sets `sender` automatically from the WebSocket username. Do not include a `sender` field.

### Get conversation history

```json
{
  "type": "history",
  "recipient": "bob"
}
```

The conversation key is deterministic (alice:bob and bob:alice point to the same conversation).

### Search messages

```json
{
  "type": "search",
  "query": "hello"
}
```

Searches text, sender, and recipient fields across all conversations involving the requesting user. Case-insensitive.

### Server responses

The server sends back JSON with the same `type` field:

- `chat` - A single `message` object (sent to both sender and recipient if online)
- `history` - A `messages` array
- `search` - A `messages` array
- `error` - An `error` string

## HTTP Endpoints

### Logout

```
POST /logout?username=alice
POST /logout
Content-Type: application/json

{"username":"alice"}
```

Removes the user from active sessions and closes their WebSocket connection.
Messages already sent are preserved until the server restarts.

### Health check

```
GET /health
→ {"status":"ok"}
```

## Architecture

The Go implementation uses a **channel-based hub** instead of `sync.RWMutex`. The Hub runs as a single goroutine and owns all shared state:

- `users` map
- `clients` map
- `messagesByConversation` map

The Hub receives commands through channels (`register`, `unregister`, `logout`,
`inbound`) and processes them sequentially. No other goroutine directly modifies
the shared maps.

## Testing Checklist

1. Start the server
2. Connect Alice
3. Connect Bob
4. Try connecting another Alice (should be rejected)
5. Alice sends Bob a message
6. Bob receives it live
7. Alice receives confirmation
8. Bob sends Alice a message
9. Alice receives it live
10. Alice requests history with Bob
11. Bob requests history with Alice
12. Both history requests show the same conversation
13. Alice sends Charlie a message while Charlie is offline
14. Message is saved but not delivered live
15. Charlie connects
16. Charlie requests history with Alice
17. Charlie sees Alice's earlier message
18. Search by keyword works
19. Search by username works
20. Empty message is rejected
21. Missing recipient is rejected
22. Restarting the server clears all data
