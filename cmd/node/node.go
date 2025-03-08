package main

import (
	"log"
	"time"

	"github.com/NethermindEth/chaoschain-launchpad/mempool"
)

func main() {
	mp := mempool.GetMempool() // Transactions expire in 60 seconds

	// Run cleanup every 10 seconds
	go func() {
		for {
			time.Sleep(10 * time.Second)
			mp.CleanupExpiredTransactions()
			log.Println("Expired transactions cleaned")
		}
	}()
}
