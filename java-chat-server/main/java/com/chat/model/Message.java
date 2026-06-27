package com.chat.model;

import java.time.LocalDateTime;
import java.util.UUID;

public class Message {

    private final String messageId;
    private final String sender;
    private final String recipient;
    private final String text;
    private final LocalDateTime timestamp;

    public Message(String sender, String recipient, String text) {
        this.messageId = UUID.randomUUID().toString();
        this.sender = sender;
        this.recipient = recipient;
        this.text = text;
        this.timestamp = LocalDateTime.now();
    }

    public String getMessageId() { return messageId; }
    public String getSender() { return sender; }
    public String getRecipient() { return recipient; }
    public String getText() { return text; }
    public LocalDateTime getTimestamp() { return timestamp; }
}
