package node

import (
	"fmt"
	"net/rpc"
	"workdistributor/common"
	sharedrpc "workdistributor/rpc"
)

// recordLocalState takes a "picture" of the node's current variables
func (n *Node) recordLocalState() sharedrpc.NodeLocalState {
	n.mu.Lock()
	defer n.mu.Unlock()

	status := "Idle"
	if n.CurrentJobProcessing != 0 {
		status = fmt.Sprintf("Processing Job %d", n.CurrentJobProcessing)
	}

	return sharedrpc.NodeLocalState{
		NodeID:          n.ID,
		IsLeader:        n.IsLeader,
		PendingCount:    len(n.PendingJobs),
		InProgressCount: len(n.InProgressJobs),
		CompletedCount:  len(n.CompletedJobs),
		WorkerStatus:    status,
	}
}

// StartSnapshot is called by the Initiator to kick off the Chandy-Lamport algorithm
func (n *Node) StartSnapshot(snapshotID int) {
	n.mu.Lock()
	n.SeenSnapshots[snapshotID] = true
	n.mu.Unlock()

	fmt.Printf("\n[SNAPSHOT] Node %d is initiating Global Snapshot #%d...\n", n.ID, snapshotID)

	// 1. Record our own local state
	myState := n.recordLocalState()

	n.mu.Lock()
	n.GlobalSnapshot[n.ID] = myState
	n.mu.Unlock()

	// 2. Broadcast the Marker to everyone else
	for id, addr := range common.NodeAddresses {
		if id != n.ID {
			client, err := rpc.Dial("tcp", addr)
			if err == nil {
				args := &sharedrpc.MarkerMsg{InitiatorID: n.ID, SnapshotID: snapshotID}
				reply := &sharedrpc.MarkerReply{}
				client.Call("Node.ReceiveMarker", args, reply)
				client.Close()
			}
		}
	}
}

// ReceiveMarker is an RPC method called when another node sends us a Marker
func (n *Node) ReceiveMarker(args *sharedrpc.MarkerMsg, reply *sharedrpc.MarkerReply) error {
	n.mu.Lock()
	alreadySeen := n.SeenSnapshots[args.SnapshotID]
	if !alreadySeen {
		n.SeenSnapshots[args.SnapshotID] = true
	}
	n.mu.Unlock()

	// If we've already recorded our state for this snapshot, do nothing
	if alreadySeen {
		reply.Ok = true
		return nil
	}

	// 1. Record our local state the MOMENT we see the marker for the first time
	myState := n.recordLocalState()
	fmt.Printf("[SNAPSHOT] Node %d recorded its local state for Snapshot #%d.\n", n.ID, args.SnapshotID)

	// 2. Send our recorded state back to the Initiator
	initiatorAddr := common.NodeAddresses[args.InitiatorID]
	client, err := rpc.Dial("tcp", initiatorAddr)
	if err == nil {
		submitArgs := &sharedrpc.StateSubmission{SnapshotID: args.SnapshotID, State: myState}
		submitReply := &sharedrpc.StateSubmissionReply{}
		client.Call("Node.SubmitState", submitArgs, submitReply)
		client.Close()
	}

	reply.Ok = true
	return nil
}

// SubmitState is an RPC method called on the Initiator to collect everyone's states
func (n *Node) SubmitState(args *sharedrpc.StateSubmission, reply *sharedrpc.StateSubmissionReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Save the received state into our global collection
	n.GlobalSnapshot[args.State.NodeID] = args.State

	// Print the collected state so it is visible and verifiable
	fmt.Printf("\n=== GLOBAL SNAPSHOT #%d UPDATE ===\n", args.SnapshotID)
	for nodeID, state := range n.GlobalSnapshot {
		role := "Worker"
		if state.IsLeader {
			role = "Leader"
		}
		fmt.Printf("Node %d (%s) - Pending: %d, InProgress: %d, Completed: %d, Status: %s\n",
			nodeID, role, state.PendingCount, state.InProgressCount, state.CompletedCount, state.WorkerStatus)
	}
	fmt.Printf("==================================\n\n")

	reply.Ok = true
	return nil
}
