package common

// WorkUnit represents a single job in our Folding@Home-like system
type WorkUnit struct {
	JobID    int
	Data     int    // The dummy data to process (e.g., a number to factorize)
	Result   int    // The computed answer
	Status   string // "pending", "in-progress", "completed"
	WorkerID int    // Which node is currently working on it
}
