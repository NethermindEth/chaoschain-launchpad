package messaging

import (
	"fmt"
	"github.com/nats-io/nats.go"
)

// Messenger encapsulates a NATS connection.
type Messenger struct {
	NC *nats.Conn
}

// NewMessenger creates a new instance of Messenger.
func NewMessenger(url string) (*Messenger, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &Messenger{NC: nc}, nil
}

// PublishGlobal publishes a message to a global subject (for public announcements).
func (m *Messenger) PublishGlobal(subject, message string) error {
	return m.NC.Publish(subject, []byte(message))
}

// PublishPrivate sends a message directly to a specific agent by using a private subject.
func (m *Messenger) PublishPrivate(agentID, message string) error {
	subject := fmt.Sprintf("agent.%s.private", agentID)
	return m.NC.Publish(subject, []byte(message))
}

// SubscribeGlobal subscribes to a global topic.
func (m *Messenger) SubscribeGlobal(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	return m.NC.Subscribe(subject, handler)
}

// SubscribePrivate subscribes to private messages for an agent.
func (m *Messenger) SubscribePrivate(agentID string, handler nats.MsgHandler) (*nats.Subscription, error) {
	subject := fmt.Sprintf("agent.%s.private", agentID)
	return m.NC.Subscribe(subject, handler)
}