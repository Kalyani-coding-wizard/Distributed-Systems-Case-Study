package node

import (
	"sync"
	"workdistributor/common"
	sharedrpc "workdistributor/rpc"
)

type Node struct {
	ID       int
	Address  string
	IsLeader bool
	LeaderID int

	mu sync.Mutex

	// Work Distributor State
	PendingJobs    []common.WorkUnit
	InProgressJobs map[int]common.WorkUnit
	CompletedJobs  []common.WorkUnit

	// Ricart-Agrawala State
	LamportClock    int
	MEState         string
	ReplyCount      int
	DeferredReplies []int

	// ==========================================
	// Chandy-Lamport Snapshot State
	// ==========================================
	CurrentJobProcessing int                              // ID of the job currently being worked on (0 if idle)
	SeenSnapshots        map[int]bool                     // Tracks which Snapshot IDs we've already processed
	GlobalSnapshot       map[int]sharedrpc.NodeLocalState // Initiator stores the collected states here
}

func NewNode(id int, address string) *Node {
	return &Node{
		ID:       id,
		Address:  address,
		IsLeader: false,
		LeaderID: -1,

		PendingJobs:    make([]common.WorkUnit, 0),
		InProgressJobs: make(map[int]common.WorkUnit),
		CompletedJobs:  make([]common.WorkUnit, 0),

		LamportClock:    0,
		MEState:         "RELEASED",
		ReplyCount:      0,
		DeferredReplies: make([]int, 0),

		// Initialize Snapshot variables
		CurrentJobProcessing: 0,
		SeenSnapshots:        make(map[int]bool),
		GlobalSnapshot:       make(map[int]sharedrpc.NodeLocalState),
	}
}
