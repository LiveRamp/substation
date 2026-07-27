package transform

import (
	"bytes"
	"fmt"
	"slices"

	iconfig "github.com/brexhq/substation/v2/internal/config"
)

type aggregateArrayConfig struct {
	ID     string         `json:"id"`
	Object iconfig.Object `json:"object"`
	Batch  iconfig.Batch  `json:"batch"`
}

func (c *aggregateArrayConfig) Decode(in interface{}) error {
	return iconfig.Decode(in, c)
}

func aggToArray(data [][]byte) []byte {
	// Blank elements are excluded because joining them produces a malformed
	// array, e.g. `[{"a":1},]`, which downstream consumers reject. Splitting on a
	// separator yields blank elements for trailing or repeated separators, so
	// callers can legitimately pass them.
	items := make([][]byte, 0, len(data))

	for _, d := range data {
		if len(bytes.TrimSpace(d)) == 0 {
			continue
		}

		items = append(items, d)
	}

	if len(items) == 0 {
		return nil
	}

	return slices.Concat([]byte("["), bytes.Join(items, []byte(",")), []byte("]"))
}

type aggregateStrConfig struct {
	// Separator is the string that is used to join and split data.
	Separator string `json:"separator"`

	ID     string         `json:"id"`
	Object iconfig.Object `json:"object"`
	Batch  iconfig.Batch  `json:"batch"`
}

func (c *aggregateStrConfig) Decode(in interface{}) error {
	return iconfig.Decode(in, c)
}

func (c *aggregateStrConfig) Validate() error {
	if c.Separator == "" {
		return fmt.Errorf("separator: %v", iconfig.ErrMissingRequiredOption)
	}

	return nil
}

func aggToStr(data [][]byte, separator []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	return bytes.Join(data, separator)
}

func aggFromStr(data []byte, separator []byte) [][]byte {
	return bytes.Split(data, separator)
}
