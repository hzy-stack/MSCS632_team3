// Command DemoSeeder seeds the chat server with realistic demo data.
//
// It connects users via WebSocket, sends conversations, then disconnects.
// After running, the server contains full message history retrievable via
// the normal {"type":"history","recipient":"..."} WebSocket request.
//
// Usage:
//
//    mvn exec:java          # requires the server to be running on :8080
package com.chat.demo;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.WebSocket;
import java.util.List;
import java.util.Map;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.CompletionStage;
import java.util.concurrent.LinkedBlockingQueue;
import java.util.concurrent.TimeUnit;

public class DemoSeeder {

    private static final String SERVER_URL = "ws://localhost:8080/ws";
    private static final ObjectMapper MAPPER = new ObjectMapper();
    private static final HttpClient HTTP = HttpClient.newHttpClient();

    public static void main(String[] args) throws Exception {
        System.out.println("=== Chat Server Demo Seed ===");
        System.out.println();

        seedLive();
        seedOffline();
        seedMore();

        System.out.println();
        System.out.println("=== Demo data seeded successfully ===");
        System.out.println("Connect via ws://localhost:8080/ws?username=alice");
        System.out.println("  then send {\"type\":\"history\",\"recipient\":\"bob\"}");
    }

    // --- helpers ---

    record ChatConn(WebSocket ws, BlockingQueue<String> inbox) {}

    static ChatConn dial(String username) {
        BlockingQueue<String> inbox = new LinkedBlockingQueue<>();
        WebSocket ws = HTTP.newWebSocketBuilder()
                .buildAsync(URI.create(SERVER_URL + "?username=" + username), new WebSocket.Listener() {
                    @Override
                    public void onOpen(WebSocket webSocket) {
                        webSocket.request(1);
                    }

                    @Override
                    public CompletionStage<?> onText(WebSocket webSocket, CharSequence data, boolean last) {
                        inbox.offer(data.toString());
                        webSocket.request(1);
                        return null;
                    }

                    @Override
                    public void onError(WebSocket webSocket, Throwable error) {
                        System.err.println("  ws error: " + error.getMessage());
                    }
                })
                .join();
        return new ChatConn(ws, inbox);
    }

    // Sends a chat message and drains the sender's echo.
    static void writeChat(ChatConn conn, String recipient, String text) throws Exception {
        String json = MAPPER.writeValueAsString(
                Map.of("type", "chat", "recipient", recipient, "text", text));
        conn.ws().sendText(json, true).join();
        String raw = conn.inbox().poll(3, TimeUnit.SECONDS);
        if (raw != null) {
            JsonNode resp = MAPPER.readTree(raw);
            if ("error".equals(resp.path("type").asText())) {
                System.err.println("  server error: " + resp.path("error").asText());
            }
        }
    }

    // Reads and returns the text of the next incoming message, or null on timeout.
    static String drainOne(ChatConn conn) throws Exception {
        String raw = conn.inbox().poll(3, TimeUnit.SECONDS);
        if (raw == null) return null;
        JsonNode resp = MAPPER.readTree(raw);
        JsonNode msg = resp.path("message");
        return msg.isMissingNode() ? null : msg.path("text").asText(null);
    }

    // --- live conversation (both users online, messages delivered in real time) ---

    static void seedLive() throws Exception {
        System.out.println("--- Live: Alice <-> Bob (both online) ---");

        ChatConn alice = dial("alice");
        System.out.println("[alice connected]");

        ChatConn bob = dial("bob");
        System.out.println("[bob connected]");

        // Alice sends to Bob.  After each send:
        //   1. writeChat drains Alice's echo
        //   2. caller drains Bob's delivered copy
        List<String> aliceMsgs = List.of(
                "Hey Bob! How's the Sprint 2 progress?",
                "I pushed the auth middleware - can you review?",
                "Also, the test coverage dropped to 78%, we need more tests."
        );
        for (String msg : aliceMsgs) {
            System.out.printf("  alice -> bob: \"%s\"%n", msg);
            writeChat(alice, "bob", msg);
            String received = drainOne(bob);
            if (received != null) System.out.printf("  bob received: \"%s\"%n", received);
            Thread.sleep(300);
        }

        // Bob responds.  Same pattern: write, drain echo, drain recipient copy.
        List<String> bobMsgs = List.of(
                "Sprint 2 is almost done, just finishing the WebSocket handler!",
                "Sure, I'll review the auth PR right after lunch."
        );
        for (String msg : bobMsgs) {
            System.out.printf("  bob -> alice: \"%s\"%n", msg);
            writeChat(bob, "alice", msg);
            String received = drainOne(alice);
            if (received != null) System.out.printf("  alice received: \"%s\"%n", received);
            Thread.sleep(300);
        }

        alice.ws().sendClose(WebSocket.NORMAL_CLOSURE, "demo done").join();
        bob.ws().sendClose(WebSocket.NORMAL_CLOSURE, "demo done").join();
        System.out.println("[alice disconnected]");
        System.out.println("[bob disconnected]");
        System.out.println();
    }

    // --- offline messages (recipient not connected, stored server-side) ---

    static void seedOffline() throws Exception {
        System.out.println("--- Offline: Charlie -> Alice (Alice not connected) ---");

        ChatConn charlie = dial("charlie");
        System.out.println("[charlie connected]");

        List<String> messages = List.of(
                "Alice, I found a bug in message search - it's case-sensitive for sender names.",
                "I opened issue #42 with a proposed fix.",
                "Also, the README needs updating with the new logout endpoint."
        );
        for (String msg : messages) {
            System.out.printf("  charlie -> alice: \"%s\"%n", msg);
            writeChat(charlie, "alice", msg);
            Thread.sleep(200);
        }

        charlie.ws().sendClose(WebSocket.NORMAL_CLOSURE, "demo done").join();
        System.out.println("[charlie disconnected]");
        System.out.println();
    }

    // --- additional messages for search/history depth ---

    static void seedMore() throws Exception {
        System.out.println("--- More: Alice -> Bob (Bob offline) ---");

        ChatConn alice = dial("alice");
        System.out.println("[alice connected]");

        List<String> messages = List.of(
                "The deployment pipeline is broken again.",
                "We should upgrade the Java version to 21.",
                "Meeting at 3pm to discuss the architecture review."
        );
        for (String msg : messages) {
            System.out.printf("  alice -> bob: \"%s\"%n", msg);
            writeChat(alice, "bob", msg);
            Thread.sleep(200);
        }

        alice.ws().sendClose(WebSocket.NORMAL_CLOSURE, "demo done").join();
        System.out.println("[alice disconnected]");
        System.out.println();
    }
}
