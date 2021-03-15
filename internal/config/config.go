package config

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/diamondburned/cchat"
	"github.com/pkg/errors"
)

type customType interface {
	Marshal() string
	Unmarshal(string) error
}

type config struct {
	Name  string
	Value interface{}
}

func (c config) Marshal(dst map[string]string) error {
	switch v := c.Value.(type) {
	case bool:
		dst[c.Name] = strconv.FormatBool(v)
	case string:
		dst[c.Name] = v
	default:
		return cchat.ErrInvalidConfigAtField{
			Key: c.Name,
			Err: fmt.Errorf("unknown type %T", c.Value),
		}
	}

	return nil
}

func (c *config) Unmarshal(src map[string]string) (err error) {
	strVal, ok := src[c.Name]
	if !ok {
		return cchat.ErrInvalidConfigAtField{
			Key: c.Name, Err: errors.New("missing field"),
		}
	}

	switch v := c.Value.(type) {
	case bool:
		c.Value, err = strconv.ParseBool(strVal)
	case string:
		c.Value = strVal
	case customType:
		err = v.Unmarshal(strVal)
		c.Value = v
	default:
		err = fmt.Errorf("unknown type %T", c.Value)
	}

	if err != nil {
		return cchat.ErrInvalidConfigAtField{
			Key: c.Name,
			Err: err,
		}
	}

	return nil
}

type registry struct {
	mutex   sync.RWMutex
	configs []config
}

func (reg *registry) get(i int) interface{} {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()

	return reg.configs[i].Value
}

func (reg *registry) Configuration() (map[string]string, error) {
	reg.mutex.RLock()
	defer reg.mutex.RUnlock()

	var configMap = map[string]string{}

	for _, config := range reg.configs {
		if err := config.Marshal(configMap); err != nil {
			return nil, err
		}
	}

	return configMap, nil
}

func (reg *registry) SetConfiguration(cfgMap map[string]string) error {
	reg.mutex.Lock()
	defer reg.mutex.Unlock()

	for i := range reg.configs {
		// reference the config inside the slice
		if err := reg.configs[i].Unmarshal(cfgMap); err != nil {
			return err
		}
	}

	return nil
}
