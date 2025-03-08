package crypto

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// Sign a message using the private key
func SignMessage(privateKeyHex string, message []byte) (string, error) {
	privateKey, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", errors.New("invalid private key format")
	}
	signature := ed25519.Sign(privateKey, message)
	return hex.EncodeToString(signature), nil
}

// Verify a signed message using the public key
func VerifySignature(publicKeyHex, message, signatureHex string) bool {
	publicKey, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false
	}
	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false
	}
	return ed25519.Verify(publicKey, []byte(message), signature)
}

// HashData creates a SHA256 hash of the input data
func HashData(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
