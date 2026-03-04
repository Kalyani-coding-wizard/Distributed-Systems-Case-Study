package node

import (
	"fmt"
	"net/rpc"
	"time"
	"workdistributor/common"
	sharedrpc "workdistributor/rpc"
)

// RequestCS is called by another node asking for permission to submit work
func (n *Node) RequestCS(args *sharedrpc.RARequest, reply *sharedrpc.RAReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// 1. Update our Lamport Clock (max of ours and theirs, + 1)
	if args.Timestamp > n.LamportClock {
		n.LamportClock = args.Timestamp
	}
	n.LamportClock++

	// 2. Do we have priority? (Our timestamp is earlier, or timestamps tie and our ID is lower)
	weHavePriority := (n.LamportClock < args.Timestamp) || (n.LamportClock == args.Timestamp && n.ID < args.NodeID)

	// 3. We use the Critical Section if we are currently HELD, or if we WANT it and have priority
	weAreUsingCS := n.MEState == "HELD" || (n.MEState == "WANTED" && weHavePriority)

	if weAreUsingCS {
		// We make them wait. Add them to our deferred list.
		reply.Ok = false
		n.DeferredReplies = append(n.DeferredReplies, args.NodeID)
		fmt.Printf("[RA] Node %d deferred Node %d's CS request.\n", n.ID, args.NodeID)
	} else {
		// We don't need it, or they have priority. Let them go!
		reply.Ok = true
	}
	return nil
}

// GrantCS is called when a node finishes its Critical Section and tells us we can go now
func (n *Node) GrantCS(args *sharedrpc.RARequest, reply *sharedrpc.RAReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.ReplyCount++ // We got our delayed permission!
	reply.Ok = true
	return nil
}

// EnterCriticalSection handles broadcasting the request and waiting for all permissions
func (n *Node) EnterCriticalSection() {
	n.mu.Lock()
	n.MEState = "WANTED"
	n.LamportClock++
	myTimestamp := n.LamportClock
	n.ReplyCount = 0
	n.mu.Unlock()

	expectedReplies := 0

	// Ask all other active nodes for permission
	for id, addr := range common.NodeAddresses {
		if id != n.ID {
			client, err := rpc.Dial("tcp", addr)
			if err != nil {
				continue // If a node is offline, we don't need its permission
			}
			expectedReplies++

			args := &sharedrpc.RARequest{NodeID: n.ID, Timestamp: myTimestamp}
			reply := &sharedrpc.RAReply{}
			client.Call("Node.RequestCS", args, reply)

			// If they said yes immediately, count it
			if reply.Ok {
				n.mu.Lock()
				n.ReplyCount++
				n.mu.Unlock()
			}
			client.Close()
		}
	}

	// Wait until we get all required permissions (either immediately, or via delayed GrantCS calls)
	for {
		n.mu.Lock()
		count := n.ReplyCount
		n.mu.Unlock()

		if count >= expectedReplies {
			break // We have everyone's permission!
		}
		time.Sleep(100 * time.Millisecond) // Don't spam the CPU while waiting
	}

	n.mu.Lock()
	n.MEState = "HELD"
	n.mu.Unlock()
	fmt.Printf(">>> Node %d ENTERED Critical Section! <<<\n", n.ID)
}

// ExitCriticalSection handles releasing the lock and notifying anyone who was waiting
func (n *Node) ExitCriticalSection() {
	n.mu.Lock()
	n.MEState = "RELEASED"
	deferred := n.DeferredReplies
	n.DeferredReplies = make([]int, 0) // Clear the list for next time
	n.mu.Unlock()

	fmt.Printf("<<< Node %d EXITED Critical Section. Granting waiting nodes... >>>\n", n.ID)

	// Send a Grant message to everyone we made wait
	for _, id := range deferred {
		addr := common.NodeAddresses[id]
		client, err := rpc.Dial("tcp", addr)
		if err == nil {
			args := &sharedrpc.RARequest{NodeID: n.ID} // Just passing our ID so they know who granted it
			reply := &sharedrpc.RAReply{}
			client.Call("Node.GrantCS", args, reply)
			client.Close()
		}
	}
}
