package node

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
)

// StartServer registers the Node for RPC and starts listening on its port
func (n *Node) StartServer() {
	// 1. Register the Node object so its methods can be called remotely
	err := rpc.Register(n)
	if err != nil {
		log.Fatalf("Failed to register RPC: %s", err)
	}

	// 2. Open a TCP listener on the node's specific address/port
	listener, err := net.Listen("tcp", n.Address)
	if err != nil {
		log.Fatalf("Node %d failed to listen on %s: %s", n.ID, n.Address, err)
	}

	fmt.Printf("Node %d successfully started and listening on %s\n", n.ID, n.Address)

	// 3. Run an infinite loop in the background to accept incoming connections
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Connection accept error: %s", err)
				continue
			}
			// Handle each incoming connection in a new goroutine (concurrently)
			go rpc.ServeConn(conn)
		}
	}()
}
