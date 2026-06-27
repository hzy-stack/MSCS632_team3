package com.chat.store;

import com.chat.model.Message;
import com.chat.model.UserSession;
import org.springframework.stereotype.Component;
import org.springframework.web.socket.WebSocketSession;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.concurrent.ConcurrentHashMap;

@Component
public class ChatStore {

    // All users who have logged in during this server run
    private final ConcurrentHashMap<String, UserSession> users = new ConcurrentHashMap<>();

    // Maps username -> active WebSocket session (only present while connection is open)
    private final ConcurrentHashMap<String, WebSocketSession> activeConnections = new ConcurrentHashMap<>();

    // Maps conversation key (e.g. "alice:bob") -> list of messages
    private final ConcurrentHashMap<String, List<Message>> messagesByConversation = new ConcurrentHashMap<>();

    // Returns a consistent key regardless of argument order
    public static String conversationKey(String u1, String u2) {
        return u1.compareTo(u2) <= 0 ? u1 + ":" + u2 : u2 + ":" + u1;
    }

    public ConcurrentHashMap<String, UserSession> getUsers() {
        return users;
    }

    public ConcurrentHashMap<String, WebSocketSession> getActiveConnections() {
        return activeConnections;
    }

    public List<Message> getOrCreateConversation(String key) {
        return messagesByConversation.computeIfAbsent(
                key,
                k -> Collections.synchronizedList(new ArrayList<>())
        );
    }

    public ConcurrentHashMap<String, List<Message>> getMessagesByConversation() {
        return messagesByConversation;
    }
}
