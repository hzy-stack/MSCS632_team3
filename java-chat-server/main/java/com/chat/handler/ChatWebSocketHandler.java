package com.chat.handler;

import com.chat.model.ServerResponse;
import com.chat.service.ChatService;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.stereotype.Component;
import org.springframework.web.socket.CloseStatus;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;
import org.springframework.web.socket.handler.TextWebSocketHandler;

@Component
public class ChatWebSocketHandler extends TextWebSocketHandler {

    private final ChatService chatService;
    private final ObjectMapper objectMapper;

    public ChatWebSocketHandler(ChatService chatService, ObjectMapper objectMapper) {
        this.chatService = chatService;
        this.objectMapper = objectMapper;
    }

    @Override
    public void afterConnectionEstablished(WebSocketSession session) throws Exception {
        String username = (String) session.getAttributes().get("username");
        // Reject missing or malformed usernames before touching any service state.
        if (username == null || !username.matches("[a-zA-Z0-9]{3,20}")) {
            session.close(CloseStatus.POLICY_VIOLATION);
            return;
        }
        // Auto-register so the frontend can connect without a prior HTTP login,
        // matching the Go server's behaviour and enabling a shared frontend.
        if (!chatService.isRegistered(username)) {
            chatService.login(username);
        }
        chatService.registerSession(username, session);
    }

    @Override
    protected void handleTextMessage(WebSocketSession session, TextMessage message) throws Exception {
        String sender = (String) session.getAttributes().get("username");
        JsonNode json = objectMapper.readTree(message.getPayload());

        // Default to "chat" so clients that omit the type field still work.
        String type = json.path("type").asText("chat");

        if ("history".equals(type)) {
            String recipient = json.path("recipient").asText(null);
            if (recipient == null || recipient.isBlank()) {
                sendError(session, "Recipient is required for history");
                return;
            }
            String responseJson = objectMapper.writeValueAsString(
                    ServerResponse.history(chatService.getHistory(sender, recipient.trim())));
            synchronized (session) {
                session.sendMessage(new TextMessage(responseJson));
            }
            return;
        }

        // type == "chat" (or any unrecognised type falls through to here)
        String recipient = json.path("recipient").asText(null);
        String text      = json.path("text").asText(null);

        if (recipient == null || recipient.isBlank() || text == null || text.isBlank()) {
            sendError(session, "Recipient and text are required");
            return;
        }

        // sendMessage stores the message and pushes it to both recipient and sender
        chatService.sendMessage(sender, recipient.trim(), text.trim());
    }

    private void sendError(WebSocketSession session, String errorMessage) throws Exception {
        String json = objectMapper.writeValueAsString(ServerResponse.error(errorMessage));
        synchronized (session) {
            session.sendMessage(new TextMessage(json));
        }
    }

    @Override
    public void afterConnectionClosed(WebSocketSession session, CloseStatus status) {
        String username = (String) session.getAttributes().get("username");
        if (username != null) {
            chatService.removeSession(username, session);
        }
    }

    @Override
    public void handleTransportError(WebSocketSession session, Throwable exception) throws Exception {
        String username = (String) session.getAttributes().get("username");
        if (username != null) {
            chatService.removeSession(username, session);
        }
        // afterConnectionClosed may not fire after a transport error, so we
        // close the session explicitly to guarantee cleanup.
        if (session.isOpen()) {
            session.close(CloseStatus.SERVER_ERROR);
        }
    }
}
