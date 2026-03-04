package rpc

import "workdistributor/common"

// ==========================================
// Work Distribution Messages
// ==========================================

// Worker asks the Leader for a job
type WorkRequest struct {
	WorkerID int
}

// Leader sends a job back to the Worker
type WorkReply struct {
	Job   common.WorkUnit
	Valid bool // True if a job is assigned, False if no jobs are left
}

// Worker submits a completed job back to the Leader
type SubmitRequest struct {
	WorkerID int
	Job      common.WorkUnit
}

// Leader acknowledges the submission
type SubmitReply struct {
	Success bool
}

// ==========================================
// Bully Election Messages
// ==========================================

// A node announces an election or declares victory
type ElectionMsg struct {
	SenderID int
}

// A higher-ID node responds to stop a lower-ID node's election
type ElectionReply struct {
	Ok bool
}

// The winner announces they are the new leader
type CoordinatorMsg struct {
	LeaderID int
}

// Other nodes acknowledge the new leader
type CoordinatorReply struct {
	Ok bool
}

// ==========================================
// Ricart-Agrawala Mutual Exclusion Messages
// ==========================================

// A node asks for permission to enter the Critical Section
type RARequest struct {
	NodeID    int
	Timestamp int
}

// A node grants permission
type RAReply struct {
	Ok bool
}

// ==========================================
// Chandy-Lamport Snapshot Messages
// ==========================================

// NodeLocalState holds the snapshot of a single node at a specific point in time
type NodeLocalState struct {
	NodeID          int
	IsLeader        bool
	PendingCount    int
	InProgressCount int
	CompletedCount  int
	WorkerStatus    string // e.g., "Idle" or "Processing Job X"
}

// MarkerMsg is sent to trigger nodes to take a local snapshot
type MarkerMsg struct {
	InitiatorID int
	SnapshotID  int
}

type MarkerReply struct {
	Ok bool
}

// StateSubmission is used by nodes to send their recorded state back to the initiator
type StateSubmission struct {
	SnapshotID int
	State      NodeLocalState
}

type StateSubmissionReply struct {
	Ok bool
}
