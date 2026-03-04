package common

// NodeAddresses maps a unique Node ID to its IP address and port.
// For now, we use localhost for testing on one machine.
// When you move to multiple computers, you will change these to actual LAN IPs (e.g., "192.168.1.10:8001").
var NodeAddresses = map[int]string{
	1: "localhost:8001",
	2: "localhost:8002",
	3: "localhost:8003",
}
