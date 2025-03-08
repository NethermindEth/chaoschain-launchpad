package main

import (
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

func main() {
	// Connect to the NATS server
	nc, err := nats.Connect("nats://localhost:4222")
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()

	// Subscribe to the "test" subject
	sub, err := nc.SubscribeSync("test")
	if err != nil {
		log.Fatalf("Error subscribing: %v", err)
	}

	// Publish a test message
	err = nc.Publish("test", []byte("Hello, NATS!"))
	if err != nil {
		log.Fatalf("Error publishing: %v", err)
	}

	// Allow the message some time to propagate
	nc.Flush()
	time.Sleep(500 * time.Millisecond)

	// Wait for a message on the test subject
	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		log.Fatalf("Did not receive a message: %v", err)
	}

	fmt.Printf("Received test message: %s\n", string(msg.Data))
}