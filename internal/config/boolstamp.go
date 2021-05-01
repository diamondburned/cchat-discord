package config

import (
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type boolStamp struct {
	stamp time.Duration
	value bool
}

var _ customType = (*boolStamp)(nil)

func (bs *boolStamp) Marshal() string {
	if bs.stamp > 0 {
		return bs.stamp.String()
	}

	return strconv.FormatBool(bs.value)
}

func (bs *boolStamp) Unmarshal(v string) error {
	t, err := time.ParseDuration(v)
	if err == nil && t > 0 {
		bs.stamp = t
		return nil
	}

	b, err := strconv.ParseBool(v)
	if err == nil {
		bs.value = b
		return nil
	}

	return errors.New("invalid bool or timestamp")
}
