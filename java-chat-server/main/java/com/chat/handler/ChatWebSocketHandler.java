package com.chat.handler;

import com.chat.service.ChatService;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.stereotype.Component;
import org.springframework.web.socket.CloseStatus;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;
import org.springframework.web.socket.handler.TextWebSocketHandler;

import java.util.Map;

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
        if (username == null || !chatService.isRegistered(username)) {
            session.close(CloseStatus.POLICY_VIOLATION);
            return;
        }
        chatService.registerSession(username, session);
    }

    @Override
    protected void handleTextMessage(WebSocketSession session, TextMessage message) throws Exception {
        String sender = (String) session.getAttributes().get("username");
        JsonNode json = objectMapper.readTree(message.getPayload());

        String recipient = json.path("recipient").asText(null);
        String text      = json.path("text").asText(null);

        if (recipient == null || recipient.isBlank() || text == null || text.isBlank()) {
            String error = objectMapper.writeValueAsString(Map.of("error", "Recipient and text are required"));
            synchronized (session) {
                session.sendMessage(new TextMessage(error));
            }
            return;
        }

        // sendMessage stores the message and pushes it to both recipient and sender
        chatService.sendMessage(sender, recipient.trim(), text.trim());
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
        if (session.isOpen()) {
            session.close(CloseStatus.SERVER_ERROR);
        }
    }
}
