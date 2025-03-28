package communication

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type WSEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

const (
	EventBlockVerdict        = "BLOCK_VERDICT"
	EventAgentVote           = "AGENT_VOTE"
	EventVotingResult        = "VOTING_RESULT"
	EventAgentAlliance       = "AGENT_ALLIANCE"
	EventAgentRegistered     = "AGENT_REGISTERED"
	EventNewTransaction      = "NEW_TRANSACTION"
	EventChainCreated        = "CHAIN_CREATED"
	EventTaskBreakdown       = "TASK_BREAKDOWN"
	EventTaskBreakdownFinal  = "TASK_BREAKDOWN_FINAL"
	EventTaskDelegation      = "TASK_DELEGATION"
	EventTaskDelegationFinal = "TASK_DELEGATION_FINAL"
	EventTaskAssignment      = "TASK_ASSIGNMENT"
)

type WebSocketManager struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan WSEvent
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mu         sync.RWMutex
}

var (
	wsManager *WebSocketManager
	once      sync.Once
)

func GetWSManager() *WebSocketManager {
	once.Do(func() {
		wsManager = &WebSocketManager{
			clients:    make(map[*websocket.Conn]bool),
			broadcast:  make(chan WSEvent),
			register:   make(chan *websocket.Conn),
			unregister: make(chan *websocket.Conn),
		}
		go wsManager.run()
	})
	return wsManager
}

func (manager *WebSocketManager) run() {
	for {
		select {
		case client := <-manager.register:
			manager.mu.Lock()
			manager.clients[client] = true
			manager.mu.Unlock()

		case client := <-manager.unregister:
			manager.mu.Lock()
			if _, ok := manager.clients[client]; ok {
				delete(manager.clients, client)
				client.Close()
			}
			manager.mu.Unlock()

		case event := <-manager.broadcast:
			manager.mu.RLock()
			for client := range manager.clients {
				if err := client.WriteJSON(event); err != nil {
					log.Printf("WebSocket error: %v", err)
					client.Close()
					delete(manager.clients, client)
				}
			}
			manager.mu.RUnlock()
		}
	}
}

func BroadcastEvent(eventType string, payload interface{}) {
	event := WSEvent{
		Type:    eventType,
		Payload: payload,
	}
	GetWSManager().broadcast <- event
}

func (w *WebSocketManager) Register() chan<- *websocket.Conn {
	return w.register
}

func (w *WebSocketManager) Unregister() chan<- *websocket.Conn {
	return w.unregister
}
