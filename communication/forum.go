package communication

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ForumMessage represents a single message in a forum thread.
type ForumMessage struct {
	ID        string    `json:"id"`
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ForumThread represents a discussion thread for a block proposal.
type ForumThread struct {
	ThreadID  string         `json:"thread_id"` // e.g., the block hash
	Title     string         `json:"title"`
	Creator   string         `json:"creator"`
	CreatedAt time.Time      `json:"created_at"`
	Messages  []ForumMessage `json:"messages"`
}

var (
	threads   = make(map[string]*ForumThread)
	threadsMu sync.Mutex
)

// CreateThread creates a new discussion thread and stores it.
func CreateThread(threadID, title, creator string) *ForumThread {
	threadsMu.Lock()
	defer threadsMu.Unlock()

	thread := &ForumThread{
		ThreadID:  threadID,
		Title:     title,
		Creator:   creator,
		CreatedAt: time.Now(),
		Messages:  []ForumMessage{},
	}
	threads[threadID] = thread
	return thread
}

// AddReply appends a reply message to an existing thread.
func AddReply(threadID, sender, content string) error {
	threadsMu.Lock()
	defer threadsMu.Unlock()

	thread, exists := threads[threadID]
	if !exists {
		return fmt.Errorf("thread with id %s does not exist", threadID)
	}

	reply := ForumMessage{
		ID:        uuid.New().String(),
		Sender:    sender,
		Content:   content,
		Timestamp: time.Now(),
	}
	thread.Messages = append(thread.Messages, reply)
	return nil
}

// GetThread retrieves a thread by its ID.
func GetThread(threadID string) (*ForumThread, error) {
	threadsMu.Lock()
	defer threadsMu.Unlock()

	thread, exists := threads[threadID]
	if !exists {
		return nil, fmt.Errorf("thread with id %s not found", threadID)
	}
	return thread, nil
}

// GetAllThreads returns a slice of all forum threads.
func GetAllThreads() []*ForumThread {
	threadsMu.Lock()
	defer threadsMu.Unlock()

	var threadsList []*ForumThread
	for _, thread := range threads {
		threadsList = append(threadsList, thread)
	}
	return threadsList
}
