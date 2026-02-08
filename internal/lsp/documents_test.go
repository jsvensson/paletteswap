package lsp

import (
	"testing"
)

// TestDocumentStore_Update verifies that the document store properly updates content
func TestDocumentStore_Update(t *testing.T) {
	store := NewDocumentStore()

	// Open a document
	store.Open("test://file.pstheme", "initial content")

	content, ok := store.Get("test://file.pstheme")
	if !ok {
		t.Fatal("Document not found after opening")
	}
	if content != "initial content" {
		t.Errorf("Expected 'initial content', got '%s'", content)
	}

	// Update the document
	store.Update("test://file.pstheme", "updated content")

	content, ok = store.Get("test://file.pstheme")
	if !ok {
		t.Fatal("Document not found after update")
	}
	if content != "updated content" {
		t.Errorf("Expected 'updated content', got '%s'", content)
	}
}

// TestDocumentStore_MultipleUpdates verifies multiple updates work correctly
func TestDocumentStore_MultipleUpdates(t *testing.T) {
	store := NewDocumentStore()
	store.Open("test://file.pstheme", "version 1")

	updates := []string{
		"version 2",
		"version 3",
		"version 4",
	}

	for i, update := range updates {
		store.Update("test://file.pstheme", update)
		content, ok := store.Get("test://file.pstheme")
		if !ok {
			t.Fatalf("Document not found after update %d", i+2)
		}
		if content != update {
			t.Errorf("Update %d: expected '%s', got '%s'", i+2, update, content)
		}
	}
}

// TestDocumentStore_ConcurrentAccess verifies thread safety
func TestDocumentStore_ConcurrentAccess(t *testing.T) {
	store := NewDocumentStore()
	store.Open("test://file.pstheme", "initial")

	// Run concurrent updates
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(n int) {
			store.Update("test://file.pstheme", string(rune('0'+n)))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify document still exists and has some content
	content, ok := store.Get("test://file.pstheme")
	if !ok {
		t.Error("Document not found after concurrent updates")
	}
	if content == "" {
		t.Error("Document content is empty after concurrent updates")
	}
}
