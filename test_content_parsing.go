package go_event_client

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestContentParsing tests the content parsing functionality
func TestContentParsing(t *testing.T) {
	// Test data that matches your logs
	testData := `[{"Key":"amount","Value":5},{"Key":"counter_id","Value":"123"}]`

	fmt.Printf("Testing content parsing with data: %s\n", testData)

	var content MessageContent
	if err := json.Unmarshal([]byte(testData), &content); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	fmt.Printf("Content parsed successfully!\n")
	fmt.Printf("IsMap(): %t\n", content.IsMap())
	fmt.Printf("IsString(): %t\n", content.IsString())

	if asMap, ok := content.AsMap(); ok {
		fmt.Printf("AsMap() returned: %+v\n", asMap)
		fmt.Printf("amount = %s\n", content.GetValue("amount"))
		fmt.Printf("counter_id = %s\n", content.GetValue("counter_id"))

		// Verify the values
		if content.GetValue("amount") != "5" {
			t.Errorf("Expected amount=5, got %s", content.GetValue("amount"))
		}
		if content.GetValue("counter_id") != "123" {
			t.Errorf("Expected counter_id=123, got %s", content.GetValue("counter_id"))
		}
	} else {
		t.Errorf("AsMap() failed - this is the bug!")
		fmt.Printf("String representation: %s\n", content.String())
	}
}
