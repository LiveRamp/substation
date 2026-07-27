package transform

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brexhq/substation/v2/config"
	"github.com/brexhq/substation/v2/message"
)

// TestAggregateToArrayStaleBatch probes aggregate.Add's duration check. a.now
// is refreshed on every successful Add and on Reset, so once an aggregate has sat
// idle for longer than the configured duration the next Add returns false. When
// that happens on the first item of a fresh batch, aggregate_to_array flushes an
// aggregate that is still empty.
func TestAggregateToArrayStaleBatch(t *testing.T) {
	ctx := context.TODO()

	// Short duration so the idle window is testable.
	toArr, err := newAggregateToArray(ctx, config.Config{
		Settings: map[string]interface{}{
			"batch": map[string]interface{}{"duration": "50ms", "count": 1000, "size": 1000000},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// First batch: one item, flushed by a control message.
	if _, err := toArr.Transform(ctx, message.New().SetData([]byte(`{"a":1}`))); err != nil {
		t.Fatal(err)
	}
	if _, err := toArr.Transform(ctx, message.New().AsControl()); err != nil {
		t.Fatal(err)
	}

	// Idle longer than the batch duration, as an idle Cloud Run instance would.
	time.Sleep(120 * time.Millisecond)

	// Next item after the idle window.
	out, err := toArr.Transform(ctx, message.New().SetData([]byte(`{"b":2}`)))
	if err != nil {
		t.Fatal(err)
	}

	for _, m := range out {
		if m.IsControl() {
			continue
		}

		d := m.Data()
		t.Logf("emitted payload: %q (len=%d)", string(d), len(d))

		if len(d) == 0 {
			t.Errorf("emitted an empty payload; sinks cannot distinguish this from real data")
			continue
		}

		if !json.Valid(d) {
			t.Errorf("  -> payload is not valid JSON: %q", string(d))
		}
	}

	// Emitting nothing is the correct outcome: the stale aggregate was empty.
	if len(out) != 0 {
		t.Logf("emitted %d message(s) on the stale add", len(out))
	}
}
