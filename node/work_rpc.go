package node

import (
	"errors"
	"fmt"
	sharedrpc "workdistributor/rpc"
)

// RequestWork is called by a Worker over the network to ask the Leader for a job
func (n *Node) RequestWork(args *sharedrpc.WorkRequest, reply *sharedrpc.WorkReply) error {
	// Lock the node so multiple workers don't mess up the lists at the same time
	n.mu.Lock()
	defer n.mu.Unlock()

	// If this node isn't the leader, it shouldn't be giving out work
	if !n.IsLeader {
		return errors.New("I am not the leader")
	}

	// Check if we have any jobs left to give
	if len(n.PendingJobs) > 0 {
		// Take the first job off the pending list
		job := n.PendingJobs[0]
		n.PendingJobs = n.PendingJobs[1:]

		// Update the job's status and track who is working on it
		job.Status = "in-progress"
		job.WorkerID = args.WorkerID
		n.InProgressJobs[job.JobID] = job

		// Send it back in the reply
		reply.Job = job
		reply.Valid = true
		fmt.Printf("[Leader] Assigned Job %d to Worker %d\n", job.JobID, args.WorkerID)
	} else {
		// No jobs left!
		reply.Valid = false
	}

	return nil
}

// SubmitWork is called by a Worker to give the computed result back to the Leader
func (n *Node) SubmitWork(args *sharedrpc.SubmitRequest, reply *sharedrpc.SubmitReply) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.IsLeader {
		return errors.New("I am not the leader")
	}

	job := args.Job
	job.Status = "completed"

	// Remove it from in-progress and add to completed
	delete(n.InProgressJobs, job.JobID)
	n.CompletedJobs = append(n.CompletedJobs, job)

	reply.Success = true
	fmt.Printf("[Leader] Worker %d completed Job %d (Result: %d)\n", args.WorkerID, job.JobID, job.Result)

	return nil
}
