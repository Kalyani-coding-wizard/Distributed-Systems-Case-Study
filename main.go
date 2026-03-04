package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
	"workdistributor/common"
	"workdistributor/node"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <NodeID>")
		os.Exit(1)
	}

	idStr := os.Args[1]
	nodeID, err := strconv.Atoi(idStr)
	if err != nil {
		fmt.Printf("Invalid Node ID '%s'. It must be a number.\n", idStr)
		os.Exit(1)
	}

	address, exists := common.NodeAddresses[nodeID]
	if !exists {
		fmt.Printf("Node ID %d not found in common/config.go\n", nodeID)
		os.Exit(1)
	}

	n := node.NewNode(nodeID, address)
	n.StartServer() // Every node starts a server (needed for P2P)

	// 1. Start the worker loop in the background (it will safely pause if it wins the election)
	go n.StartWorkerLoop()

	// 2. Wait just a tiny fraction of a second to ensure the RPC server is fully up
	time.Sleep(500 * time.Millisecond)

	// 3. Kick off the Bully Algorithm!
	go n.StartElection()

	// 4. Trigger Periodic Global Snapshots (Only Node 1 will initiate them)
	if n.ID == 1 {
		go func() {
			snapshotID := 1
			for {
				time.Sleep(8 * time.Second)
				n.StartSnapshot(snapshotID)
				snapshotID++
			}
		}()
	}

	// Block main thread so it stays alive
	fmt.Scanln()
}
