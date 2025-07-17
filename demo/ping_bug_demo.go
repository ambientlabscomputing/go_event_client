package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("=== PING INTERVAL BUG DEMONSTRATION ===")
	
	pingInterval := 30 // seconds (typical value)
	
	fmt.Printf("PingInterval setting: %d (intended as seconds)\n\n", pingInterval)
	
	// BROKEN: What the code was doing
	brokenDuration := time.Duration(pingInterval)
	fmt.Printf("BROKEN - ticker := time.NewTicker(time.Duration(%d))\n", pingInterval)
	fmt.Printf("  Result: %v (%d nanoseconds)\n", brokenDuration, brokenDuration.Nanoseconds())
	fmt.Printf("  Frequency: %.2f times per second\n", float64(time.Second)/float64(brokenDuration))
	fmt.Printf("  Effect: Sends pings every %.0f nanoseconds = CONSTANTLY!\n\n", float64(brokenDuration))
	
	// FIXED: What it should be doing
	fixedDuration := time.Duration(pingInterval) * time.Second
	fmt.Printf("FIXED - ticker := time.NewTicker(time.Duration(%d) * time.Second)\n", pingInterval)
	fmt.Printf("  Result: %v (%d nanoseconds)\n", fixedDuration, fixedDuration.Nanoseconds())
	fmt.Printf("  Frequency: %.4f times per second\n", float64(time.Second)/float64(fixedDuration))
	fmt.Printf("  Effect: Sends pings every %d seconds = NORMAL\n\n", pingInterval)
	
	fmt.Println("=== IMPACT ===")
	fmt.Printf("The broken version was sending pings %.0f times more frequently than intended!\n", 
		float64(fixedDuration)/float64(brokenDuration))
	fmt.Println("This caused the server to respond with constant pong messages,")
	fmt.Println("overwhelming the client with 'READPUMP: Received pong' logs.")
}
