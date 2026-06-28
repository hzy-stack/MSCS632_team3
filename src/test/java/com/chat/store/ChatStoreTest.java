package com.chat.store;

import com.chat.model.Message;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class ChatStoreTest {

    private ChatStore store;

    @BeforeEach
    void setUp() {
        store = new ChatStore();
    }

    @Test
    void conversationKey_isSymmetric() {
        assertEquals(
            ChatStore.conversationKey("alice", "bob"),
            ChatStore.conversationKey("bob", "alice")
        );
    }

    @Test
    void conversationKey_alphabeticallyFirstComes_first() {
        assertEquals("alice:bob", ChatStore.conversationKey("alice", "bob"));
        assertEquals("alice:bob", ChatStore.conversationKey("bob", "alice"));
    }

    @Test
    void conversationKey_isCaseSensitive() {
        assertNotEquals(
            ChatStore.conversationKey("alice", "bob"),
            ChatStore.conversationKey("Alice", "bob")
        );
    }

    @Test
    void getOrCreateConversation_returnsEmptyListForNewKey() {
        List<Message> list = store.getOrCreateConversation("alice:bob");
        assertNotNull(list);
        assertTrue(list.isEmpty());
    }

    @Test
    void getOrCreateConversation_returnsSameListForSameKey() {
        List<Message> first  = store.getOrCreateConversation("alice:bob");
        List<Message> second = store.getOrCreateConversation("alice:bob");
        assertSame(first, second);
    }

    @Test
    void getOrCreateConversation_returnsDifferentListsForDifferentKeys() {
        List<Message> ab = store.getOrCreateConversation("alice:bob");
        List<Message> ac = store.getOrCreateConversation("alice:carol");
        assertNotSame(ab, ac);
    }
}
