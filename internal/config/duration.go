package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Duration time.Duration

func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	if d == nil {
		return nil
	}
	if len(data) == 0 || string(data) == "null" {
		*d = 0
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		s = strings.TrimSpace(s)
		if s == "" {
			*d = 0
			return nil
		}
		if parsed, err := time.ParseDuration(s); err == nil {
			*d = Duration(parsed)
			return nil
		}
		if secs, err := strconv.ParseInt(s, 10, 64); err == nil {
			*d = Duration(time.Duration(secs) * time.Second)
			return nil
		}
		return fmt.Errorf("invalid duration %q", s)
	}

	var secs int64
	if err := json.Unmarshal(data, &secs); err == nil {
		*d = Duration(time.Duration(secs) * time.Second)
		return nil
	}

	return fmt.Errorf("invalid duration %q", string(data))
}

func (d Duration) MarshalYAML() (any, error) {
	return time.Duration(d).String(), nil
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	if value == nil {
		return nil
	}
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("duration must be scalar")
	}
	if value.Value == "" {
		*d = 0
		return nil
	}
	if parsed, err := time.ParseDuration(value.Value); err == nil {
		*d = Duration(parsed)
		return nil
	}
	if secs, err := strconv.ParseInt(value.Value, 10, 64); err == nil {
		*d = Duration(time.Duration(secs) * time.Second)
		return nil
	}
	return fmt.Errorf("invalid duration %q", value.Value)
}
