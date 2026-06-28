package com.chat.controller;

import com.chat.model.Message;
import com.chat.service.ChatService;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;

import java.util.List;
import java.util.Map;

import static org.mockito.Mockito.*;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@SpringBootTest
@AutoConfigureMockMvc
class ChatControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private ObjectMapper objectMapper;

    @MockBean
    private ChatService chatService;

    // ── POST /login ───────────────────────────────────────────────────────────

    @Test
    void login_returns200_withValidUsername() throws Exception {
        mockMvc.perform(post("/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(Map.of("username", "alice"))))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.username").value("alice"));

        verify(chatService).login("alice");
    }

    @Test
    void login_trimsWhitespace_andAccepts() throws Exception {
        mockMvc.perform(post("/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(Map.of("username", "  alice  "))))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.username").value("alice"));
    }

    @Test
    void login_returns400_whenUsernameBlank() throws Exception {
        mockMvc.perform(post("/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(Map.of("username", ""))))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").exists());
    }

    @Test
    void login_returns400_whenUsernameTooShort() throws Exception {
        mockMvc.perform(post("/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(Map.of("username", "ab"))))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").value("Username must be between 3 and 20 characters"));
    }

    @Test
    void login_returns400_whenUsernameTooLong() throws Exception {
        mockMvc.perform(post("/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(Map.of("username", "a".repeat(21)))))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").value("Username must be between 3 and 20 characters"));
    }

    @Test
    void login_returns400_whenUsernameHasSpecialChars() throws Exception {
        mockMvc.perform(post("/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(Map.of("username", "alice!"))))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").value("Username may only contain letters and numbers"));
    }

    @Test
    void login_returns400_whenUsernameHasSpace() throws Exception {
        mockMvc.perform(post("/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(Map.of("username", "ali ce"))))
                .andExpect(status().isBadRequest());
    }

    @Test
    void login_accepts20CharUsername() throws Exception {
        String name = "a".repeat(20);
        mockMvc.perform(post("/login")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(Map.of("username", name))))
                .andExpect(status().isOk());
    }

    // ── DELETE /logout/{username} ─────────────────────────────────────────────

    @Test
    void logout_returns204() throws Exception {
        mockMvc.perform(delete("/logout/alice"))
                .andExpect(status().isNoContent());

        verify(chatService).logout("alice");
    }

    // ── GET /history/{u1}/{u2} ────────────────────────────────────────────────

    @Test
    void history_returnsMessageList() throws Exception {
        Message msg = new Message("alice", "bob", "Hello");
        when(chatService.getHistory("alice", "bob")).thenReturn(List.of(msg));

        mockMvc.perform(get("/history/alice/bob"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[0].sender").value("alice"))
                .andExpect(jsonPath("$[0].recipient").value("bob"))
                .andExpect(jsonPath("$[0].text").value("Hello"));
    }

    @Test
    void history_returnsEmptyList_whenNoMessages() throws Exception {
        when(chatService.getHistory("alice", "bob")).thenReturn(List.of());

        mockMvc.perform(get("/history/alice/bob"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$").isEmpty());
    }

    // ── GET /search?q= ────────────────────────────────────────────────────────

    @Test
    void search_returnsResults() throws Exception {
        Message msg = new Message("alice", "bob", "Hello world");
        when(chatService.search("world")).thenReturn(List.of(msg));

        mockMvc.perform(get("/search").param("q", "world"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[0].text").value("Hello world"));
    }

    @Test
    void search_returns400_whenQueryBlank() throws Exception {
        mockMvc.perform(get("/search").param("q", "   "))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.error").exists());
    }

    // ── GET /users ────────────────────────────────────────────────────────────

    @Test
    void users_returnsOnlineList() throws Exception {
        when(chatService.getOnlineUsers()).thenReturn(List.of("alice", "bob"));

        mockMvc.perform(get("/users"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[0]").value("alice"))
                .andExpect(jsonPath("$[1]").value("bob"));
    }

    @Test
    void users_returnsEmptyList_whenNobodyOnline() throws Exception {
        when(chatService.getOnlineUsers()).thenReturn(List.of());

        mockMvc.perform(get("/users"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$").isEmpty());
    }
}
