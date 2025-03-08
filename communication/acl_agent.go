package acl

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"net"
	"time"
)

// FIPAMessage defines a standard FIPA ACL message.
type FIPAMessage struct {
	MessageID      string `json:"message_id"`      // Unique identifier for this message.
	Performative   string `json:"performative"`    // e.g., "REQUEST", "INFORM", "PROPOSE", etc.
	Sender         string `json:"sender"`          // Agent sending the message.
	Receiver       string `json:"receiver"`        // Intended recipient.
	Content        string `json:"content"`         // The payload (could be a proposal, vote, etc.).
	Language       string `json:"language"`        // e.g., "JSON"
	Ontology       string `json:"ontology"`        // Domain of the message.
	ConversationID string `json:"conversation_id"` // To correlate interactive dialogues.
	ReplyWith      string `json:"reply_with"`      // For replies.
	InReplyTo      string `json:"in_reply_to"`     // Optional.
	Protocol       string `json:"protocol"`        // e.g., "FIPA-ACL"
}

// NewFIPAMessage creates a new FIPA ACL message with a unique MessageID.
func NewFIPAMessage(performative, sender, receiver, content, ontology, conversationID, protocol string) *FIPAMessage {
	return &FIPAMessage{
		MessageID:      uuid.New().String(),
		Performative:   performative,
		Sender:         sender,
		Receiver:       receiver,
		Content:        content,
		Language:       "JSON",
		Ontology:       ontology,
		ConversationID: conversationID,
		Protocol:       protocol,
		ReplyWith:      "",
		InReplyTo:      "",
	}
}

// Serialize converts the FIPAMessage into a JSON string.
func (msg *FIPAMessage) Serialize() (string, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SendFIPAMessage sends the message via TCP to the receiver’s address.
func SendFIPAMessage(receiverAddr string, msg *FIPAMessage) error {
	conn, err := net.DialTimeout("tcp", receiverAddr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	data, err := msg.Serialize()
	if err != nil {
		return err
	}

	// Append a newline to indicate end-of-message.
	_, err = conn.Write([]byte(data + "\n"))
	return err
}

// ListenFIPAMessages starts a simple TCP listener that prints incoming FIPA messages.
func ListenFIPAMessages(port int) error {
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil && err != io.EOF {
		return
	}
	var msg FIPAMessage
	if err := json.Unmarshal(buf[:n], &msg); err != nil {
		fmt.Println("Error unmarshalling message:", err)
		return
	}
	fmt.Printf("Received FIPA Message: %+v\n", msg)
}