package transform

import (
	"encoding/json"
	"testing"
)

var aggToArrayTests = []struct {
	name     string
	data     []string
	expected string
}{
	{
		"objects",
		[]string{`{"a":1}`, `{"b":2}`},
		`[{"a":1},{"b":2}]`,
	},
	{
		// A trailing separator in the source yields a blank element. Joining it
		// verbatim produces `[{"a":1},{"b":2},]`, which is not valid JSON.
		"trailing blank element",
		[]string{`{"a":1}`, `{"b":2}`, ``},
		`[{"a":1},{"b":2}]`,
	},
	{
		"interior blank element",
		[]string{`{"a":1}`, ``, `{"b":2}`},
		`[{"a":1},{"b":2}]`,
	},
	{
		// Splitting CRLF data on a newline leaves a carriage return behind.
		"whitespace-only element",
		[]string{`{"a":1}`, "\r"},
		`[{"a":1}]`,
	},
	{
		"only blank elements",
		[]string{``, ``},
		``,
	},
}

func TestAggToArray(t *testing.T) {
	for _, test := range aggToArrayTests {
		t.Run(test.name, func(t *testing.T) {
			var data [][]byte
			for _, d := range test.data {
				data = append(data, []byte(d))
			}

			result := aggToArray(data)

			if string(result) != test.expected {
				t.Errorf("expected %q, got %q", test.expected, string(result))
			}

			// Any non-empty result must be valid JSON; that is the point of the test.
			if len(result) > 0 && !json.Valid(result) {
				t.Errorf("result is not valid JSON: %q", string(result))
			}
		})
	}
}

func TestAggregateArrayConfigDecode(t *testing.T) {
	config := &aggregateArrayConfig{}
	err := config.Decode(map[string]interface{}{
		"id": "test_id",
		"object": map[string]interface{}{
			"source_key": "foo",
		},
		"batch": map[string]interface{}{
			"size": 10,
		},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if config.ID != "test_id" {
		t.Errorf("expected id to be 'test_id', got %s", config.ID)
	}
	if config.Object.SourceKey != "foo" {
		t.Errorf("expected source_key to be 'foo', got %s", config.Object.SourceKey)
	}
	if config.Batch.Size != 10 {
		t.Errorf("expected batch size to be 10, got %d", config.Batch.Size)
	}
}

func TestAggregateStrConfigDecode(t *testing.T) {
	config := &aggregateStrConfig{}
	err := config.Decode(map[string]interface{}{
		"id": "test_id",
		"object": map[string]interface{}{
			"source_key": "foo",
		},
		"batch": map[string]interface{}{
			"size": 10,
		},
		"separator": "\n",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if config.ID != "test_id" {
		t.Errorf("expected id to be 'test_id', got %s", config.ID)
	}
	if config.Object.SourceKey != "foo" {
		t.Errorf("expected source_key to be 'foo', got %s", config.Object.SourceKey)
	}
	if config.Batch.Size != 10 {
		t.Errorf("expected batch size to be 10, got %d", config.Batch.Size)
	}
	if config.Separator != "\n" {
		t.Errorf("expected separator to be '\\n', got %s", config.Separator)
	}
}

func TestAggregateStrConfigValidate(t *testing.T) {
	config := &aggregateStrConfig{
		Separator: "\n",
	}
	err := config.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test with missing separator
	config = &aggregateStrConfig{}
	err = config.Validate()
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if err.Error() != "separator: missing required option" {
		t.Errorf("expected error 'separator: missing required option', got %v", err)
	}
}

func FuzzTestAggregateArrayConfigDecode(f *testing.F) {
	testcases := []string{
		`{"id":"test_id","object":{"source_key":"foo"},"batch":{"size":10}}`,
		`{"id":"test_id","object":{"source_key":"bar"},"batch":{"size":5}}`,
		`{"id":"test_id","object":{"source_key":"baz"},"batch":{"size":0}}`,
	}

	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, data string) {
		config := &aggregateArrayConfig{}
		err := config.Decode([]byte(data))
		if err != nil {
			return
		}
	})
}

func FuzzTestAggToArray(f *testing.F) {
	testcases := [][]byte{
		[]byte(`{"a":"b"}`),
		[]byte(`{"c":"d"}`),
		[]byte(`{"e":"f"}`),
		[]byte(`{"a":"b"}\n{"c":"d"}\n{"e":"f"}`),
		[]byte(`{"a":"b"}\n{"c":"d"}`),
		[]byte(`{"a":"b"}`),
		[]byte(`{"a":"b"}\n`),
		[]byte(``),
	}

	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_ = aggToArray([][]byte{data})
	})
}

func FuzzTestAggregateStrConfigDecode(f *testing.F) {
	testcases := []string{
		`{"id":"test_id","object":{"source_key":"foo"},"batch":{"size":10},"separator":"\n"}`,
		`{"id":"test_id","object":{"source_key":"bar"},"batch":{"size":5},"separator":"\t"}`,
		`{"id":"test_id","object":{"source_key":"baz"},"batch":{"size":0},"separator":""}`,
	}

	for _, tc := range testcases {
		f.Add(tc)
	}

	f.Fuzz(func(t *testing.T, data string) {
		config := &aggregateStrConfig{}
		err := config.Decode([]byte(data))
		if err != nil {
			return
		}
	})
}
