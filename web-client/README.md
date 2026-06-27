# Web Client

A single-page client (`index.html`) that renders the chat UI and connects to **either**
backend in this repo — the Go server or the Java (Spring Boot) server. No build step.

## Run it

Open `index.html` in a browser, or serve the folder (recommended, especially for Go):

```bash
cd web-client
python3 -m http.server 5500   # then visit http://localhost:5500
```

On the connect screen choose the server, set the host, enter a username, and Connect.

## How it talks to each server

| | Go server | Java server (Spring Boot) |
|---|---|---|
| Login | none — connect directly | `POST /login` first (3–20 letters/numbers) |
| Connect | `ws://host/ws?username=` | `ws://host/ws?username=` |
| Send | `{ "type":"chat", "recipient", "text" }` | `{ "recipient", "text" }` |
| Receive | `{ "type":"chat", "message":{…} }` | raw `Message` object |
| History | WS `{ "type":"history", "recipient" }` | `GET /history/{me}/{peer}` |
| Search | WS `{ "type":"search", "query" }` | `GET /search?q=` |

Both return the same `Message` shape (`messageId, sender, recipient, text, timestamp`),
so the UI renders them identically.

## Notes

- **Ports:** both servers default to `:8080`. To run them at the same time, start one on a
  different port and point the client's Host field at it.
- **Go + cross-origin:** the Go server uses `coder/websocket`'s default `Accept`, which
  rejects WebSocket handshakes from a different origin. Either serve this client from the
  Go server itself, or set `websocket.AcceptOptions{InsecureSkipVerify: true}` (dev only)
  or `OriginPatterns` in `HandleWebSocket`. The Java server already allows all origins.
- **Two-user model:** type the other person's username in “Chat with” and click Open. Run a
  second browser/tab logged in as that user to chat back and forth.
