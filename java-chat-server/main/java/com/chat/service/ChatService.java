package com.chat.service;

import com.chat.model.Message;
import com.chat.model.ServerResponse;
import com.chat.model.UserSession;
import com.chat.store.ChatStore;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.stereotype.Service;
import org.springframework.web.socket.TextMessage;
import org.springframework.web.socket.WebSocketSession;

import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

@Service
public class ChatService {

    private final ChatStore store;
    private final ObjectMapper objectMapper;

    public ChatService(ChatStore store, ObjectMapper objectMapper) {
        this.store = store;
        this.objectMapper = objectMapper;
    }

    // Always succeeds — if the username already exists the session is refreshed and
    // the caller rejoins with full history intact
    public void login(String username) {
        store.getUsers().put(username, new UserSession(username));
    }

    public boolean isRegistered(String username) {
        return store.getUsers().containsKey(username);
    }

    public void logout(String username) {
        store.getUsers().remove(username);
        WebSocketSession session = store.getActiveConnections().remove(username);
        if (session != null && session.isOpen()) {
            try { session.close(); } catch (IOException e) { /* session already closing */ }
        }
    }

    // Called by the WebSocket handler when a connection opens
    public void registerSession(String username, WebSocketSession session) {
        WebSocketSession existing = store.getActiveConnections().put(username, session);
        if (existing != null && existing.isOpen()) {
            try { existing.close(); } catch (IOException e) { /* ignore */ }
        }
        UserSession userSession = store.getUsers().get(username);
        if (userSession != null) {
            userSession.setConnected(true);
        }
    }

    // Called by the WebSocket handler when a connection closes
    public void removeSession(String username, WebSocketSession session) {
        // Two-argument remove only deletes the entry if the value still matches
        // this session, preventing accidental removal of a newer session that
        // replaced this one during a reconnect.
        store.getActiveConnections().remove(username, session);
        UserSession userSession = store.getUsers().get(username);
        if (userSession != null) {
            userSession.setConnected(false);
        }
    }

    public void sendMessage(String sender, String recipient, String text) throws IOException {
        Message message = new Message(sender, recipient, text);
        String key = ChatStore.conversationKey(sender, recipient);
        store.getOrCreateConversation(key).add(message);

        String json = objectMapper.writeValueAsString(ServerResponse.chat(message));
        push(store.getActiveConnections().get(recipient), json);
        // Echo back to the sender so their UI updates through the same WebSocket
        // path as the recipient, without needing a separate HTTP response.
        push(store.getActiveConnections().get(sender), json);
    }

    private void push(WebSocketSession session, String json) {
        if (session != null && session.isOpen()) {
            try {
                // WebSocketSession is not thread-safe for concurrent writes;
                // synchronizing on the session prevents interleaved frames.
                synchronized (session) {
                    session.sendMessage(new TextMessage(json));
                }
            } catch (IOException e) {
                // Sending failed because the session is closing; the
                // afterConnectionClosed callback will remove it from the store.
            }
        }
    }

    public List<Message> getHistory(String u1, String u2) {
        String key = ChatStore.conversationKey(u1, u2);
        List<Message> history = store.getMessagesByConversation().get(key);
        if (history == null) {
            return List.of();
        }
        // synchronizedList makes individual operations thread-safe but not
        // iteration; an explicit synchronized block is required here.
        synchronized (history) {
            return new ArrayList<>(history);
        }
    }

    public List<Message> search(String query) {
        String lowerQuery = query.toLowerCase();
        List<Message> results = new ArrayList<>();
        for (List<Message> messages : store.getMessagesByConversation().values()) {
            synchronized (messages) {
                for (Message m : messages) {
                    if (m.getText().toLowerCase().contains(lowerQuery)
                            || m.getSender().toLowerCase().contains(lowerQuery)) {
                        results.add(m);
                    }
                }
            }
        }
        return results;
    }

    public List<String> getOnlineUsers() {
        return new ArrayList<>(store.getActiveConnections().keySet());
    }
}
