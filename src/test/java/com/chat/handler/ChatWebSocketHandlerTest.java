package com.chat.handler;

import com.chat.model.Message;
import com.chat.service.ChatService;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.web.socket.CloseStatus;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

class ChatWebSocketHandlerTest {

    private ChatService chatService;
    private ChatWebSocketHandler handler;
    private WebSocketSession session;

    @BeforeEach
    void setUp() {
        chatService = mock(ChatService.class);
        ObjectMapper objectMapper = new ObjectMapper()
                .registerModule(new JavaTimeModule())
                .disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS);
        handler = new ChatWebSocketHandler(chatService, objectMapper);
        session = mock(WebSocketSession.class);
        when(session.isOpen()).thenReturn(true);
    }

    private void putUsername(String username) {
        Map<String, Object> attrs = new HashMap<>();
        attrs.put("username", username);
        when(session.getAttributes()).thenReturn(attrs);
    }

    // ── Connection established ────────────────────────────────────────────────

    @Test
    void afterConnectionEstablished_registersSession_forAlreadyLoggedInUser() throws Exception {
        putUsername("alice");
        when(chatService.isRegistered("alice")).thenReturn(true);

        handler.afterConnectionEstablished(session);

        verify(chatService, never()).login(any());
        verify(chatService).registerSession("alice", session);
        verify(session, never()).close(any(CloseStatus.class));
    }

    @Test
    void afterConnectionEstablished_autoRegisters_whenUserNotLoggedIn() throws Exception {
        putUsername("ghost");
        when(chatService.isRegistered("ghost")).thenReturn(false);

        handler.afterConnectionEstablished(session);

        verify(chatService).login("ghost");
        verify(chatService).registerSession("ghost", session);
        verify(session, never()).close(any(CloseStatus.class));
    }

    @Test
    void afterConnectionEstablished_closesSession_whenUsernameAttributeMissing() throws Exception {
        when(session.getAttributes()).thenReturn(new HashMap<>());

        handler.afterConnectionEstablished(session);

        verify(session).close(CloseStatus.POLICY_VIOLATION);
        verify(chatService, never()).registerSession(any(), any());
    }

    @Test
    void afterConnectionEstablished_closesSession_whenUsernameTooShort() throws Exception {
        putUsername("ab");

        handler.afterConnectionEstablished(session);

        verify(session).close(CloseStatus.POLICY_VIOLATION);
        verify(chatService, never()).registerSession(any(), any());
    }

    @Test
    void afterConnectionEstablished_closesSession_whenUsernameHasSpecialChars() throws Exception {
        putUsername("alice!");

        handler.afterConnectionEstablished(session);

        verify(session).close(CloseStatus.POLICY_VIOLATION);
        verify(chatService, never()).registerSession(any(), any());
    }

    // ── Text message — chat ───────────────────────────────────────────────────

    @Test
    void handleTextMessage_callsSendMessage_withValidChatPayload() throws Exception {
        putUsername("alice");
        String json = "{\"type\":\"chat\",\"recipient\":\"bob\",\"text\":\"Hello\"}";

        handler.handleTextMessage(session, new TextMessage(json));

        verify(chatService).sendMessage("alice", "bob", "Hello");
    }

    @Test
    void handleTextMessage_defaultsToChatType_whenTypeFieldAbsent() throws Exception {
        putUsername("alice");
        String json = "{\"recipient\":\"bob\",\"text\":\"Hello\"}";

        handler.handleTextMessage(session, new TextMessage(json));

        verify(chatService).sendMessage("alice", "bob", "Hello");
    }

    @Test
    void handleTextMessage_sendsError_whenRecipientBlank() throws Exception {
        putUsername("alice");
        String json = "{\"type\":\"chat\",\"recipient\":\"\",\"text\":\"Hello\"}";

        handler.handleTextMessage(session, new TextMessage(json));

        verify(chatService, never()).sendMessage(any(), any(), any());
        verify(session).sendMessage(any(TextMessage.class));
    }

    @Test
    void handleTextMessage_sendsError_whenTextBlank() throws Exception {
        putUsername("alice");
        String json = "{\"type\":\"chat\",\"recipient\":\"bob\",\"text\":\"\"}";

        handler.handleTextMessage(session, new TextMessage(json));

        verify(chatService, never()).sendMessage(any(), any(), any());
        verify(session).sendMessage(any(TextMessage.class));
    }

    @Test
    void handleTextMessage_trimsRecipientAndText() throws Exception {
        putUsername("alice");
        String json = "{\"recipient\":\" bob \",\"text\":\" Hello \"}";

        handler.handleTextMessage(session, new TextMessage(json));

        verify(chatService).sendMessage("alice", "bob", "Hello");
    }

    // ── Text message — history ────────────────────────────────────────────────

    @Test
    void handleTextMessage_returnsHistory_onHistoryType() throws Exception {
        putUsername("alice");
        Message msg = new Message("alice", "bob", "Hello");
        when(chatService.getHistory("alice", "bob")).thenReturn(List.of(msg));
        String json = "{\"type\":\"history\",\"recipient\":\"bob\"}";

        handler.handleTextMessage(session, new TextMessage(json));

        verify(chatService).getHistory("alice", "bob");
        verify(session).sendMessage(any(TextMessage.class));
    }

    @Test
    void handleTextMessage_sendsError_whenHistoryRecipientBlank() throws Exception {
        putUsername("alice");
        String json = "{\"type\":\"history\",\"recipient\":\"\"}";

        handler.handleTextMessage(session, new TextMessage(json));

        verify(chatService, never()).getHistory(any(), any());
        verify(session).sendMessage(any(TextMessage.class));
    }

    // ── Connection closed ─────────────────────────────────────────────────────

    @Test
    void afterConnectionClosed_removesSession() throws Exception {
        putUsername("alice");

        handler.afterConnectionClosed(session, CloseStatus.NORMAL);

        verify(chatService).removeSession("alice", session);
    }

    @Test
    void afterConnectionClosed_doesNothing_whenUsernameAttributeMissing() throws Exception {
        when(session.getAttributes()).thenReturn(new HashMap<>());

        handler.afterConnectionClosed(session, CloseStatus.NORMAL);

        verify(chatService, never()).removeSession(any(), any());
    }

    // ── Transport error ───────────────────────────────────────────────────────

    @Test
    void handleTransportError_removesSession_andClosesIt() throws Exception {
        putUsername("alice");

        handler.handleTransportError(session, new RuntimeException("network error"));

        verify(chatService).removeSession("alice", session);
        verify(session).close(CloseStatus.SERVER_ERROR);
    }

    @Test
    void handleTransportError_doesNotClose_whenSessionAlreadyClosed() throws Exception {
        putUsername("alice");
        when(session.isOpen()).thenReturn(false);

        handler.handleTransportError(session, new RuntimeException("network error"));

        verify(session, never()).close(any(CloseStatus.class));
    }
}
