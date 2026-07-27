package transform

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/brexhq/substation/v2/config"
	"github.com/brexhq/substation/v2/internal/aggregate"
	"github.com/brexhq/substation/v2/message"
)

func newAggregateToArray(_ context.Context, cfg config.Config) (*aggregateToArray, error) {
	conf := aggregateArrayConfig{}
	if err := conf.Decode(cfg.Settings); err != nil {
		return nil, fmt.Errorf("transform aggregate_to_array: %v", err)
	}

	if conf.ID == "" {
		conf.ID = "aggregate_to_array"
	}

	tf := aggregateToArray{
		conf:      conf,
		hasObjTrg: conf.Object.TargetKey != "",
	}

	agg, err := aggregate.New(aggregate.Config{
		Count:    conf.Batch.Count,
		Size:     conf.Batch.Size,
		Duration: conf.Batch.Duration,
	})
	if err != nil {
		return nil, fmt.Errorf("transform %s: %v", conf.ID, err)
	}
	tf.agg = *agg

	return &tf, nil
}

type aggregateToArray struct {
	conf      aggregateArrayConfig
	hasObjTrg bool

	mu  sync.Mutex
	agg aggregate.Aggregate
}

func (tf *aggregateToArray) Transform(ctx context.Context, msg *message.Message) ([]*message.Message, error) {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	if msg.IsControl() {
		var output []*message.Message

		for _, items := range tf.agg.GetAll() {
			outMsg, err := tf.newArrayMessage(items.Get())
			if err != nil {
				return nil, err
			}

			if outMsg == nil {
				continue
			}

			output = append(output, outMsg)
		}

		tf.agg.ResetAll()

		output = append(output, msg)
		return output, nil
	}

	key := msg.GetValue(tf.conf.Object.BatchKey).String()
	if ok := tf.agg.Add(key, msg.Data()); ok {
		return nil, nil
	}

	// Add can also fail because the aggregate has been idle for longer than the
	// configured duration, in which case the aggregate is still empty and
	// newArrayMessage returns nil rather than an empty payload.
	outMsg, err := tf.newArrayMessage(tf.agg.Get(key))
	if err != nil {
		return nil, err
	}

	// If data cannot be added after reset, then the batch is misconfgured.
	tf.agg.Reset(key)
	if ok := tf.agg.Add(key, msg.Data()); !ok {
		return nil, fmt.Errorf("transform %s: %v", tf.conf.ID, errBatchNoMoreData)
	}

	if outMsg == nil {
		return nil, nil
	}

	return []*message.Message{outMsg}, nil
}

// newArrayMessage builds a message holding the aggregated array, or nil when the
// aggregate is empty. An empty payload is not forwarded because sinks cannot
// distinguish it from real data; send_http_post, for example, would POST an empty
// body.
func (tf *aggregateToArray) newArrayMessage(items [][]byte) (*message.Message, error) {
	array := aggToArray(items)
	if len(array) == 0 {
		return nil, nil
	}

	outMsg := message.New()
	if tf.hasObjTrg {
		if err := outMsg.SetValue(tf.conf.Object.TargetKey, array); err != nil {
			return nil, fmt.Errorf("transform %s: %v", tf.conf.ID, err)
		}

		return outMsg, nil
	}

	outMsg.SetData(array)

	return outMsg, nil
}

func (tf *aggregateToArray) String() string {
	b, _ := json.Marshal(tf.conf)
	return string(b)
}
