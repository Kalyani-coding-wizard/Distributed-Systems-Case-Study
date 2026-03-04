package node

import (
	"fmt"
	"net/rpc"
	"workdistributor/common"
	sharedrpc "workdistributor/rpc"
)

// Election is called by a lower-ID node.
// If our ID is higher, we reply "Ok" to make them stop, and then we start our own election.
func (n *Node) Election(args *sharedrpc.ElectionMsg, reply *sharedrpc.ElectionReply) error {
	if n.ID > args.SenderID {
		reply.Ok = true
		fmt.Printf("Received Election msg from Node %d. I am higher (Node %d), telling them to stop.\n", args.SenderID, n.ID)

		// Start our own election in the background
		go n.StartElection()
	} else {
		reply.Ok = false
	}
	return nil
}

// Coordinator is called by the winning node to tell everyone it is the new leader
func (n *Node) Coordinator(args *sharedrpc.CoordinatorMsg, reply *sharedrpc.CoordinatorReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.LeaderID = args.LeaderID

	// If the new leader is us, we set IsLeader to true (though we usually know this already if we won)
	n.IsLeader = (n.ID == args.LeaderID)

	reply.Ok = true
	fmt.Printf("Node %d acknowledges new Leader: Node %d\n", n.ID, n.LeaderID)
	return nil
}

// StartElection initiates the Bully Algorithm
func (n *Node) StartElection() {
	fmt.Printf("\n--- Node %d is starting an election! ---\n", n.ID)

	gotHigherNode := false

	// Phase 1: Challenge higher nodes
	for id, addr := range common.NodeAddresses {
		if id > n.ID {
			// Try to contact the higher-ID node
			client, err := rpc.Dial("tcp", addr)
			if err != nil {
				// Node is offline, ignore it
				continue
			}

			args := &sharedrpc.ElectionMsg{SenderID: n.ID}
			reply := &sharedrpc.ElectionReply{}

			// Call the Election method on the higher node
			err = client.Call("Node.Election", args, reply)
			if err == nil && reply.Ok {
				fmt.Printf("Node %d replied to my election. I am stepping down.\n", id)
				gotHigherNode = true
			}
			client.Close()
		}
	}

	// Phase 2: Declare Victory if no higher nodes replied
	if !gotHigherNode {
		fmt.Printf("*** Node %d WON THE ELECTION! I am the new Leader. ***\n", n.ID)

		n.mu.Lock()
		n.IsLeader = true
		n.LeaderID = n.ID
		n.mu.Unlock()

		// If we just became leader, we should generate some jobs to distribute!
		n.GenerateDummyJobs(5)

		// Bully everyone else into accepting us
		for id, addr := range common.NodeAddresses {
			if id != n.ID {
				client, err := rpc.Dial("tcp", addr)
				if err != nil {
					continue // Node offline
				}

				args := &sharedrpc.CoordinatorMsg{LeaderID: n.ID}
				reply := &sharedrpc.CoordinatorReply{}
				client.Call("Node.Coordinator", args, reply)
				client.Close()
			}
		}
	} else {
		// We stepped down. Wait a bit for the winner to send a Coordinator message.
		// If we don't hear from a new leader soon, we will restart the election (we will add this loop next).
		fmt.Printf("Node %d is waiting for the new leader to announce itself...\n", n.ID)
	}
}
