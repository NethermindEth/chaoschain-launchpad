package da

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func generateKey() string {
	// Generate a new ECDSA private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	// Convert to hex string
	privateKeyBytes := privateKey.D.Bytes()
	privateKeyHex := hex.EncodeToString(privateKeyBytes)
	
	return privateKeyHex
}

func main() {
	privateKeyHex := generateKey()
	fmt.Printf("EIGENDA_AUTH_PK=%s\n", privateKeyHex)
	fmt.Println("\nTo use this key, run:")
	fmt.Printf("export EIGENDA_AUTH_PK=%s\n", privateKeyHex)
}