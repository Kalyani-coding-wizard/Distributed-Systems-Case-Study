package node

import (
	"fmt"
	"net/rpc"
	"time"
	"workdistributor/common"
	sharedrpc "workdistributor/rpc"
)

// GenerateDummyJobs is used by the Leader to fill its pending queue
func (n *Node) GenerateDummyJobs(count int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for i := 1; i <= count; i++ {
		job := common.WorkUnit{
			JobID:  i,
			Data:   i * 10, // The dummy data to process (e.g., 10, 20, 30)
			Status: "pending",
		}
		n.PendingJobs = append(n.PendingJobs, job)
	}
	fmt.Printf("[Leader] Generated %d dummy jobs ready for processing.\n", count)
}

// StartWorkerLoop runs continuously, acting as the client asking for work
func (n *Node) StartWorkerLoop() {
	fmt.Printf("Worker %d loop started. Waiting for a leader...\n", n.ID)

	for {
		time.Sleep(2 * time.Second) // Pause between jobs (and wait for election)

		// Safely check who the current leader is
		n.mu.Lock()
		leaderID := n.LeaderID
		isLeader := n.IsLeader
		n.mu.Unlock()

		// If no leader is elected yet, or if WE are the leader, skip this loop
		if leaderID == -1 || isLeader {
			continue
		}

		leaderAddr := common.NodeAddresses[leaderID]

		// Dial connects to the RPC server over TCP
		client, err := rpc.Dial("tcp", leaderAddr)
		if err != nil {
			// If leader is down, trigger a new election!
			fmt.Printf("Worker %d: Leader %d seems dead! Starting new election...\n", n.ID, leaderID)

			n.mu.Lock()
			n.LeaderID = -1 // Reset leader
			n.mu.Unlock()

			go n.StartElection()
			continue
		}

		// 1. Ask for Work
		args := &sharedrpc.WorkRequest{WorkerID: n.ID}
		reply := &sharedrpc.WorkReply{}
		err = client.Call("Node.RequestWork", args, reply)

		if err != nil || !reply.Valid {
			client.Close()
			continue // No jobs or error, try again next loop
		}

		// 2. Process the Work
		job := reply.Job
		fmt.Printf("Worker %d got Job %d (Data: %d). Processing...\n", n.ID, job.JobID, job.Data)

		// Update state for the snapshot
		n.mu.Lock()
		n.CurrentJobProcessing = job.JobID
		n.mu.Unlock()

		time.Sleep(3 * time.Second) // Increased to 3 seconds so the snapshot has time to catch it!
		job.Result = job.Data * 2   // Our "complex" task

		// Clear state after processing
		n.mu.Lock()
		n.CurrentJobProcessing = 0
		n.mu.Unlock()

		// ========================================================
		// MUTUAL EXCLUSION: Request Critical Section BEFORE submitting
		// ========================================================
		fmt.Printf("Worker %d wants to submit Job %d. Requesting Critical Section...\n", n.ID, job.JobID)
		n.EnterCriticalSection()

		// 3. Submit the Result (We are now in the Critical Section!)
		submitArgs := &sharedrpc.SubmitRequest{WorkerID: n.ID, Job: job}
		submitReply := &sharedrpc.SubmitReply{}
		client.Call("Node.SubmitWork", submitArgs, submitReply)

		if submitReply.Success {
			fmt.Printf("Worker %d successfully submitted Job %d!\n", n.ID, job.JobID)
		}

		// ========================================================
		// MUTUAL EXCLUSION: Release Critical Section AFTER submitting
		// ========================================================
		n.ExitCriticalSection()

		client.Close()
	}
}
