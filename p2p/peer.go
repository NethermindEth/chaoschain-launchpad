package p2p

import "log"

// KnownPeers stores known peer addresses
var KnownPeers = []string{
	"localhost:4001",
	"localhost:4002",
}

// DiscoverPeers attempts to connect to known peers
func (p *Node) DiscoverPeers() {
	for _, peer := range KnownPeers {
		if _, exists := p.Peers[peer]; !exists {
			p.ConnectToPeer(peer)
		}
	}
	log.Println("Peer discovery completed")
}
