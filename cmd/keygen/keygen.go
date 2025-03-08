package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Generate a new Ed25519 keypair
func generateKeyPair() (string, string) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic("Failed to generate key pair")
	}
	return hex.EncodeToString(pub), hex.EncodeToString(priv)
}

func main() {
	pub, priv := generateKeyPair()
	fmt.Println("Public Key:", pub)
	fmt.Println("Private Key:", priv)
}
