package com.chat.model;

import com.fasterxml.jackson.annotation.JsonInclude;

import java.util.List;

@JsonInclude(JsonInclude.Include.NON_NULL)
public class ServerResponse {

    private final String type;
    private final Message message;
    private final List<Message> messages;
    private final String error;

    private ServerResponse(String type, Message message, List<Message> messages, String error) {
        this.type = type;
        this.message = message;
        this.messages = messages;
        this.error = error;
    }

    public static ServerResponse chat(Message message) {
        return new ServerResponse("chat", message, null, null);
    }

    public static ServerResponse history(List<Message> messages) {
        return new ServerResponse("history", null, messages, null);
    }

    public static ServerResponse error(String error) {
        return new ServerResponse("error", null, null, error);
    }

    public String getType() { return type; }
    public Message getMessage() { return message; }
    public List<Message> getMessages() { return messages; }
    public String getError() { return error; }
}
