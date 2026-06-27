package com.chat.model;

import java.time.LocalDateTime;

public class UserSession {

    private final String username;
    private final LocalDateTime connectedAt;
    private boolean connected;

    public UserSession(String username) {
        this.username = username;
        this.connectedAt = LocalDateTime.now();
        this.connected = false;
    }

    public String getUsername() { return username; }
    public LocalDateTime getConnectedAt() { return connectedAt; }
    public boolean isConnected() { return connected; }
    public void setConnected(boolean connected) { this.connected = connected; }
}
