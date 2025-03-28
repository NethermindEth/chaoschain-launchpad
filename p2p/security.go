package p2p

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"os"
	"path/filepath"
)

// KeyPair represents a public/private key pair for signing and verification
type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  *ecdsa.PublicKey
}

// MessageSignature contains r and s values of an ECDSA signature
type MessageSignature struct {
	R *big.Int
	S *big.Int
}

// SecurityProvider handles cryptographic operations for a Node
type SecurityProvider struct {
	keyPair         *KeyPair
	knownPublicKeys map[string]*ecdsa.PublicKey // AgentID -> PublicKey
}

// NewSecurityProvider creates a new security provider
func NewSecurityProvider() *SecurityProvider {
	return &SecurityProvider{
		knownPublicKeys: make(map[string]*ecdsa.PublicKey),
	}
}

// GenerateKeyPair creates a new ECDSA key pair
func (sp *SecurityProvider) GenerateKeyPair() error {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	sp.keyPair = &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}

	log.Println("Generated new ECDSA key pair")
	return nil
}

// LoadOrCreateKeyPair loads a key pair from disk or creates a new one
func (sp *SecurityProvider) LoadOrCreateKeyPair(keyDir string, agentID string) error {
	keyPath := filepath.Join(keyDir, "agent_"+agentID+".key")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(keyDir, 0755); err != nil {
		return err
	}

	// Try to load existing key
	if fileExists(keyPath) {
		return sp.LoadKeyPair(keyPath)
	}

	// Generate new key pair
	if err := sp.GenerateKeyPair(); err != nil {
		return err
	}

	// Save the new key pair
	return sp.SaveKeyPair(keyPath)
}

// LoadKeyPair loads a key pair from a file
func (sp *SecurityProvider) LoadKeyPair(keyPath string) error {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}

	privateKey, err := x509.ParseECPrivateKey(data)
	if err != nil {
		return err
	}

	sp.keyPair = &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}

	log.Printf("Loaded ECDSA key pair from %s", keyPath)
	return nil
}

// SaveKeyPair saves a key pair to a file
func (sp *SecurityProvider) SaveKeyPair(keyPath string) error {
	if sp.keyPair == nil {
		return errors.New("no key pair to save")
	}

	data, err := x509.MarshalECPrivateKey(sp.keyPair.PrivateKey)
	if err != nil {
		return err
	}

	return os.WriteFile(keyPath, data, 0600)
}

// RegisterPublicKey associates a public key with an agent ID
func (sp *SecurityProvider) RegisterPublicKey(agentID string, publicKey *ecdsa.PublicKey) {
	sp.knownPublicKeys[agentID] = publicKey
	log.Printf("Registered public key for agent %s", agentID)
}

// SignMessage signs a message using the node's private key
func (sp *SecurityProvider) SignMessage(msg *Message) error {
	if sp.keyPair == nil {
		return errors.New("no key pair available for signing")
	}

	// Create a copy of the message without the signature field
	messageCopy := *msg
	messageCopy.Signature = nil

	// Marshal the message to JSON
	data, err := json.Marshal(messageCopy)
	if err != nil {
		return err
	}

	// Hash the message
	hash := sha256.Sum256(data)

	// Sign the hash
	r, s, err := ecdsa.Sign(rand.Reader, sp.keyPair.PrivateKey, hash[:])
	if err != nil {
		return err
	}

	// Encode signature
	signature := MessageSignature{R: r, S: s}
	signatureBytes, err := json.Marshal(signature)
	if err != nil {
		return err
	}

	// Set the signature
	msg.Signature = signatureBytes
	return nil
}

// VerifyMessageSignature verifies a message signature
func (sp *SecurityProvider) VerifyMessageSignature(msg Message) (bool, error) {
	// If no signature, it can't be verified
	if msg.Signature == nil || len(msg.Signature) == 0 {
		return false, errors.New("message has no signature")
	}

	// If no sender ID, it can't be verified
	if msg.SenderID == "" {
		return false, errors.New("message has no sender ID")
	}

	// Get the public key for this sender
	publicKey, exists := sp.knownPublicKeys[string(msg.SenderID)]
	if !exists {
		return false, errors.New("unknown sender, public key not registered")
	}

	// Parse the signature
	var signature MessageSignature
	if err := json.Unmarshal(msg.Signature, &signature); err != nil {
		return false, err
	}

	// Create a copy of the message without the signature
	messageCopy := msg
	messageCopy.Signature = nil

	// Marshal the message to JSON
	data, err := json.Marshal(messageCopy)
	if err != nil {
		return false, err
	}

	// Hash the message
	hash := sha256.Sum256(data)

	// Verify the signature
	return ecdsa.Verify(publicKey, hash[:], signature.R, signature.S), nil
}

// ExportPublicKey exports the node's public key as a string
func (sp *SecurityProvider) ExportPublicKey() (string, error) {
	if sp.keyPair == nil {
		return "", errors.New("no key pair available")
	}

	data, err := x509.MarshalPKIXPublicKey(sp.keyPair.PublicKey)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

// ImportPublicKey imports a public key from a string
func (sp *SecurityProvider) ImportPublicKey(encodedKey string) (*ecdsa.PublicKey, error) {
	data, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, err
	}

	pub, err := x509.ParsePKIXPublicKey(data)
	if err != nil {
		return nil, err
	}

	switch pub := pub.(type) {
	case *ecdsa.PublicKey:
		return pub, nil
	default:
		return nil, errors.New("unknown public key type")
	}
}

// Helper function to check if a file exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
