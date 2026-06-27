package com.chat.controller;

import com.chat.model.Message;
import com.chat.service.ChatService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.Map;

@RestController
@CrossOrigin(origins = "*")
public class ChatController {

    private final ChatService chatService;

    public ChatController(ChatService chatService) {
        this.chatService = chatService;
    }

    // ── Auth ─────────────────────────────────────────────────────────────────

    @PostMapping("/login")
    public ResponseEntity<Map<String, String>> login(@RequestBody Map<String, String> body) {
        String username = body.get("username");
        if (username == null || username.isBlank()) {
            return ResponseEntity.badRequest().body(Map.of("error", "Username is required"));
        }
        String trimmed = username.trim();
        if (trimmed.length() < 3 || trimmed.length() > 20) {
            return ResponseEntity.badRequest().body(Map.of("error", "Username must be between 3 and 20 characters"));
        }
        if (!trimmed.matches("[a-zA-Z0-9]+")) {
            return ResponseEntity.badRequest().body(Map.of("error", "Username may only contain letters and numbers"));
        }
        chatService.login(trimmed);
        return ResponseEntity.ok(Map.of("username", trimmed));
    }

    @DeleteMapping("/logout/{username}")
    public ResponseEntity<Void> logout(@PathVariable String username) {
        chatService.logout(username);
        return ResponseEntity.noContent().build();
    }

    // ── History ──────────────────────────────────────────────────────────────

    @GetMapping("/history/{u1}/{u2}")
    public ResponseEntity<List<Message>> history(@PathVariable String u1, @PathVariable String u2) {
        return ResponseEntity.ok(chatService.getHistory(u1, u2));
    }

    // ── Search ───────────────────────────────────────────────────────────────

    @GetMapping("/search")
    public ResponseEntity<?> search(@RequestParam String q) {
        if (q == null || q.isBlank()) {
            return ResponseEntity.badRequest().body(Map.of("error", "Search query is required"));
        }
        return ResponseEntity.ok(chatService.search(q.trim()));
    }

    // ── Users ────────────────────────────────────────────────────────────────

    @GetMapping("/users")
    public ResponseEntity<List<String>> onlineUsers() {
        return ResponseEntity.ok(chatService.getOnlineUsers());
    }
}
