package go_event_client

import (
	"encoding/json"
	"testing"
)

// TestKeyValuePairContentParsing tests the key-value pair to map conversion functionality
func TestKeyValuePairContentParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantMap  bool
		expected map[string]string
	}{
		{
			name:    "key-value pairs with mixed types",
			input:   `[{"Key":"amount","Value":5},{"Key":"counter_id","Value":"123"}]`,
			wantMap: true,
			expected: map[string]string{
				"amount":     "5",
				"counter_id": "123",
			},
		},
		{
			name:    "key-value pairs with string values",
			input:   `[{"Key":"name","Value":"John"},{"Key":"email","Value":"john@example.com"}]`,
			wantMap: true,
			expected: map[string]string{
				"name":  "John",
				"email": "john@example.com",
			},
		},
		{
			name:    "key-value pairs with boolean values",
			input:   `[{"Key":"active","Value":true},{"Key":"verified","Value":false}]`,
			wantMap: true,
			expected: map[string]string{
				"active":   "true",
				"verified": "false",
			},
		},
		{
			name:     "empty key-value array",
			input:    `[]`,
			wantMap:  true,
			expected: map[string]string{},
		},
		{
			name:     "simple string content",
			input:    `"hello world"`,
			wantMap:  false,
			expected: nil,
		},
		{
			name:     "JSON object content",
			input:    `{"message": "hello"}`,
			wantMap:  false,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var content MessageContent
			if err := json.Unmarshal([]byte(tt.input), &content); err != nil {
				t.Fatalf("Failed to unmarshal %s: %v", tt.input, err)
			}

			if got := content.IsMap(); got != tt.wantMap {
				t.Errorf("IsMap() = %v, want %v", got, tt.wantMap)
			}

			if tt.wantMap {
				asMap, ok := content.AsMap()
				if !ok {
					t.Errorf("AsMap() failed, expected success")
					return
				}

				for key, expectedValue := range tt.expected {
					if got := asMap[key]; got != expectedValue {
						t.Errorf("AsMap()[%s] = %s, want %s", key, got, expectedValue)
					}
					if got := content.GetValue(key); got != expectedValue {
						t.Errorf("GetValue(%s) = %s, want %s", key, got, expectedValue)
					}
				}

				// Verify all keys are present
				if len(asMap) != len(tt.expected) {
					t.Errorf("AsMap() has %d keys, want %d", len(asMap), len(tt.expected))
				}
			} else {
				if asMap, ok := content.AsMap(); ok {
					t.Errorf("AsMap() should have failed but returned: %v", asMap)
				}
			}
		})
	}
}
