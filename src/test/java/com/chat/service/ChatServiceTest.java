package com.chat.service;

import com.chat.model.Message;
import com.chat.store.ChatStore;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import com.fasterxml.jackson.datatype.jsr310.JavaTimeModule;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

class ChatServiceTest {

    private ChatStore store;
    private ChatService service;

    @BeforeEach
    void setUp() {
        store = new ChatStore();
        ObjectMapper objectMapper = new ObjectMapper()
                .registerModule(new JavaTimeModule())
                .disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS);
        service = new ChatService(store, objectMapper);
    }

    // ── Login / registration ──────────────────────────────────────────────────

    @Test
    void login_registersUser() {
        service.login("alice");
        assertTrue(service.isRegistered("alice"));
    }

    @Test
    void login_allowsRejoin_whenUsernameAlreadyExists() {
        service.login("alice");
        assertDoesNotThrow(() -> service.login("alice"));
        assertTrue(service.isRegistered("alice"));
    }

    @Test
    void isRegistered_falseForUnknownUser() {
        assertFalse(service.isRegistered("nobody"));
    }

    // ── Logout ────────────────────────────────────────────────────────────────

    @Test
    void logout_removesUser() {
        service.login("alice");
        service.logout("alice");
        assertFalse(service.isRegistered("alice"));
    }

    @Test
    void logout_closesActiveWebSocketSession() throws Exception {
        WebSocketSession session = mock(WebSocketSession.class);
        when(session.isOpen()).thenReturn(true);
        service.login("alice");
        service.registerSession("alice", session);
        service.logout("alice");
        verify(session).close();
    }

    @Test
    void logout_doesNotThrow_whenNoActiveSession() {
        service.login("alice");
        assertDoesNotThrow(() -> service.logout("alice"));
    }

    // ── Messaging ─────────────────────────────────────────────────────────────

    @Test
    void sendMessage_storesMessageInConversation() throws Exception {
        service.sendMessage("alice", "bob", "Hello");
        List<Message> history = service.getHistory("alice", "bob");
        assertEquals(1, history.size());
        assertEquals("alice", history.get(0).getSender());
        assertEquals("bob",   history.get(0).getRecipient());
        assertEquals("Hello", history.get(0).getText());
    }

    @Test
    void sendMessage_pushesToRecipient() throws Exception {
        WebSocketSession recipientSession = mock(WebSocketSession.class);
        when(recipientSession.isOpen()).thenReturn(true);
        service.login("bob");
        service.registerSession("bob", recipientSession);
        service.sendMessage("alice", "bob", "Hello");
        verify(recipientSession).sendMessage(any(TextMessage.class));
    }

    @Test
    void sendMessage_echoesToSender() throws Exception {
        WebSocketSession senderSession = mock(WebSocketSession.class);
        when(senderSession.isOpen()).thenReturn(true);
        service.login("alice");
        service.registerSession("alice", senderSession);
        service.sendMessage("alice", "bob", "Hello");
        verify(senderSession).sendMessage(any(TextMessage.class));
    }

    @Test
    void sendMessage_doesNotThrow_whenRecipientIsOffline() {
        assertDoesNotThrow(() -> service.sendMessage("alice", "bob", "Hello"));
    }

    // ── History ───────────────────────────────────────────────────────────────

    @Test
    void getHistory_returnsEmptyList_whenNoMessages() {
        assertTrue(service.getHistory("alice", "bob").isEmpty());
    }

    @Test
    void getHistory_isSymmetric() throws Exception {
        service.sendMessage("alice", "bob", "Hello");
        List<Message> ab = service.getHistory("alice", "bob");
        List<Message> ba = service.getHistory("bob", "alice");
        assertEquals(ab.size(), ba.size());
        assertEquals(ab.get(0).getMessageId(), ba.get(0).getMessageId());
    }

    @Test
    void getHistory_returnsMessagesInOrder() throws Exception {
        service.sendMessage("alice", "bob", "First");
        service.sendMessage("alice", "bob", "Second");
        List<Message> history = service.getHistory("alice", "bob");
        assertEquals(2, history.size());
        assertEquals("First",  history.get(0).getText());
        assertEquals("Second", history.get(1).getText());
    }

    // ── Search ────────────────────────────────────────────────────────────────

    @Test
    void search_findsByMessageText() throws Exception {
        service.sendMessage("alice", "bob", "Hello world");
        assertEquals(1, service.search("world").size());
    }

    @Test
    void search_findsBySenderName() throws Exception {
        service.sendMessage("alice", "bob", "Hello");
        assertEquals(1, service.search("alice").size());
    }

    @Test
    void search_isCaseInsensitive() throws Exception {
        service.sendMessage("alice", "bob", "Hello World");
        assertEquals(1, service.search("hello").size());
        assertEquals(1, service.search("WORLD").size());
        assertEquals(1, service.search("Alice").size());
    }

    @Test
    void search_returnsEmpty_whenNoMatch() throws Exception {
        service.sendMessage("alice", "bob", "Hello");
        assertTrue(service.search("xyz").isEmpty());
    }

    @Test
    void search_acrossMultipleConversations() throws Exception {
        service.sendMessage("alice", "bob",   "Hello");
        service.sendMessage("carol", "dave",  "Hello there");
        assertEquals(2, service.search("hello").size());
    }

    // ── Online users ──────────────────────────────────────────────────────────

    @Test
    void getOnlineUsers_includesUserWithActiveSession() throws Exception {
        WebSocketSession session = mock(WebSocketSession.class);
        when(session.isOpen()).thenReturn(true);
        service.login("alice");
        service.registerSession("alice", session);
        assertTrue(service.getOnlineUsers().contains("alice"));
    }

    @Test
    void getOnlineUsers_excludesUserAfterSessionRemoved() throws Exception {
        WebSocketSession session = mock(WebSocketSession.class);
        when(session.isOpen()).thenReturn(true);
        service.login("alice");
        service.registerSession("alice", session);
        service.removeSession("alice", session);
        assertFalse(service.getOnlineUsers().contains("alice"));
    }

    // ── Session management ────────────────────────────────────────────────────

    @Test
    void removeSession_doesNotRemoveNewerSession_onReconnect() throws Exception {
        WebSocketSession oldSession = mock(WebSocketSession.class);
        WebSocketSession newSession = mock(WebSocketSession.class);
        when(oldSession.isOpen()).thenReturn(false);
        when(newSession.isOpen()).thenReturn(true);
        service.login("alice");
        service.registerSession("alice", oldSession);
        service.registerSession("alice", newSession);
        service.removeSession("alice", oldSession); // stale close — must not evict newSession
        assertTrue(service.getOnlineUsers().contains("alice"));
    }
}
