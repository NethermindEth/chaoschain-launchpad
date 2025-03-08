package main

import (
	"log"

	"github.com/NethermindEth/chaoschain-launchpad/core"
)

func main() {
	// Initialize NATS before bootstrapping the rest of your services.
	core.SetupNATS("nats://localhost:4222")
	defer core.NatsBrokerInstance.Close()

	log.Println("Application started with NATS messaging enabled.")
	// ... initialize consensus manager, producers, validators, etc.
}