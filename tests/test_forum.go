package forum

import (
	"testing"
)

func TestCreateAndRetrieveThread(t *testing.T) {
	// Create a test thread.
	threadID := "test-thread-id"
	title := "Test Thread"
	creator := "TestProducer"
	CreateThread(threadID, title, creator)

	// Retrieve thread and verify properties.
	thread, err := GetThread(threadID)
	if err != nil {
		t.Fatalf("Expected thread with id %s, but got error: %v", threadID, err)
	}
	if thread.Title != title || thread.Creator != creator {
		t.Errorf("Thread data mismatch, got title %s and creator %s", thread.Title, thread.Creator)
	}
}