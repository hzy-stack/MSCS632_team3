# MSCS 632 — WebSocket Chat Application (Team 3)

A web-based, real-time chat application implemented in **Go** and 
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

### 1. Go server

```bash
cd go-chat-server
go run ./cmd/server          # http://localhost:8080
```

Check it: open <http://localhost:8080/health> → `{"status":"ok"}`.

### 2. Java server (Spring Boot)

```bash
mvn spring-boot:run          # http://localhost:8081
```

Check it: open <http://localhost:8081/users> → `[]`.

### 3. Web client

Serve the `web-client` folder and open it in a browser:

```bash
cd web-client
python3 -m http.server 5050  # then open http://localhost:5050
```

On the connect screen: pick the **Server** (Go or Java), set **Host** to the chat server,
enter a username, and **Connect**.
Open a second browser window as a different username to chat between them.

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
```
