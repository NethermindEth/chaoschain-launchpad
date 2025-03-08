package core

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// NATSBroker encapsulates a NATS connection.
type NATSBroker struct {
	Conn *nats.Conn
}

// NewNATSBroker creates a new NATSBroker connected to the provided URL.
func NewNATSBroker(url string) (*NATSBroker, error) {
	nc, err := nats.Connect(url,
		nats.Timeout(10*time.Second),
	)
	if err != nil {
		return nil, err
	}
	return &NATSBroker{Conn: nc}, nil
}

// Publish sends data on the provided subject.
func (b *NATSBroker) Publish(subject string, data []byte) error {
	log.Printf("Sending data to %s", subject)
	return b.Conn.Publish(subject, data)
}

// Subscribe registers a callback for a specific subject.
func (b *NATSBroker) Subscribe(subject string, cb nats.MsgHandler) error {
	_, err := b.Conn.Subscribe(subject, cb)
	return err
}

// Close gracefully closes the connection.
func (b *NATSBroker) Close() {
	b.Conn.Close()
}

// Global instance of the NATS broker.
var NatsBrokerInstance *NATSBroker

// SetupNATS initializes the global NATS broker. Call this function at startup.
func SetupNATS(url string) {
	broker, err := NewNATSBroker(url)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	NatsBrokerInstance = broker
	log.Printf("Connected to NATS at %s", url)
}