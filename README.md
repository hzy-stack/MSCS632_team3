# MSCS 632 — WebSocket Chat Application (Team 3)

A web-based, real-time chat application implemented twice — once in **Go** and once in
**Java (Spring Boot)** — behind a shared **web client**. A user connects over a WebSocket
with a username and has one-on-one (two-user) conversations with real-time messaging,
in-memory history, and keyword search. Holding the behavior constant lets us compare the
two language implementations directly.

## Functional Requirements

### In Scope

- Web client connecting to a backend server through WebSockets.
- Username entry before joining the chat.
- Two-user conversations only. Each chat contains exactly two users.
- Real-time message sending and receiving while both users are connected.
- In-memory storage for users, active connections, and messages.
- Message history retrieval while the server process is still running.
- Basic global keyword search across stored messages.
- Basic error handling for invalid input, duplicate active usernames, and disconnected recipients.

### Out of Scope

- Password-based authentication or account security.
- Persistent database storage after server restart.
- Group chats with more than two users.
- File attachments, images, voice messages, or video calls.
- End-to-end encryption or advanced security features.
- Advanced search ranking, fuzzy matching, or full-text indexing.
- A complex frontend beyond the basic screens needed to test the backend.

## Repository Layout

```
MSCS632_team3/
  README.md
  pom.xml                       # Java (Spring Boot) build
  go-chat-server/               # Go implementation (Echo + coder/websocket)
    cmd/server/main.go
    internal/chat/*.go
    go.mod
  java-chat-server/             # Java implementation (Spring Boot)
    main/java/com/chat/...
    main/resources/
  web-client/                   # browser client for both servers
    index.html
  docs/                         # design & comparison documents
```

## Running It

> Both servers listen on **port 8080** by default, so run only one at a time — or change a
> port if you want both up. Storage is in-memory: restarting a server clears all users and
> messages.

### 1. Go server

```bash
cd go-chat-server
go run ./cmd/server          # http://localhost:8080
```

Check it: open <http://localhost:8080/health> → `{"status":"ok"}`.

### 2. Java server (Spring Boot)

```bash
mvn spring-boot:run          # from the project root (where pom.xml is) → http://localhost:8080
```

Check it: open <http://localhost:8080/users> → `[]`.

### 3. Web client

Serve the `web-client` folder and open it in a browser:

```bash
cd web-client
python3 -m http.server 5050  # then open http://localhost:5050
```

On the connect screen: pick the **Server** (Go or Java), set **Host** to the chat server
(`localhost:8080` — this is the server, *not* the page's own port), enter a username, and
**Connect**. Open a second browser window as a different username to chat between them.

## Client ↔ Server Protocols

Both servers expose the WebSocket endpoint at `/ws?username=<name>` and return the same
`Message` shape: `{ messageId, sender, recipient, text, timestamp }`.

| Action | Go server | Java server |
|--------|-----------|-------------|
| Login | none (connect directly) | `POST /login` `{username}` (3–20 letters/numbers) |
| Send | WS `{ "type":"chat", "recipient", "text" }` | WS `{ "recipient", "text" }` |
| Receive | WS `{ "type":"chat", "message":{…} }` | WS raw `Message` object |
| History | WS `{ "type":"history", "recipient" }` | `GET /history/{u1}/{u2}` |
| Search | WS `{ "type":"search", "query" }` | `GET /search?q=` |
| Online users | — | `GET /users` |
| Logout | `POST /logout?username=` | `DELETE /logout/{username}` |

## Troubleshooting

- **"WebSocket … failed"** — the **Host** field must point at the chat server
  (`localhost:8080`), not the port serving the HTML (e.g. a static server on 5050/5501).
- **Go cross-origin** — `coder/websocket` rejects WebSocket handshakes from a different
  origin. When opening the client from a file or a different port, set
  `websocket.AcceptOptions{ InsecureSkipVerify: true }` in `HandleWebSocket` (dev only), or
  serve the client from the Go server itself. The Java server already allows all origins.
- **`Address already in use`** — a previous server/static-server is still on that port.
  Find and stop it: `lsof -i :<port>` then `kill <PID>`.
- **Restart to apply changes** — a running server must be stopped (Ctrl+C) and started again
  for code edits to take effect.

## Languages & Frameworks

- **Go:** Echo (HTTP routing), `coder/websocket` (+ `wsjson`), `google/uuid`. Concurrency via
  a single hub goroutine over channels (no locks).
- **Java:** Spring Boot 3 (`spring-boot-starter-web`, `spring-boot-starter-websocket`),
  Jackson. Concurrency via `ConcurrentHashMap` and `synchronized` blocks.

See `docs/` for the full design and the Java-vs-Go comparison report.

---

## Java Server: Wire Protocol Details

Connect via WebSocket with your username as a query parameter:

```
ws://localhost:8080/ws?username=alice
ws://localhost:8080/ws?username=bob
```

**Username rules:**
- 3 to 20 characters
- Letters and numbers only (no spaces or special characters)
- If the username is not already registered, the server registers it automatically on connect
- If the username is already connected on another session, the old session is closed and replaced

All WebSocket messages are JSON. Every message from the client includes a `type` field.

### Send a chat message

```json
{
  "type": "chat",
  "recipient": "bob",
  "text": "Hello Bob"
}
```

The server sets `sender` from the WebSocket connection — do not include it in the request.  
The `type` field may be omitted; the server defaults to `"chat"`.

### Request conversation history

```json
{
  "type": "history",
  "recipient": "bob"
}
```

Returns the full message history between the requesting user and `recipient`.

### Server responses

Every response from the server includes a `type` field. Fields not relevant to the type are omitted.

| `type` | Additional fields | Description |
|--------|-------------------|-------------|
| `chat` | `message` | A single message object, sent to both recipient (if online) and sender |
| `history` | `messages` | Array of message objects for the requested conversation |
| `error` | `error` | Human-readable error string |

**Message object:**

```json
{
  "messageId": "3f2a1b...",
  "sender":    "alice",
  "recipient": "bob",
  "text":      "Hello Bob",
  "timestamp": "2026-06-27T15:04:05"
}
```

**Full exchange example:**

```
Client → {"type":"chat","recipient":"bob","text":"Hello"}
Server → {"type":"chat","message":{"messageId":"...","sender":"alice","recipient":"bob","text":"Hello","timestamp":"..."}}

Client → {"type":"history","recipient":"bob"}
Server → {"type":"history","messages":[{"messageId":"...","sender":"alice","recipient":"bob","text":"Hello","timestamp":"..."}]}
```

---

## Java Server: HTTP Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/login` | Register a username (validates format; returns 400 with an error message on failure) |
| `DELETE` | `/logout/{username}` | Deregister a user and close their WebSocket session |
| `GET` | `/history/{u1}/{u2}` | Retrieve conversation history between two users |
| `GET` | `/search?q={query}` | Search all messages by text or sender name (case-insensitive) |
| `GET` | `/users` | List usernames with an active WebSocket connection |

**POST /login request body:**

```json
{ "username": "alice" }
```

**POST /login response (200):**

```json
{ "username": "alice" }
```

**POST /login response (400):**

```json
{ "error": "Username must be between 3 and 20 characters" }
```

> Note: connecting via WebSocket auto-registers the user, so calling `/login` first is optional. It is provided for clients that prefer an explicit registration step.

---

## Java Server: Running the Tests

```bash
mvn test
```

58 tests across four suites:

| Suite | Tests | What it covers |
|-------|-------|----------------|
| `ChatStoreTest` | 6 | Conversation key symmetry and list creation |
| `ChatServiceTest` | 21 | Login, logout, messaging, history, search, session management |
| `ChatControllerTest` | 15 | All HTTP endpoints with valid and invalid inputs |
| `ChatWebSocketHandlerTest` | 16 | Connection lifecycle, message routing, history requests, error handling |

---

## Java Server: Demo Seeder

A seed script populates a running server with realistic conversation data — useful for manual testing or demonstrating the app.

In a second terminal (while the server is running):

```bash
mvn exec:java
```

This connects three users, exchanges messages across live and offline scenarios, then disconnects. After it finishes, you can connect as `alice` and request history with `bob` or `charlie` to see the seeded data.

Expected output:

```
=== Chat Server Demo Seed ===

--- Live: Alice <-> Bob (both online) ---
[alice connected]
[bob connected]
  alice -> bob: "Hey Bob! How's the Sprint 2 progress?"
  bob received: "Hey Bob! How's the Sprint 2 progress?"
  ...
[alice disconnected]
[bob disconnected]

--- Offline: Charlie -> Alice (Alice not connected) ---
[charlie connected]
  charlie -> alice: "Alice, I found a bug in message search..."
  ...
[charlie disconnected]

--- More: Alice -> Bob (Bob offline) ---
[alice connected]
  alice -> bob: "The deployment pipeline is broken again."
  ...
[alice disconnected]

=== Demo data seeded successfully ===
Connect via ws://localhost:8080/ws?username=alice
  then send {"type":"history","recipient":"bob"}
```

---

## Java Server: Project Structure

```
java-chat/
├── pom.xml
├── java-chat-server/
│   └── main/
│       ├── java/com/chat/
│       │   ├── ChatApplication.java          — Spring Boot entry point
│       │   ├── config/
│       │   │   ├── WebSocketConfig.java      — registers the WebSocket endpoint at /ws
│       │   │   └── UsernameHandshakeInterceptor.java  — extracts username from query string
│       │   ├── controller/
│       │   │   └── ChatController.java       — REST endpoints (login, logout, history, search, users)
│       │   ├── demo/
│       │   │   └── DemoSeeder.java           — standalone seed script
│       │   ├── handler/
│       │   │   └── ChatWebSocketHandler.java — WebSocket message routing (chat, history)
│       │   ├── model/
│       │   │   ├── Message.java              — immutable message record (id, sender, recipient, text, timestamp)
│       │   │   ├── ServerResponse.java       — outgoing envelope with type field
│       │   │   └── UserSession.java          — tracks registration and connection state per user
│       │   ├── service/
│       │   │   └── ChatService.java          — business logic (send, store, search, push)
│       │   └── store/
│       │       └── ChatStore.java            — thread-safe in-memory data store
│       └── resources/
│           └── application.properties
└── src/
    └── test/java/com/chat/
        ├── controller/ChatControllerTest.java
        ├── handler/ChatWebSocketHandlerTest.java
        ├── service/ChatServiceTest.java
        └── store/ChatStoreTest.java
```

---

## Java Server: Architecture Notes

**Thread safety** — The store uses `ConcurrentHashMap` for all shared maps and `Collections.synchronizedList` for message lists. WebSocket writes are synchronized on the session object to prevent interleaved frames. The two-argument `ConcurrentHashMap.remove(key, value)` is used when removing sessions to avoid accidentally evicting a newer session that replaced one during a reconnect.

**Conversation keys** — Two usernames are sorted alphabetically and joined with `:` before being used as a map key (`alice:bob`). This guarantees that both participants always reference the same message list regardless of who initiates the conversation.

**Go compatibility** — The wire protocol (the `type` field, `ServerResponse` envelope, and WebSocket-based history) matches the Go server's protocol, enabling a shared frontend to connect to either implementation without changes.
