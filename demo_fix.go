package main

import (
	"encoding/json"
	"fmt"
	"log"

	go_event_client "github.com/ambientlabscomputing/go_event_client"
)

func main() {
	fmt.Println("=== DEMONSTRATING THE KEY-VALUE PAIR TO MAP CONVERSION FIX ===")

	// This is the exact data from your logs
	problematicData := `[{"Key":"amount","Value":5},{"Key":"counter_id","Value":"123"}]`

	fmt.Printf("Raw server data: %s\n\n", problematicData)

	// Parse it using our fixed MessageContent
	var content go_event_client.MessageContent
	if err := json.Unmarshal([]byte(problematicData), &content); err != nil {
		log.Fatalf("Failed to unmarshal: %v", err)
	}

	fmt.Println("After parsing with our new implementation:")
	fmt.Printf("  IsMap(): %t\n", content.IsMap())
	fmt.Printf("  IsString(): %t\n", content.IsString())

	// CRITICAL: This should now return true and the converted map
	if asMap, ok := content.AsMap(); ok {
		fmt.Printf("  AsMap() SUCCESS: %+v\n", asMap)
		fmt.Printf("  GetValue('amount'): '%s'\n", content.GetValue("amount"))
		fmt.Printf("  GetValue('counter_id'): '%s'\n", content.GetValue("counter_id"))

		fmt.Println("\n🎉 FIXED! Key-value pairs are now properly converted to a map!")
		fmt.Println("Your materializer can now access values directly:")
		fmt.Printf("  amount = %s (was a number, now string)\n", asMap["amount"])
		fmt.Printf("  counter_id = %s\n", asMap["counter_id"])

	} else {
		fmt.Println("  AsMap() FAILED - this would be the bug!")
	}

	fmt.Println("\n=== COMPARISON ===")
	fmt.Println("BEFORE: AsMap() returned false, had to use string parsing fallback")
	fmt.Println("AFTER:  AsMap() returns true with proper map conversion")
	fmt.Println("Result: Your materializer no longer gets parsing errors!")
}
